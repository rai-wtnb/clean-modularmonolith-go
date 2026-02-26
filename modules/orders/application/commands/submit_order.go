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
		publisher := eventbus.NewTransactionalPublisher(h.handlerRegistry, 10)

		order, err := h.repo.FindByID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("finding order: %w", err)
		}

		if err := order.Submit(); err != nil {
			return err
		}

		if err := h.repo.Save(ctx, order); err != nil {
			return fmt.Errorf("saving order: %w", err)
		}

		for _, event := range order.DomainEvents() {
			if err := publisher.Publish(ctx, event); err != nil {
				return fmt.Errorf("publishing event: %w", err)
			}
		}
		order.ClearDomainEvents()

		if err := publisher.Flush(ctx); err != nil {
			return fmt.Errorf("flushing events: %w", err)
		}

		return nil
	})
}
