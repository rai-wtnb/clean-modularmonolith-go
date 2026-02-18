// Package eventbus provides an in-memory event bus for inter-module communication.
// For production, this would be replaced with Google Cloud Pub/Sub, RabbitMQ, Kafka, or a similar service by adopting the outbox pattern.
package eventbus

import (
	"context"
	"log/slog"
	"sync"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// InMemoryEventBus implements a simple synchronous event bus.
// Events are delivered synchronously in the same goroutine.
type InMemoryEventBus struct {
	mu       sync.RWMutex
	handlers map[string][]events.Handler
	logger   *slog.Logger
}

func New(logger *slog.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]events.Handler),
		logger:   logger,
	}
}

// Publish implements events.Publisher.
func (b *InMemoryEventBus) Publish(ctx context.Context, event events.Event) error {
	b.mu.RLock()
	handlers := b.handlers[event.EventType()]
	b.mu.RUnlock()

	b.logger.Debug("publishing event", slog.String("event_type", event.EventType()), slog.String("event_id", event.EventID()), slog.Int("handler_count", len(handlers)))

	// FIXME: They should be processed concurrently.
	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			b.logger.Error("event handler failed", slog.String("event_type", event.EventType()), slog.String("event_id", event.EventID()), slog.Any("error", err))
			// Continue processing other handlers even if one fails
		}
	}

	return nil
}

// Subscribe implements events.Subscriber.
func (b *InMemoryEventBus) Subscribe(eventType string, handler events.Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers[eventType] = append(b.handlers[eventType], handler)
	b.logger.Debug("subscribed to event", slog.String("event_type", eventType))

	return nil
}

// HandlerFunc is an adapter to use ordinary functions as event handlers.
type HandlerFunc func(ctx context.Context, event events.Event) error

func (f HandlerFunc) Handle(ctx context.Context, event events.Event) error {
	return f(ctx, event)
}
