package eventhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events/contracts"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// UserDeletedHandler handles UserDeleted events by canceling pending orders.
// It uses TransactionScope to declare its own transactional boundary.
// If a transaction already exists in the context (e.g., from the originating command),
// TransactionScope joins it; otherwise, it creates a new one.
type UserDeletedHandler struct {
	orderRepo domain.OrderRepository
	txScope   transaction.Scope
	logger    *slog.Logger
}

func NewUserDeletedHandler(orderRepo domain.OrderRepository, txScope transaction.Scope, logger *slog.Logger) *UserDeletedHandler {
	return &UserDeletedHandler{
		orderRepo: orderRepo,
		txScope:   txScope,
		logger:    logger,
	}
}

func (h *UserDeletedHandler) HandlerName() string { return "UserDeletedHandler" }
func (h *UserDeletedHandler) Subdomain() string   { return "orders" }

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

	fn := func(ctx context.Context) error {
		orders, _, err := h.orderRepo.FindByUserRef(ctx, userRef, 0, 1000)
		if err != nil {
			return fmt.Errorf("finding user orders: %w", err)
		}

		for order := range slices.Values(orders) {
			if order.Status() != domain.StatusDraft && order.Status() != domain.StatusPending {
				continue
			}

			if err := order.Cancel(ctx); err != nil {
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
	return h.txScope.Execute(ctx, fn)
}
