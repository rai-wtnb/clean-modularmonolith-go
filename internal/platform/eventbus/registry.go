// Package eventbus provides event infrastructure for inter-module communication.
// For production, this would be replaced with Google Cloud Pub/Sub, RabbitMQ, Kafka, or a similar service by adopting the outbox pattern.
package eventbus

import (
	"log/slog"
	"sync"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// HandlerRegistry provides access to registered event handlers.
// This allows TransactionalPublisher to dispatch to handlers without
// managing subscriptions itself.
type HandlerRegistry interface {
	// HandlersFor returns all handlers registered for the given event type.
	HandlersFor(eventType events.EventType) []events.Handler
}

// EventHandlerRegistry manages event handler subscriptions.
// It implements both events.Subscriber (for registering handlers) and
// HandlerRegistry (for retrieving handlers by event type).
//
// This is the "registration" side of the event system. For publishing
// events within transactions, use TransactionalEventBus with this registry.
type EventHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[events.EventType][]events.Handler
	logger   *slog.Logger
}

// NewEventHandlerRegistry creates a new registry for event handlers.
func NewEventHandlerRegistry(logger *slog.Logger) *EventHandlerRegistry {
	return &EventHandlerRegistry{
		handlers: make(map[events.EventType][]events.Handler),
		logger:   logger,
	}
}

// Subscribe implements events.Subscriber.
// It should be called once per event type per module at initialization.
func (r *EventHandlerRegistry) Subscribe(eventType events.EventType, handler events.Handler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[eventType] = append(r.handlers[eventType], handler)
	r.logger.Debug("subscribed to event", slog.String("event_type", eventType.String()))

	return nil
}

// HandlersFor implements HandlerRegistry.
// Returns a copy of the handlers slice to avoid race conditions.
func (r *EventHandlerRegistry) HandlersFor(eventType events.EventType) []events.Handler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := r.handlers[eventType]
	result := make([]events.Handler, len(handlers))
	copy(result, handlers)
	return result
}

// Compile-time interface checks.
var (
	_ events.Subscriber = (*EventHandlerRegistry)(nil)
	_ HandlerRegistry   = (*EventHandlerRegistry)(nil)
)
