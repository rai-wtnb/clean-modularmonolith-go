// Package commands contains write use cases for the orders module.
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
)

// CreateOrderCommand creates a new order for a user.
type CreateOrderCommand struct {
	UserID string
}

type CreateOrderHandler struct {
	repo      domain.OrderRepository
	publisher events.Publisher
}

func NewCreateOrderHandler(repo domain.OrderRepository, publisher events.Publisher) *CreateOrderHandler {
	return &CreateOrderHandler{repo: repo, publisher: publisher}
}

func (h *CreateOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (string, error) {
	userID, err := types.ParseUserID(cmd.UserID)
	if err != nil {
		return "", fmt.Errorf("invalid user ID: %w", err)
	}

	order := domain.NewOrder(userID)

	if err := h.repo.Save(ctx, order); err != nil {
		return "", fmt.Errorf("saving order: %w", err)
	}

	if h.publisher != nil {
		event := domain.NewOrderCreatedEvent(order)
		_ = h.publisher.Publish(ctx, event)
	}

	return order.ID().String(), nil
}
