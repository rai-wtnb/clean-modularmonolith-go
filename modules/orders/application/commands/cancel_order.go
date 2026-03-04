package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// CancelOrderCommand cancels an order.
type CancelOrderCommand struct {
	OrderID string
}

type CancelOrderHandler struct {
	repo    domain.OrderRepository
	txScope transaction.Scope
}

func NewCancelOrderHandler(repo domain.OrderRepository, txScope transaction.Scope) *CancelOrderHandler {
	return &CancelOrderHandler{
		repo:    repo,
		txScope: txScope,
	}
}

// Handle executes the cancel order use case.
// The operation runs within a transaction. Domain events are collected
// in the context and automatically published by EventAwareScope.
func (h *CancelOrderHandler) Handle(ctx context.Context, cmd CancelOrderCommand) (*domain.Order, error) {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	return transaction.ExecuteWithResult(ctx, h.txScope, func(ctx context.Context) (*domain.Order, error) {
		order, err := h.repo.FindByID(ctx, orderID)
		if err != nil {
			return nil, fmt.Errorf("finding order: %w", err)
		}

		if err := order.Cancel(ctx); err != nil {
			return nil, err
		}

		if err := h.repo.Save(ctx, order); err != nil {
			return nil, fmt.Errorf("saving order: %w", err)
		}

		return order, nil
	})
}
