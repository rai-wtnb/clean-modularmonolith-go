package domain

import (
	"context"

	userdomain "github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// OrderRepository defines persistence operations for orders.
type OrderRepository interface {
	Save(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id OrderID) (*Order, error)
	FindByUserID(ctx context.Context, userID userdomain.UserID, offset, limit int) ([]*Order, int, error)
	Delete(ctx context.Context, id OrderID) error
}
