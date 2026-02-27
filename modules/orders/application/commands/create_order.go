// Package commands contains write use cases for the orders module.
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// CreateOrderCommand creates a new order for a user.
type CreateOrderCommand struct {
	UserID string
}

type CreateOrderHandler struct {
	repo      domain.OrderRepository
	txScope   transaction.TransactionScope
	publisher events.Publisher
}

func NewCreateOrderHandler(
	repo domain.OrderRepository,
	txScope transaction.TransactionScope,
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

	var orderID string

	err = h.txScope.Execute(ctx, func(ctx context.Context) error {
		// Create the order aggregate (adds OrderCreatedEvent internally)
		order := domain.NewOrder(userRef)
		orderID = order.ID().String()

		// Persist the order
		if err := h.repo.Save(ctx, order); err != nil {
			return fmt.Errorf("saving order: %w", err)
		}

		// Publish events (handlers execute immediately within same transaction)
		if err := h.publisher.Publish(ctx, order.PopDomainEvents()...); err != nil {
			return fmt.Errorf("publishing events: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return orderID, nil
}
