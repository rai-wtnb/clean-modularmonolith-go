// Package eventbus provides event infrastructure for inter-module communication.
// For production, this would be replaced with Google Cloud Pub/Sub, RabbitMQ, Kafka,
// or a similar service by adopting the outbox pattern.
package eventbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type depthKey struct{}

// EventBus manages event subscriptions and publishing.
// It implements events.Subscriber, events.Publisher,
// events.PostCommitSubscriber, and events.PostCommitPublisher.
//
// A single instance is shared across the application. Event processing
// depth is tracked per-context, so concurrent transactions are isolated.
type EventBus struct {
	mu                 sync.RWMutex
	handlers           map[events.EventType][]events.Handler // pre-commit handlers
	postCommitHandlers map[events.EventType][]events.Handler // post-commit handlers
	logger             *slog.Logger
	maxDepth           int // max depth of event processing.
}

var (
	_ events.Subscriber           = (*EventBus)(nil)
	_ events.Publisher            = (*EventBus)(nil)
	_ events.PostCommitSubscriber = (*EventBus)(nil)
	_ events.PostCommitPublisher  = (*EventBus)(nil)

	tracer = otel.Tracer("eventbus")
)

// NewEventBus creates a new event bus.
func NewEventBus(logger *slog.Logger) *EventBus {
	return &EventBus{
		handlers:           make(map[events.EventType][]events.Handler),
		postCommitHandlers: make(map[events.EventType][]events.Handler),
		logger:             logger,
		maxDepth:           10,
	}
}

// LogSubscriptions logs all registered event subscriptions.
// Call this after all modules have been initialized to verify wiring.
func (b *EventBus) LogSubscriptions() {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for eventType, handlers := range b.handlers {
		for _, h := range handlers {
			b.logger.Info("event subscription", slog.String("event_type", eventType.String()), slog.String("handler", h.HandlerName()), slog.String("subdomain", h.Subdomain()), slog.String("phase", "pre-commit"))
		}
	}
	for eventType, handlers := range b.postCommitHandlers {
		for _, h := range handlers {
			b.logger.Info("event subscription", slog.String("event_type", eventType.String()), slog.String("handler", h.HandlerName()), slog.String("subdomain", h.Subdomain()), slog.String("phase", "post-commit"))
		}
	}
}

// Subscribe registers a pre-commit handler for an event type.
// Pre-commit handlers run inside the transaction boundary.
// Returns an error if a handler with the same name is already registered for the event type.
// Implements events.Subscriber.
func (b *EventBus) Subscribe(eventType events.EventType, handler events.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isDuplicate(b.handlers[eventType], handler) {
		return fmt.Errorf("duplicate handler %q for event %s (pre-commit)", handler.HandlerName(), eventType)
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.logger.Debug("subscribed to event", slog.String("event_type", eventType.String()), slog.String("phase", "pre-commit"))

	return nil
}

// SubscribePostCommit registers a post-commit handler for an event type.
// Post-commit handlers run after the transaction commits successfully.
// Returns an error if a handler with the same name is already registered for the event type.
// Implements events.PostCommitSubscriber.
func (b *EventBus) SubscribePostCommit(eventType events.EventType, handler events.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.isDuplicate(b.postCommitHandlers[eventType], handler) {
		return fmt.Errorf("duplicate handler %q for event %s (post-commit)", handler.HandlerName(), eventType)
	}

	b.postCommitHandlers[eventType] = append(b.postCommitHandlers[eventType], handler)
	b.logger.Debug("subscribed to event", slog.String("event_type", eventType.String()), slog.String("phase", "post-commit"))

	return nil
}

// Publish dispatches the given domain events to registered handlers synchronously.
// Implements events.Publisher.
func (b *EventBus) Publish(ctx context.Context, evts []events.Event) error {
	for event := range slices.Values(evts) {
		if err := b.processEvent(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

func (b *EventBus) processEvent(ctx context.Context, event events.Event) error {
	depth := b.depthFromContext(ctx)
	if depth >= b.maxDepth {
		return errors.New("event processing depth exceeded")
	}
	ctx = b.contextWithDepth(ctx, depth+1)

	handlers := b.handlersFor(event.EventType())
	for handler := range slices.Values(handlers) {
		ctx, span := tracer.Start(ctx,
			fmt.Sprintf("event.handle %s/%s", handler.Subdomain(), handler.HandlerName()),
			trace.WithAttributes(attribute.String("event.type", event.EventType().String()), attribute.String("event.id", event.EventID()), attribute.String("event.handler", handler.HandlerName()), attribute.String("event.subdomain", handler.Subdomain())),
		)

		if err := handler.Handle(ctx, event); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return fmt.Errorf("handler failed for event %s: %w", event.EventType().String(), err)
		}
		span.End()
	}
	return nil
}

// PublishPostCommit dispatches events to post-commit handlers asynchronously.
// Handlers run in a separate goroutine with a detached context so they are not
// cancelled when the caller's context (e.g. HTTP request) completes.
// Errors are logged but not propagated.
// Implements events.PostCommitPublisher.
func (b *EventBus) PublishPostCommit(ctx context.Context, evts []events.Event) {
	detachedCtx := detachContext(ctx)

	copied := make([]events.Event, len(evts))
	copy(copied, evts)

	go func() {
		for _, event := range copied {
			b.processPostCommitEvent(detachedCtx, event)
		}
	}()
}

func (b *EventBus) processPostCommitEvent(ctx context.Context, event events.Event) {
	handlers := b.postCommitHandlersFor(event.EventType())

	for handler := range slices.Values(handlers) {
		spanName := fmt.Sprintf("event.handle.post-commit %s/%s", handler.Subdomain(), handler.HandlerName())
		options := []trace.SpanStartOption{
			trace.WithAttributes(attribute.String("event.type", event.EventType().String()), attribute.String("event.id", event.EventID()), attribute.String("event.handler", handler.HandlerName()), attribute.String("event.subdomain", handler.Subdomain())),
		}
		ctx, span := tracer.Start(ctx, spanName, options...)

		err := handler.Handle(ctx, event)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			b.logger.Error("post-commit handler failed", slog.String("handler", handler.HandlerName()), slog.String("subdomain", handler.Subdomain()), slog.String("event_type", event.EventType().String()), slog.String("event_id", event.EventID()), slog.Any("error", err))
		}

		span.End()
	}
}

func (b *EventBus) postCommitHandlersFor(eventType events.EventType) []events.Handler {
	b.mu.RLock()
	defer b.mu.RUnlock()

	handlers := b.postCommitHandlers[eventType]
	result := make([]events.Handler, len(handlers))
	copy(result, handlers)
	return result
}

func (b *EventBus) handlersFor(eventType events.EventType) []events.Handler {
	b.mu.RLock()
	defer b.mu.RUnlock()

	handlers := b.handlers[eventType]
	result := make([]events.Handler, len(handlers))
	copy(result, handlers)
	return result
}

// detachContext creates a new context that carries trace span from the parent
// but is not subject to the parent's cancellation or deadline.
func detachContext(ctx context.Context) context.Context {
	return trace.ContextWithSpan(context.Background(), trace.SpanFromContext(ctx))
}

func (b *EventBus) isDuplicate(existing []events.Handler, handler events.Handler) bool {
	for _, h := range existing {
		if h.HandlerName() == handler.HandlerName() {
			return true
		}
	}
	return false
}

func (b *EventBus) depthFromContext(ctx context.Context) int {
	if depth, ok := ctx.Value(depthKey{}).(int); ok {
		return depth
	}
	return 0
}

func (b *EventBus) contextWithDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, depthKey{}, depth)
}
