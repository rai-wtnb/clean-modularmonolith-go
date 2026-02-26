// Package commands contains write use cases for the orders module.
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/eventbus"
	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
)

// CreateOrderCommand creates a new order for a user.
type CreateOrderCommand struct {
	UserID string
}

type CreateOrderHandler struct {
	repo            domain.OrderRepository
	txScope         transaction.TransactionScope
	handlerRegistry eventbus.HandlerRegistry
}

func NewCreateOrderHandler(
	repo domain.OrderRepository,
	txScope transaction.TransactionScope,
	handlerRegistry eventbus.HandlerRegistry,
) *CreateOrderHandler {
	return &CreateOrderHandler{
		repo:            repo,
		txScope:         txScope,
		handlerRegistry: handlerRegistry,
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
		// Create publisher inside closure for Spanner retry safety
		publisher := eventbus.NewTransactionalPublisher(h.handlerRegistry, 10)

		// Create the order aggregate (adds OrderCreatedEvent internally)
		order := domain.NewOrder(userRef)
		orderID = order.ID().String()

		// Persist the order
		if err := h.repo.Save(ctx, order); err != nil {
			return fmt.Errorf("saving order: %w", err)
		}

		// Collect events from aggregate and publish
		for _, event := range order.DomainEvents() {
			if err := publisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("publishing event: %w", err)
			}
		}
		order.ClearDomainEvents()

		// Flush events (handlers run within same transaction)
		if err := publisher.Flush(ctx); err != nil {
			return fmt.Errorf("flushing events: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return orderID, nil
}
