package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"

	orderevents "github.com/rai/clean-modularmonolith-go/modules/orders/domain/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// OrderSubmittedHandler handles OrderSubmitted events by sending notifications.
// Performs external side effects; must not run within a database transaction.
type OrderSubmittedHandler struct {
	events.IdempotentBase
	logger *slog.Logger
}

func NewOrderSubmittedHandler(logger *slog.Logger) *OrderSubmittedHandler {
	return &OrderSubmittedHandler{logger: logger}
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

	onceFn := func() error {
		h.logger.Info("sending email to user", slog.String("order_id", e.OrderID), slog.String("action", "order_confirmation"))
		return nil
	}
	return h.Once(fmt.Sprintf("send-confirmation:%s", e.OrderID), onceFn)
}
