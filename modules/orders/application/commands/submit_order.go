package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// SubmitOrderCommand submits an order for processing.
type SubmitOrderCommand struct {
	OrderID string
}

type SubmitOrderHandler struct {
	repo      domain.OrderRepository
	publisher events.Publisher
}

func NewSubmitOrderHandler(repo domain.OrderRepository, publisher events.Publisher) *SubmitOrderHandler {
	return &SubmitOrderHandler{repo: repo, publisher: publisher}
}

func (h *SubmitOrderHandler) Handle(ctx context.Context, cmd SubmitOrderCommand) error {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

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

	if h.publisher != nil {
		event := domain.NewOrderSubmittedEvent(order)
		_ = h.publisher.Publish(ctx, event)
	}

	return nil
}
