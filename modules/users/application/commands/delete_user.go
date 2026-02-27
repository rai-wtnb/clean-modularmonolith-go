package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// DeleteUserCommand represents the intent to delete a user.
type DeleteUserCommand struct {
	UserID string
}

// DeleteUserHandler handles the DeleteUserCommand.
type DeleteUserHandler struct {
	repo           domain.UserRepository
	txScope        transaction.TransactionScope
	eventPublisher events.Publisher
}

func NewDeleteUserHandler(repo domain.UserRepository, txScope transaction.TransactionScope, eventPublisher events.Publisher) *DeleteUserHandler {
	return &DeleteUserHandler{
		repo:           repo,
		txScope:        txScope,
		eventPublisher: eventPublisher,
	}
}

// Handle executes the delete user use case.
// The operation runs within a transaction, and domain events are dispatched
// before commit, allowing event handlers to participate in the same transaction.
func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
	userID, err := domain.ParseUserID(cmd.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	fn := func(ctx context.Context) error {
		user, err := h.repo.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("finding user: %w", err)
		}

		if err := user.Delete(); err != nil {
			return fmt.Errorf("deleting user: %w", err)
		}

		if err := h.repo.Save(ctx, user); err != nil {
			return fmt.Errorf("saving user: %w", err)
		}

		if err := h.eventPublisher.Publish(ctx, user.PopDomainEvents()...); err != nil {
			return fmt.Errorf("publishing events: %w", err)
		}

		return nil
	}
	return h.txScope.Execute(ctx, fn)
}
