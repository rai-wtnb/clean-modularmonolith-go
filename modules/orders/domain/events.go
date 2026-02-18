package domain

import (
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

const (
	OrderCreatedEventType   = "orders.OrderCreated"
	OrderSubmittedEventType = "orders.OrderSubmitted"
	OrderCancelledEventType = "orders.OrderCancelled"
)

// OrderCreatedEvent is published when a new order is created.
type OrderCreatedEvent struct {
	events.BaseEvent
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

func NewOrderCreatedEvent(order *Order) OrderCreatedEvent {
	return OrderCreatedEvent{
		BaseEvent: events.NewBaseEvent(OrderCreatedEventType, order.ID().String()),
		OrderID:   order.ID().String(),
		UserID:    order.UserID().String(),
	}
}

// OrderSubmittedEvent is published when an order is submitted.
type OrderSubmittedEvent struct {
	events.BaseEvent
	OrderID     string `json:"order_id"`
	UserID      string `json:"user_id"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}

func NewOrderSubmittedEvent(order *Order) OrderSubmittedEvent {
	return OrderSubmittedEvent{
		BaseEvent:   events.NewBaseEvent(OrderSubmittedEventType, order.ID().String()),
		OrderID:     order.ID().String(),
		UserID:      order.UserID().String(),
		TotalAmount: order.Total().Amount(),
		Currency:    order.Total().Currency(),
	}
}

// OrderCancelledEvent is published when an order is cancelled.
type OrderCancelledEvent struct {
	events.BaseEvent
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

func NewOrderCancelledEvent(order *Order) OrderCancelledEvent {
	return OrderCancelledEvent{
		BaseEvent: events.NewBaseEvent(OrderCancelledEventType, order.ID().String()),
		OrderID:   order.ID().String(),
		UserID:    order.UserID().String(),
	}
}
