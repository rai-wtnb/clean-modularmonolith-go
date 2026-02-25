package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/eventbus"
	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
)

// CancelOrderCommand cancels an order.
type CancelOrderCommand struct {
	OrderID string
}

type CancelOrderHandler struct {
	repo            domain.OrderRepository
	txScope         transaction.TransactionScope
	handlerRegistry eventbus.HandlerRegistry
}

func NewCancelOrderHandler(
	repo domain.OrderRepository,
	txScope transaction.TransactionScope,
	handlerRegistry eventbus.HandlerRegistry,
) *CancelOrderHandler {
	return &CancelOrderHandler{
		repo:            repo,
		txScope:         txScope,
		handlerRegistry: handlerRegistry,
	}
}

// Handle executes the cancel order use case.
// The operation runs within a transaction, and domain events are dispatched
// before commit, allowing event handlers to participate in the same transaction.
func (h *CancelOrderHandler) Handle(ctx context.Context, cmd CancelOrderCommand) error {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	return h.txScope.Execute(ctx, func(ctx context.Context) error {
		// Create event bus inside closure for Spanner retry safety
		eventBus := eventbus.NewTransactional(h.handlerRegistry, 10)

		// Load aggregate
		order, err := h.repo.FindByID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("finding order: %w", err)
		}

		// Execute business logic (adds OrderCancelledEvent internally)
		if err := order.Cancel(); err != nil {
			return err
		}

		// Persist changes
		if err := h.repo.Save(ctx, order); err != nil {
			return fmt.Errorf("saving order: %w", err)
		}

		// Collect events from aggregate and publish to bus
		for _, event := range order.DomainEvents() {
			if err := eventBus.Publish(ctx, event); err != nil {
				return fmt.Errorf("publishing event: %w", err)
			}
		}
		order.ClearDomainEvents()

		// Flush events (handlers run within same transaction)
		if err := eventBus.Flush(ctx); err != nil {
			return fmt.Errorf("flushing events: %w", err)
		}

		return nil
	})
}
