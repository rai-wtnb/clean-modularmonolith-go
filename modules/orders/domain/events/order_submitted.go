package events

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

const OrderSubmittedEventType events.EventType = "orders.OrderSubmitted"

// OrderSubmittedEvent is published when an order is submitted.
// This is a public domain event — it may be imported by event handlers in other modules.
type OrderSubmittedEvent struct {
	events.BaseEvent
	OrderID     string
	UserID      string
	TotalAmount int64
	Currency    string
}
