package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/eventbus"
	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
)

// SubmitOrderCommand submits an order for processing.
type SubmitOrderCommand struct {
	OrderID string
}

type SubmitOrderHandler struct {
	repo            domain.OrderRepository
	txScope         transaction.TransactionScope
	handlerRegistry eventbus.HandlerRegistry
}

func NewSubmitOrderHandler(
	repo domain.OrderRepository,
	txScope transaction.TransactionScope,
	handlerRegistry eventbus.HandlerRegistry,
) *SubmitOrderHandler {
	return &SubmitOrderHandler{
		repo:            repo,
		txScope:         txScope,
		handlerRegistry: handlerRegistry,
	}
}

// Handle executes the submit order use case.
// The operation runs within a transaction, and domain events are dispatched
// before commit, allowing event handlers to participate in the same transaction.
//
// NOTE: OrderSubmittedEvent is dispatched within the transaction, but
// notification handlers (e.g., email) should NOT run here. They should
// be handled via Pub/Sub with idempotency on the subscriber side.
func (h *SubmitOrderHandler) Handle(ctx context.Context, cmd SubmitOrderCommand) error {
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

		// Execute business logic (adds OrderSubmittedEvent internally)
		if err := order.Submit(); err != nil {
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
