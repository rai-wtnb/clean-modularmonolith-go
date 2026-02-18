// Package persistence implements repository interfaces for orders.
package persistence

import (
	"context"
	"sync"

	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
)

// InMemoryRepository implements OrderRepository using in-memory storage.
type InMemoryRepository struct {
	mu     sync.RWMutex
	orders map[string]*domain.Order
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		orders: make(map[string]*domain.Order),
	}
}

func (r *InMemoryRepository) Save(ctx context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID().String()] = order
	return nil
}

func (r *InMemoryRepository) FindByID(ctx context.Context, id types.OrderID) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	order, exists := r.orders[id.String()]
	if !exists {
		return nil, domain.ErrOrderNotFound
	}
	return order, nil
}

func (r *InMemoryRepository) FindByUserID(ctx context.Context, userID types.UserID, offset, limit int) ([]*domain.Order, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var userOrders []*domain.Order
	for _, order := range r.orders {
		if order.UserID().String() == userID.String() {
			userOrders = append(userOrders, order)
		}
	}

	total := len(userOrders)

	if offset >= total {
		return []*domain.Order{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return userOrders[offset:end], total, nil
}

func (r *InMemoryRepository) Delete(ctx context.Context, id types.OrderID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.orders, id.String())
	return nil
}
