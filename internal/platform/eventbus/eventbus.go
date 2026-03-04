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
	"golang.org/x/sync/errgroup"
)

type depthKey struct{}

// EventBus manages event subscriptions and publishing.
// It implements both events.Subscriber and events.Publisher.
//
// A single instance is shared across the application. Event processing
// depth is tracked per-context, so concurrent transactions are isolated.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[events.EventType][]events.Handler // handlers are registered per DomainEvent:Subdomain.
	logger   *slog.Logger
	maxDepth int // max depth of event processing.
}

var (
	_ events.Subscriber = (*EventBus)(nil)
	_ events.Publisher  = (*EventBus)(nil)

	tracer = otel.Tracer("eventbus")
)

// NewEventBus creates a new event bus.
func NewEventBus(logger *slog.Logger) *EventBus {
	return &EventBus{
		handlers: make(map[events.EventType][]events.Handler),
		logger:   logger,
		maxDepth: 10,
	}
}

// LogSubscriptions logs all registered event subscriptions.
// Call this after all modules have been initialized to verify wiring.
func (b *EventBus) LogSubscriptions() {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for eventType, handlers := range b.handlers {
		for _, h := range handlers {
			b.logger.Info("event subscription", slog.String("event_type", eventType.String()), slog.String("handler", h.HandlerName()), slog.String("subdomain", h.Subdomain()))
		}
	}
}

// Subscribe registers a handler for an event type.
// Implements events.Subscriber.
func (b *EventBus) Subscribe(eventType events.EventType, handler events.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.logger.Debug("subscribed to event", slog.String("event_type", eventType.String()))

	return nil
}

// Publish extracts domain events from the context and dispatches them
// to registered handlers synchronously. Uses a drain loop: after processing
// the current batch, it checks for new events added by handlers and
// processes those too, repeating until no events remain.
// Implements events.Publisher.
func (b *EventBus) Publish(ctx context.Context) error {
	for {
		pending := events.Collect(ctx)
		if len(pending) == 0 {
			return nil
		}
		for event := range slices.Values(pending) {
			if err := b.processEvent(ctx, event); err != nil {
				return err
			}
		}
	}
}

func (b *EventBus) processEvent(ctx context.Context, event events.Event) error {
	depth := b.depthFromContext(ctx)
	if depth >= b.maxDepth {
		return errors.New("event processing depth exceeded")
	}
	ctx = b.contextWithDepth(ctx, depth+1)

	handlers := b.handlersFor(event.EventType())
	g, ctx := errgroup.WithContext(ctx)
	for handler := range slices.Values(handlers) {
		g.Go(func() error {
			ctx, span := tracer.Start(ctx,
				fmt.Sprintf("event.handle %s/%s", handler.Subdomain(), handler.HandlerName()),
				trace.WithAttributes(attribute.String("event.type", event.EventType().String()), attribute.String("event.id", event.EventID()), attribute.String("event.handler", handler.HandlerName()), attribute.String("event.subdomain", handler.Subdomain())),
			)
			defer span.End()

			if err := handler.Handle(ctx, event); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return err
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return fmt.Errorf("handler failed for event %s: %w", event.EventType().String(), err)
	}
	return nil
}

func (b *EventBus) handlersFor(eventType events.EventType) []events.Handler {
	b.mu.RLock()
	defer b.mu.RUnlock()

	handlers := b.handlers[eventType]
	result := make([]events.Handler, len(handlers))
	copy(result, handlers)
	return result
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
