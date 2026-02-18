// Package queries contains read use cases for the orders module.
package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
)

// OrderDTO is a read model for order data.
type OrderDTO struct {
	ID        string         `json:"id"`
	UserID    string         `json:"user_id"`
	Items     []OrderItemDTO `json:"items"`
	Status    string         `json:"status"`
	Total     MoneyDTO       `json:"total"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type OrderItemDTO struct {
	ProductID   string   `json:"product_id"`
	ProductName string   `json:"product_name"`
	Quantity    int      `json:"quantity"`
	UnitPrice   MoneyDTO `json:"unit_price"`
	Subtotal    MoneyDTO `json:"subtotal"`
}

type MoneyDTO struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// GetOrderQuery retrieves an order by ID.
type GetOrderQuery struct {
	OrderID string
}

type GetOrderHandler struct {
	repo domain.OrderRepository
}

func NewGetOrderHandler(repo domain.OrderRepository) *GetOrderHandler {
	return &GetOrderHandler{repo: repo}
}

func (h *GetOrderHandler) Handle(ctx context.Context, query GetOrderQuery) (*OrderDTO, error) {
	orderID, err := types.ParseOrderID(query.OrderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	return toOrderDTO(order), nil
}

func toOrderDTO(order *domain.Order) *OrderDTO {
	items := make([]OrderItemDTO, len(order.Items()))
	for i, item := range order.Items() {
		subtotal := item.Subtotal()
		items[i] = OrderItemDTO{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice: MoneyDTO{
				Amount:   item.UnitPrice.Amount(),
				Currency: item.UnitPrice.Currency(),
			},
			Subtotal: MoneyDTO{
				Amount:   subtotal.Amount(),
				Currency: subtotal.Currency(),
			},
		}
	}

	return &OrderDTO{
		ID:     order.ID().String(),
		UserID: order.UserID().String(),
		Items:  items,
		Status: order.Status().String(),
		Total: MoneyDTO{
			Amount:   order.Total().Amount(),
			Currency: order.Total().Currency(),
		},
		CreatedAt: order.CreatedAt(),
		UpdatedAt: order.UpdatedAt(),
	}
}
