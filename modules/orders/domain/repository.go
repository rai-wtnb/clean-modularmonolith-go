package domain

import "context"

// OrderRepository defines persistence operations for orders.
type OrderRepository interface {
	Save(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id OrderID) (*Order, error)
	FindByUserRef(ctx context.Context, userRef UserRef, offset, limit int) ([]*Order, int, error)
	Delete(ctx context.Context, id OrderID) error
}
