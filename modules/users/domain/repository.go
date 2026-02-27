package domain

import (
	"context"
)

// UserRepository defines the persistence interface for users.
// This is a port - defined in domain, implemented in infrastructure.
// Following the Interface Segregation Principle, we keep the interface minimal.
type UserRepository interface {
	// Save persists a user (create or update).
	Save(ctx context.Context, user *User) error

	// FindByID retrieves a user by ID.
	// Returns ErrUserNotFound if user doesn't exist.
	FindByID(ctx context.Context, id UserID) (*User, error)

	// FindByEmail retrieves a user by email.
	// Returns ErrUserNotFound if user doesn't exist.
	FindByEmail(ctx context.Context, email Email) (*User, error)

	// Exists checks if a user with the given email exists.
	Exists(ctx context.Context, email Email) (bool, error)

	// FindAll retrieves users with pagination.
	FindAll(ctx context.Context, offset, limit int) ([]*User, int, error)
}
