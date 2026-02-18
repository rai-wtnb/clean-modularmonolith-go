// Package persistence implements repository interfaces using specific storage backends.
// This is the outermost layer - it implements ports defined in the domain layer.
package persistence

import (
	"context"
	"sync"

	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// InMemoryRepository implements UserRepository using in-memory storage.
// Useful for testing and development.
type InMemoryRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users: make(map[string]*domain.User),
	}
}

func (r *InMemoryRepository) Save(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.ID().String()] = user
	return nil
}

func (r *InMemoryRepository) FindByID(ctx context.Context, id types.UserID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[id.String()]
	if !exists {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email().Equals(email) {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *InMemoryRepository) Delete(ctx context.Context, id types.UserID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.users, id.String())
	return nil
}

func (r *InMemoryRepository) Exists(ctx context.Context, email domain.Email) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.Email().Equals(email) && user.Status() != domain.StatusDeleted {
			return true, nil
		}
	}
	return false, nil
}

func (r *InMemoryRepository) FindAll(ctx context.Context, offset, limit int) ([]*domain.User, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Collect active users
	var activeUsers []*domain.User
	for _, user := range r.users {
		if user.Status() != domain.StatusDeleted {
			activeUsers = append(activeUsers, user)
		}
	}

	total := len(activeUsers)

	// Apply pagination
	if offset >= total {
		return []*domain.User{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return activeUsers[offset:end], total, nil
}
