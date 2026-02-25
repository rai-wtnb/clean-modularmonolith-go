// Package contracts defines public event contracts for inter-module communication.
// Modules should import event types from here, NOT from other module's domain packages.
package contracts

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

// Order module event types.
// These are the "public API" of the orders module for event-driven communication.
const (
	OrderCreatedEventType   events.EventType = "orders.OrderCreated"
	OrderSubmittedEventType events.EventType = "orders.OrderSubmitted"
	OrderCancelledEventType events.EventType = "orders.OrderCancelled"
)

// OrderSubmittedEvent is the public contract for order submission events.
// Other modules should use this type to handle order submissions.
type OrderSubmittedEvent struct {
	events.BaseEvent
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}
