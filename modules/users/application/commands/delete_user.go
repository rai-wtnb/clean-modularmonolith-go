package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/types"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// DeleteUserCommand represents the intent to delete a user.
type DeleteUserCommand struct {
	UserID string
}

// DeleteUserHandler handles the DeleteUserCommand.
type DeleteUserHandler struct {
	repo      domain.UserRepository
	publisher events.Publisher
}

func NewDeleteUserHandler(repo domain.UserRepository, publisher events.Publisher) *DeleteUserHandler {
	return &DeleteUserHandler{
		repo:      repo,
		publisher: publisher,
	}
}

// Handle executes the delete user use case.
func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
	// Parse user ID
	userID, err := types.ParseUserID(cmd.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Verify user exists
	user, err := h.repo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("finding user: %w", err)
	}

	// Mark as deleted (soft delete via domain method)
	if err := user.Delete(); err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}

	// Persist the deletion
	if err := h.repo.Save(ctx, user); err != nil {
		return fmt.Errorf("saving user: %w", err)
	}

	// Publish domain event
	if h.publisher != nil {
		event := domain.NewUserDeletedEvent(userID)
		if err := h.publisher.Publish(ctx, event); err != nil {
			_ = err
		}
	}

	return nil
}
