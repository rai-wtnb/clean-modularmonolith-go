package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// SubmitOrderCommand submits an order for processing.
type SubmitOrderCommand struct {
	OrderID string
}

type SubmitOrderHandler struct {
	repo    domain.OrderRepository
	txScope transaction.ScopeWithDomainEvent
}

func NewSubmitOrderHandler(repo domain.OrderRepository, txScope transaction.ScopeWithDomainEvent) *SubmitOrderHandler {
	return &SubmitOrderHandler{
		repo:    repo,
		txScope: txScope,
	}
}

// Handle executes the submit order use case.
//
// NOTE: OrderSubmittedEvent is dispatched within the transaction, but
// notification handlers (e.g., email) should NOT run here. They should
// be handled via Pub/Sub with idempotency on the subscriber side.
func (h *SubmitOrderHandler) Handle(ctx context.Context, cmd SubmitOrderCommand) error {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	return h.txScope.ExecuteWithPublish(ctx, func(ctx context.Context) error {
		order, err := h.repo.FindByID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("finding order: %w", err)
		}

		if err := order.Submit(ctx); err != nil {
			return err
		}

		if err := h.repo.Save(ctx, order); err != nil {
			return fmt.Errorf("saving order: %w", err)
		}

		return nil
	})
}
