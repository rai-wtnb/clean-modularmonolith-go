package eventhandlers

import (
	"context"
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

type OrderSubmittedHandler struct {
	logger *slog.Logger
}

func NewOrderSubmittedHandler(logger *slog.Logger) *OrderSubmittedHandler {
	return &OrderSubmittedHandler{logger: logger}
}

// FIXME: It should be idempotent in real development
func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
	// In a real application, we would unmarshal the payload to get details
	// For this example, we just trust the event type and ID

	// Mock sending email
	h.logger.Info("sending email to user", slog.String("order_id", event.AggregateID()), slog.String("action", "order_confirmation"))

	return nil
}
