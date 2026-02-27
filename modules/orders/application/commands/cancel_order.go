package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// CancelOrderCommand cancels an order.
type CancelOrderCommand struct {
	OrderID string
}

type CancelOrderHandler struct {
	repo           domain.OrderRepository
	txScope        transaction.Scope
	eventPublisher events.Publisher
}

func NewCancelOrderHandler(repo domain.OrderRepository, txScope transaction.Scope, eventPublisher events.Publisher) *CancelOrderHandler {
	return &CancelOrderHandler{
		repo:           repo,
		txScope:        txScope,
		eventPublisher: eventPublisher,
	}
}

// Handle executes the cancel order use case.
// The operation runs within a transaction, and domain events are dispatched
// before commit, allowing event handlers to participate in the same transaction.
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

		if err := order.Cancel(); err != nil {
			return nil, err
		}

		if err := h.repo.Save(ctx, order); err != nil {
			return nil, fmt.Errorf("saving order: %w", err)
		}

		if err := h.eventPublisher.Publish(ctx, order.PopDomainEvents()...); err != nil {
			return nil, fmt.Errorf("publishing events: %w", err)
		}

		return order, nil
	})
}
