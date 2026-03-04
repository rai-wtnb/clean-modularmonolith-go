// Package commands contains write use cases for the users module.
// Commands change state and typically don't return data (except IDs).
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// CreateUserCommand represents the intent to create a new user.
type CreateUserCommand struct {
	Email     string
	FirstName string
	LastName  string
}

// CreateUserHandler handles the CreateUserCommand.
type CreateUserHandler struct {
	repo    domain.UserRepository
	txScope transaction.Scope
}

func NewCreateUserHandler(repo domain.UserRepository, txScope transaction.Scope) *CreateUserHandler {
	return &CreateUserHandler{
		repo:    repo,
		txScope: txScope,
	}
}

// Handle executes the create user use case.
// The operation runs within a transaction. Domain events are collected
// in the context and automatically published by EventAwareScope.
func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (string, error) {
	// Validate and create value objects (before transaction)
	email, err := domain.NewEmail(cmd.Email)
	if err != nil {
		return "", fmt.Errorf("invalid email: %w", err)
	}

	name, err := domain.NewName(cmd.FirstName, cmd.LastName)
	if err != nil {
		return "", fmt.Errorf("invalid name: %w", err)
	}

	return transaction.ExecuteWithResult(ctx, h.txScope, func(ctx context.Context) (string, error) {
		exists, err := h.repo.Exists(ctx, email)
		if err != nil {
			return "", fmt.Errorf("checking email existence: %w", err)
		}
		if exists {
			return "", domain.ErrEmailExists
		}

		// Create the user aggregate (adds UserCreatedEvent to ctx)
		user := domain.NewUser(ctx, email, name)

		// Persist the user
		if err := h.repo.Save(ctx, user); err != nil {
			return "", fmt.Errorf("saving user: %w", err)
		}

		return user.ID().String(), nil
	})
}
