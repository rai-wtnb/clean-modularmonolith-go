package queries

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
)

// OrderListDTO contains a paginated list of orders.
type OrderListDTO struct {
	Orders     []*OrderDTO `json:"orders"`
	TotalCount int         `json:"total_count"`
	Offset     int         `json:"offset"`
	Limit      int         `json:"limit"`
}

// ListUserOrdersQuery retrieves orders for a specific user.
type ListUserOrdersQuery struct {
	UserID string
	Offset int
	Limit  int
}

type ListUserOrdersHandler struct {
	repo domain.OrderRepository
}

func NewListUserOrdersHandler(repo domain.OrderRepository) *ListUserOrdersHandler {
	return &ListUserOrdersHandler{repo: repo}
}

func (h *ListUserOrdersHandler) Handle(ctx context.Context, query ListUserOrdersQuery) (*OrderListDTO, error) {
	userID, err := types.ParseUserID(query.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	orders, total, err := h.repo.FindByUserID(ctx, userID, query.Offset, limit)
	if err != nil {
		return nil, err
	}

	dtos := make([]*OrderDTO, len(orders))
	for i, order := range orders {
		dtos[i] = toOrderDTO(order)
	}

	return &OrderListDTO{
		Orders:     dtos,
		TotalCount: total,
		Offset:     query.Offset,
		Limit:      limit,
	}, nil
}
