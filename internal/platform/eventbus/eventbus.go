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
)

// NewEventBus creates a new event bus.
func NewEventBus(logger *slog.Logger) *EventBus {
	return &EventBus{
		handlers: make(map[events.EventType][]events.Handler),
		logger:   logger,
		maxDepth: 10,
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

// Publish dispatches events to registered handlers synchronously.
// Handlers execute within the current transaction context.
// Implements events.Publisher.
func (b *EventBus) Publish(ctx context.Context, domainEvents ...events.Event) error {
	for event := range slices.Values(domainEvents) {
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
	g, ctx := errgroup.WithContext(ctx)
	for handler := range slices.Values(handlers) {
		g.Go(func() error {
			return handler.Handle(ctx, event)
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
