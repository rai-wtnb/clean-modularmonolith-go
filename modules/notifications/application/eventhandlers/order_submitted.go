package eventhandlers

import (
	"context"
	"fmt"

	orderevents "github.com/rai/clean-modularmonolith-go/modules/orders/domain/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// OrderSubmittedHandler handles OrderSubmitted events by sending notifications.
// Performs external side effects; must not run within a database transaction.
type OrderSubmittedHandler struct {
	sender *NotificationSender
}

func NewOrderSubmittedHandler(sender *NotificationSender) *OrderSubmittedHandler {
	return &OrderSubmittedHandler{sender: sender}
}

func (h *OrderSubmittedHandler) HandlerName() string { return "OrderSubmittedHandler" }
func (h *OrderSubmittedHandler) Subdomain() string   { return "notifications" }
func (h *OrderSubmittedHandler) EventType() events.EventType {
	return orderevents.OrderSubmittedEventType
}

func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
	e, ok := event.(orderevents.OrderSubmittedEvent)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}
	return h.sender.SendOrderConfirmation(e.OrderID)
}
