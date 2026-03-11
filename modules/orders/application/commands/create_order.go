// Package commands contains write use cases for the orders module.
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// CreateOrderCommand creates a new order for a user.
type CreateOrderCommand struct {
	UserID string
}

type CreateOrderHandler struct {
	repo    domain.OrderRepository
	txScope transaction.ScopeWithDomainEvent
}

func NewCreateOrderHandler(repo domain.OrderRepository, txScope transaction.ScopeWithDomainEvent) *CreateOrderHandler {
	return &CreateOrderHandler{
		repo:    repo,
		txScope: txScope,
	}
}

// Handle executes the create order use case.
func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (string, error) {
	userRef, err := domain.NewUserRef(cmd.UserID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	return transaction.ExecuteWithPublishResult(ctx, h.txScope, func(ctx context.Context) (string, error) {
		// Create the order aggregate (adds OrderCreatedEvent to ctx)
		order := domain.NewOrder(ctx, userRef)

		// Persist the order
		if err := h.repo.Save(ctx, order); err != nil {
			return "", fmt.Errorf("saving order: %w", err)
		}

		return order.ID().String(), nil
	})
}
