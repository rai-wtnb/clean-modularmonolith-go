package eventhandlers

import (
	"context"
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/orders/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
	userdomain "github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// UserDeletedHandler handles UserDeleted events by canceling pending orders.
type UserDeletedHandler struct {
	cancelOrderHandler *commands.CancelOrderHandler
	logger             *slog.Logger
}

func NewUserDeletedHandler(cancelOrderHandler *commands.CancelOrderHandler, logger *slog.Logger) *UserDeletedHandler {
	return &UserDeletedHandler{
		cancelOrderHandler: cancelOrderHandler,
		logger:             logger,
	}
}

func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
	userDeleted, ok := event.(userdomain.UserDeletedEvent)
	if !ok {
		h.logger.Warn("unexpected event type", slog.String("expected", "UserDeletedEvent"))
		return nil
	}

	h.logger.Info("handling user deleted event, canceling user orders",
		slog.String("user_id", userDeleted.UserID.String()),
	)

	// Cancel all pending orders for this user
	// In a real application, we would:
	// 1. Query all pending orders for the user
	// 2. Cancel each order
	// For now, we just log the intent
	_ = types.UserID(userDeleted.UserID)

	return nil
}
