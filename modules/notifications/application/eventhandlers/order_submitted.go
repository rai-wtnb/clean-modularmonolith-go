package eventhandlers

import (
	"context"
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// OrderSubmittedHandler handles OrderSubmitted events by sending notifications.
//
// IMPORTANT: This handler performs external side effects (email sending) and
// MUST NOT run within a database transaction. Currently it runs via
// InMemoryEventBus (outside transactions).
//
// Future: This will be migrated to Pub/Sub where:
// - Events are published to a message queue
// - This handler subscribes and processes asynchronously
// - Idempotency is handled on the subscriber side (using event ID deduplication)
type OrderSubmittedHandler struct {
	logger *slog.Logger
}

func NewOrderSubmittedHandler(logger *slog.Logger) *OrderSubmittedHandler {
	return &OrderSubmittedHandler{logger: logger}
}

// Handle processes the OrderSubmitted event.
// TODO: Implement idempotency using event ID before production use.
func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
	// In a real application, we would unmarshal the payload to get details
	// For this example, we just trust the event type and ID

	// Mock sending email
	h.logger.Info("sending email to user", slog.String("order_id", event.AggregateID()), slog.String("action", "order_confirmation"))

	return nil
}
