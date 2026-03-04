package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
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
	repo    domain.UserRepository
	txScope transaction.Scope
}

func NewUpdateUserHandler(repo domain.UserRepository, txScope transaction.Scope) *UpdateUserHandler {
	return &UpdateUserHandler{
		repo:    repo,
		txScope: txScope,
	}
}

// Handle executes the update user use case.
// The operation runs within a transaction. Domain events are collected
// in the context and automatically published by EventAwareScope.
func (h *UpdateUserHandler) Handle(ctx context.Context, cmd UpdateUserCommand) error {
	userID, err := domain.ParseUserID(cmd.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	name, err := domain.NewName(cmd.FirstName, cmd.LastName)
	if err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	fn := func(ctx context.Context) error {
		user, err := h.repo.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("finding user: %w", err)
		}

		if err := user.UpdateProfile(ctx, name); err != nil {
			return fmt.Errorf("updating profile: %w", err)
		}

		if err := h.repo.Save(ctx, user); err != nil {
			return fmt.Errorf("saving user: %w", err)
		}

		return nil
	}
	return h.txScope.Execute(ctx, fn)
}
