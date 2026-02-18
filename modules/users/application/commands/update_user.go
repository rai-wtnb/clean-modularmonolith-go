package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// UpdateUserCommand represents the intent to update a user's profile.
type UpdateUserCommand struct {
	UserID    string
	FirstName string
	LastName  string
}

// UpdateUserHandler handles the UpdateUserCommand.
type UpdateUserHandler struct {
	repo      domain.UserRepository
	publisher events.Publisher
}

func NewUpdateUserHandler(repo domain.UserRepository, publisher events.Publisher) *UpdateUserHandler {
	return &UpdateUserHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the update user use case.
func (h *UpdateUserHandler) Handle(ctx context.Context, cmd UpdateUserCommand) error {
	// Parse user ID
	userID, err := types.ParseUserID(cmd.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Retrieve the user
	user, err := h.repo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}

	// Create new name value object
	name, err := domain.NewName(cmd.FirstName, cmd.LastName)
	if err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Update the user (domain method enforces business rules)
	if err := user.UpdateProfile(name); err != nil {
		return fmt.Errorf("updating profile: %w", err)
	}

	// Persist changes
	if err := h.repo.Save(ctx, user); err != nil {
		return fmt.Errorf("saving user: %w", err)
	}

	// Publish domain event
	if h.publisher != nil {
		event := domain.NewUserUpdatedEvent(user)
		if err := h.publisher.Publish(ctx, event); err != nil {
			_ = err
		}
	}

	return nil
}
