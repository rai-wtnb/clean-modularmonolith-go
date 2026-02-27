// Package contracts defines public event contracts for inter-module communication.
// Modules should import event types from here, NOT from other module's domain packages.
package contracts

import "github.com/rai/clean-modularmonolith-go/modules/shared/events"

const (
	OrderSubmittedEventType events.EventType = "orders.OrderSubmitted"
)

type OrderSubmittedEvent struct {
	events.BaseEvent
	OrderID     string
	UserID      string
	TotalAmount int64
	Currency    string
}
