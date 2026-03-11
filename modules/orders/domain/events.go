package domain

import (
	orderevents "github.com/rai/clean-modularmonolith-go/modules/orders/domain/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// Internal event types (not used cross-module)
const (
	OrderCreatedEventType   events.EventType = "orders.OrderCreated"
	OrderCancelledEventType events.EventType = "orders.OrderCancelled"
	OrderSubmittedEventType                  = orderevents.OrderSubmittedEventType
)

// OrderCreatedEvent is published when a new order is created.
type OrderCreatedEvent struct {
	events.BaseEvent
	OrderID string `json:"order_id"`
	UserID  string `json:"user_id"`
}

func NewOrderCreatedEvent(order *Order) OrderCreatedEvent {
	return OrderCreatedEvent{
		BaseEvent: events.NewBaseEvent(OrderCreatedEventType),
		OrderID:   order.ID().String(),
		UserID:    order.UserRef().String(),
	}
}

func NewOrderSubmittedEvent(order *Order) orderevents.OrderSubmittedEvent {
	return orderevents.OrderSubmittedEvent{
		BaseEvent:   events.NewBaseEvent(OrderSubmittedEventType),
		OrderID:     order.ID().String(),
		UserID:      order.UserRef().String(),
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
		BaseEvent: events.NewBaseEvent(OrderCancelledEventType),
		OrderID:   order.ID().String(),
		UserID:    order.UserRef().String(),
	}
}
