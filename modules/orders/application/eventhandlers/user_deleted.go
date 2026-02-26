package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events/contracts"
)

// UserDeletedHandler handles UserDeleted events by canceling pending orders.
// This handler runs within the same transaction as the user deletion,
// ensuring atomic consistency between user deletion and order cancellation.
type UserDeletedHandler struct {
	orderRepo domain.OrderRepository
	logger    *slog.Logger
}

func NewUserDeletedHandler(orderRepo domain.OrderRepository, logger *slog.Logger) *UserDeletedHandler {
	return &UserDeletedHandler{
		orderRepo: orderRepo,
		logger:    logger,
	}
}

func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
	userDeletedEvent, ok := event.(contracts.UserDeletedEvent)
	if !ok {
		return fmt.Errorf("unexpected event type: %T", event)
	}

	h.logger.Info("handling user deleted event, canceling user orders", slog.String("user_id", userDeletedEvent.UserID))

	userRef, err := domain.NewUserRef(userDeletedEvent.UserID)
	if err != nil {
		return fmt.Errorf("parsing user ID: %w", err)
	}

	// The context contains the transaction from the originating command
	// Repository operations here participate in the same transaction
	orders, _, err := h.orderRepo.FindByUserRef(ctx, userRef, 0, 1000)
	if err != nil {
		return fmt.Errorf("finding user orders: %w", err)
	}

	for order := range slices.Values(orders) {
		// Only cancel orders that are in draft or pending status
		if order.Status() != domain.StatusDraft && order.Status() != domain.StatusPending {
			continue
		}

		if err := order.Cancel(); err != nil {
			h.logger.Warn("failed to cancel order",
				slog.String("order_id", order.ID().String()),
				slog.Any("error", err),
			)
			continue
		}

		if err := h.orderRepo.Save(ctx, order); err != nil {
			return fmt.Errorf("saving canceled order %s: %w", order.ID().String(), err)
		}

		h.logger.Info("canceled order for deleted user",
			slog.String("order_id", order.ID().String()),
			slog.String("user_id", userDeletedEvent.UserID),
		)
	}

	return nil
}
