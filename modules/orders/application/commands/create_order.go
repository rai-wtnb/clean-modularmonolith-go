// Package commands contains write use cases for the orders module.
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// CreateOrderCommand creates a new order for a user.
type CreateOrderCommand struct {
	UserID string
}

type CreateOrderHandler struct {
	repo      domain.OrderRepository
	txScope   transaction.Scope
	publisher events.Publisher
}

func NewCreateOrderHandler(
	repo domain.OrderRepository,
	txScope transaction.Scope,
	publisher events.Publisher,
) *CreateOrderHandler {
	return &CreateOrderHandler{
		repo:      repo,
		txScope:   txScope,
		publisher: publisher,
	}
}

// Handle executes the create order use case.
// The operation runs within a transaction, and domain events are dispatched
// before commit, allowing event handlers to participate in the same transaction.
func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (string, error) {
	userRef, err := domain.NewUserRef(cmd.UserID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	return transaction.ExecuteWithResult(ctx, h.txScope, func(ctx context.Context) (string, error) {
		// Create the order aggregate (adds OrderCreatedEvent internally)
		order := domain.NewOrder(userRef)

		// Persist the order
		if err := h.repo.Save(ctx, order); err != nil {
			return "", fmt.Errorf("saving order: %w", err)
		}

		// Publish events (handlers execute immediately within same transaction)
		if err := h.publisher.Publish(ctx, order.PopDomainEvents()...); err != nil {
			return "", fmt.Errorf("publishing events: %w", err)
		}

		return order.ID().String(), nil
	})
}
