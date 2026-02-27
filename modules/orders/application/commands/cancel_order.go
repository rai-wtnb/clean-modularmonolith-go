package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// CancelOrderCommand cancels an order.
type CancelOrderCommand struct {
	OrderID string
}

type CancelOrderHandler struct {
	repo           domain.OrderRepository
	txScope        transaction.TransactionScope
	eventPublisher events.Publisher
}

func NewCancelOrderHandler(repo domain.OrderRepository, txScope transaction.TransactionScope, eventPublisher events.Publisher) *CancelOrderHandler {
	return &CancelOrderHandler{
		repo:           repo,
		txScope:        txScope,
		eventPublisher: eventPublisher,
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
		order, err := h.repo.FindByID(ctx, orderID)
		if err != nil {
			return fmt.Errorf("finding order: %w", err)
		}

		if err := order.Cancel(); err != nil {
			return err
		}

		if err := h.repo.Save(ctx, order); err != nil {
			return fmt.Errorf("saving order: %w", err)
		}

		if err := h.eventPublisher.Publish(ctx, order.PopDomainEvents()...); err != nil {
			return fmt.Errorf("publishing events: %w", err)
		}

		return nil
	})
}
