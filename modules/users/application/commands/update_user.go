package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
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
	txScope   transaction.TransactionScope
	publisher events.Publisher
}

func NewUpdateUserHandler(
	repo domain.UserRepository,
	txScope transaction.TransactionScope,
	publisher events.Publisher,
) *UpdateUserHandler {
	return &UpdateUserHandler{
		repo:      repo,
		txScope:   txScope,
		publisher: publisher,
	}
}

// Handle executes the update user use case.
// The operation runs within a transaction, and domain events are dispatched
// before commit, allowing event handlers to participate in the same transaction.
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

		if err := user.UpdateProfile(name); err != nil {
			return fmt.Errorf("updating profile: %w", err)
		}

		if err := h.repo.Save(ctx, user); err != nil {
			return fmt.Errorf("saving user: %w", err)
		}

		if err := h.publisher.Publish(ctx, user.PopDomainEvents()...); err != nil {
			return fmt.Errorf("publishing events: %w", err)
		}

		return nil
	}
	return h.txScope.Execute(ctx, fn)
}
