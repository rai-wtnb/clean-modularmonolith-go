package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
)

// AddItemCommand adds an item to an order.
type AddItemCommand struct {
	OrderID     string
	ProductID   string
	ProductName string
	Quantity    int
	UnitPrice   int64
	Currency    string
}

type AddItemHandler struct {
	repo domain.OrderRepository
}

func NewAddItemHandler(repo domain.OrderRepository) *AddItemHandler {
	return &AddItemHandler{repo: repo}
}

func (h *AddItemHandler) Handle(ctx context.Context, cmd AddItemCommand) error {
	orderID, err := domain.ParseOrderID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("finding order: %w", err)
	}

	unitPrice, err := domain.NewMoney(cmd.UnitPrice, cmd.Currency)
	if err != nil {
		return fmt.Errorf("invalid unit price: %w", err)
	}

	if err := order.AddItem(cmd.ProductID, cmd.ProductName, cmd.Quantity, unitPrice); err != nil {
		return err
	}

	if err := h.repo.Save(ctx, order); err != nil {
		return fmt.Errorf("saving order: %w", err)
	}

	return nil
}
