package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// CancelOrderCommand cancels an order.
type CancelOrderCommand struct {
	OrderID string
}

type CancelOrderHandler struct {
	repo      domain.OrderRepository
	publisher events.Publisher
}

func NewCancelOrderHandler(repo domain.OrderRepository, publisher events.Publisher) *CancelOrderHandler {
	return &CancelOrderHandler{repo: repo, publisher: publisher}
}

func (h *CancelOrderHandler) Handle(ctx context.Context, cmd CancelOrderCommand) error {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

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

	if h.publisher != nil {
		event := domain.NewOrderCancelledEvent(order)
		_ = h.publisher.Publish(ctx, event)
	}

	return nil
}
