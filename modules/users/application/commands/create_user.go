// Package commands contains write use cases for the users module.
// Commands change state and typically don't return data (except IDs).
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
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
	repo      domain.UserRepository
	txScope   transaction.Scope
	publisher events.Publisher
}

func NewCreateUserHandler(repo domain.UserRepository, txScope transaction.Scope, publisher events.Publisher) *CreateUserHandler {
	return &CreateUserHandler{
		repo:      repo,
		txScope:   txScope,
		publisher: publisher,
	}
}

// Handle executes the create user use case.
// The operation runs within a transaction, and domain events are dispatched
// before commit, allowing event handlers to participate in the same transaction.
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

		// Create the user aggregate (adds UserCreatedEvent internally)
		user := domain.NewUser(email, name)

		// Persist the user
		if err := h.repo.Save(ctx, user); err != nil {
			return "", fmt.Errorf("saving user: %w", err)
		}

		if err := h.publisher.Publish(ctx, user.PopDomainEvents()...); err != nil {
			return "", fmt.Errorf("publishing events: %w", err)
		}

		return user.ID().String(), nil
	})
}
