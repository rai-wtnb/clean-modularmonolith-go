package eventbus

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

// HandlerRegistry provides access to registered event handlers.
// This allows TransactionalEventBus to dispatch to handlers without
// managing subscriptions itself.
type HandlerRegistry interface {
	// HandlersFor returns all handlers registered for the given event type.
	HandlersFor(eventType events.EventType) []events.Handler
}
