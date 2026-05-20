package domain

import (
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// Domain events for the orders bounded context.
//
// Per ADR (domain-event placement), every DomainEvent type — including
// cross-module ones — is defined in the publishing subdomain's domain package.
// Handlers in other subdomains may import the event TYPE only; the event's
// constructor stays unexported so creation is confined to this subdomain.

const (
	OrderCreatedEventType   events.EventType = "orders.OrderCreated"
	OrderCancelledEventType events.EventType = "orders.OrderCancelled"
	OrderSubmittedEventType events.EventType = "orders.OrderSubmitted"
)

// OrderCreatedEvent is published when a new order is created.
type OrderCreatedEvent struct {
	events.BaseEvent
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

func newOrderCreatedEvent(order *Order) OrderCreatedEvent {
	return OrderCreatedEvent{
		BaseEvent: events.NewBaseEvent(OrderCreatedEventType),
		OrderID:   order.ID().String(),
		UserID:    order.UserRef().String(),
	}
}

// OrderCancelledEvent is published when an order is cancelled.
type OrderCancelledEvent struct {
	events.BaseEvent
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

func newOrderCancelledEvent(order *Order) OrderCancelledEvent {
	return OrderCancelledEvent{
		BaseEvent: events.NewBaseEvent(OrderCancelledEventType),
		OrderID:   order.ID().String(),
		UserID:    order.UserRef().String(),
	}
}

// OrderSubmittedEvent is published when an order is submitted.
// This is a public domain event — event handlers in other subdomains may
// import this type. Only the type may be referenced cross-module; the
// constructor is unexported so events are created only within this subdomain.
type OrderSubmittedEvent struct {
	events.BaseEvent
	OrderID     string
	UserID      string
	TotalAmount int64
	Currency    string
}

func newOrderSubmittedEvent(order *Order) OrderSubmittedEvent {
	return OrderSubmittedEvent{
		BaseEvent:   events.NewBaseEvent(OrderSubmittedEventType),
		OrderID:     order.ID().String(),
		UserID:      order.UserRef().String(),
		TotalAmount: order.Total().Amount(),
		Currency:    order.Total().Currency(),
	}
}
