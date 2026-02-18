package domain

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
)

// OrderRepository defines persistence operations for orders.
type OrderRepository interface {
	Save(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id types.OrderID) (*Order, error)
	FindByUserID(ctx context.Context, userID types.UserID, offset, limit int) ([]*Order, int, error)
	Delete(ctx context.Context, id types.OrderID) error
}
