// Package commands contains write use cases for the users module.
// Commands change state and typically don't return data (except IDs).
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/eventbus"
	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// CreateUserCommand represents the intent to create a new user.
type CreateUserCommand struct {
	Email     string
	FirstName string
	LastName  string
}

// CreateUserResult holds the result of creating a user.
type CreateUserResult struct {
	UserID string
}

// CreateUserHandler handles the CreateUserCommand.
type CreateUserHandler struct {
	repo            domain.UserRepository
	txScope         transaction.TransactionScope
	handlerRegistry eventbus.HandlerRegistry
}

func NewCreateUserHandler(
	repo domain.UserRepository,
	txScope transaction.TransactionScope,
	handlerRegistry eventbus.HandlerRegistry,
) *CreateUserHandler {
	return &CreateUserHandler{
		repo:            repo,
		txScope:         txScope,
		handlerRegistry: handlerRegistry,
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

	var userID string

	err = h.txScope.Execute(ctx, func(ctx context.Context) error {
		// Create event bus inside closure for Spanner retry safety
		eventBus := eventbus.NewTransactional(h.handlerRegistry, 10)

		// Check for existing email
		exists, err := h.repo.Exists(ctx, email)
		if err != nil {
			return fmt.Errorf("checking email existence: %w", err)
		}
		if exists {
			return domain.ErrEmailExists
		}

		// Create the user aggregate (adds UserCreatedEvent internally)
		user := domain.NewUser(email, name)
		userID = user.ID().String()

		// Persist the user
		if err := h.repo.Save(ctx, user); err != nil {
			return fmt.Errorf("saving user: %w", err)
		}

		// Collect events from aggregate and publish to bus
		for _, event := range user.DomainEvents() {
			if err := eventBus.Publish(ctx, event); err != nil {
				return fmt.Errorf("publishing event: %w", err)
			}
		}
		user.ClearDomainEvents()

		// Flush events (handlers run within same transaction)
		if err := eventBus.Flush(ctx); err != nil {
			return fmt.Errorf("flushing events: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return userID, nil
}
