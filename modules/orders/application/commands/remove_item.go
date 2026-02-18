package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
)

// RemoveItemCommand removes an item from an order.
type RemoveItemCommand struct {
	OrderID   string
	ProductID string
}

type RemoveItemHandler struct {
	repo domain.OrderRepository
}

func NewRemoveItemHandler(repo domain.OrderRepository) *RemoveItemHandler {
	return &RemoveItemHandler{repo: repo}
}

func (h *RemoveItemHandler) Handle(ctx context.Context, cmd RemoveItemCommand) error {
	orderID, err := types.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("finding order: %w", err)
	}

	if err := order.RemoveItem(cmd.ProductID); err != nil {
		return err
	}

	if err := h.repo.Save(ctx, order); err != nil {
		return fmt.Errorf("saving order: %w", err)
	}

	return nil
}
