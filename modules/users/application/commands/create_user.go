// Package commands contains write use cases for the users module.
// Commands change state and typically don't return data (except IDs).
package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
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
	publisher events.Publisher
}

func NewCreateUserHandler(repo domain.UserRepository, publisher events.Publisher) *CreateUserHandler {
	return &CreateUserHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the create user use case.
func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (string, error) {
	// Validate and create value objects
	email, err := domain.NewEmail(cmd.Email)
	if err != nil {
		return "", fmt.Errorf("invalid email: %w", err)
	}

	name, err := domain.NewName(cmd.FirstName, cmd.LastName)
	if err != nil {
		return "", fmt.Errorf("invalid name: %w", err)
	}

	// Check for existing email
	exists, err := h.repo.Exists(ctx, email)
	if err != nil {
		return "", fmt.Errorf("checking email existence: %w", err)
	}
	if exists {
		return "", domain.ErrEmailExists
	}

	// Create the user aggregate
	user := domain.NewUser(email, name)

	// Persist the user
	if err := h.repo.Save(ctx, user); err != nil {
		return "", fmt.Errorf("saving user: %w", err)
	}

	// Publish domain event
	if h.publisher != nil {
		event := domain.NewUserCreatedEvent(user)
		if err := h.publisher.Publish(ctx, event); err != nil {
			// Log but don't fail - event publishing is eventually consistent
			// In production, use outbox pattern for reliability
			_ = err
		}
	}

	return user.ID().String(), nil
}
