package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/eventbus"
	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
)

// DeleteUserCommand represents the intent to delete a user.
type DeleteUserCommand struct {
	UserID string
}

// DeleteUserHandler handles the DeleteUserCommand.
type DeleteUserHandler struct {
	repo            domain.UserRepository
	txScope         transaction.TransactionScope
	handlerRegistry eventbus.HandlerRegistry
}

func NewDeleteUserHandler(
	repo domain.UserRepository,
	txScope transaction.TransactionScope,
	handlerRegistry eventbus.HandlerRegistry,
) *DeleteUserHandler {
	return &DeleteUserHandler{
		repo:            repo,
		txScope:         txScope,
		handlerRegistry: handlerRegistry,
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

	return h.txScope.Execute(ctx, func(ctx context.Context) error {
		// Create event bus inside closure for Spanner retry safety
		eventBus := eventbus.NewTransactional(h.handlerRegistry, 10)

		// 1. Load aggregate
		user, err := h.repo.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("finding user: %w", err)
		}

		// 2. Execute business logic (adds event internally)
		if err := user.Delete(); err != nil {
			return fmt.Errorf("deleting user: %w", err)
		}

		// 3. Persist aggregate
		if err := h.repo.Save(ctx, user); err != nil {
			return fmt.Errorf("saving user: %w", err)
		}

		// 4. Collect events from aggregate and publish to bus
		for _, event := range user.DomainEvents() {
			if err := eventBus.Publish(ctx, event); err != nil {
				return fmt.Errorf("publishing event: %w", err)
			}
		}
		user.ClearDomainEvents()

		// 5. Flush events (handlers run within same transaction)
		if err := eventBus.Flush(ctx); err != nil {
			return fmt.Errorf("flushing events: %w", err)
		}

		return nil
	})
}
