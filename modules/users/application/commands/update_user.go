package commands

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/internal/platform/eventbus"
	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
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
	repo            domain.UserRepository
	txScope         transaction.TransactionScope
	handlerRegistry eventbus.HandlerRegistry
}

func NewUpdateUserHandler(
	repo domain.UserRepository,
	txScope transaction.TransactionScope,
	handlerRegistry eventbus.HandlerRegistry,
) *UpdateUserHandler {
	return &UpdateUserHandler{
		repo:            repo,
		txScope:         txScope,
		handlerRegistry: handlerRegistry,
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

	return h.txScope.Execute(ctx, func(ctx context.Context) error {
		// Create event bus inside closure for Spanner retry safety
		eventBus := eventbus.NewTransactional(h.handlerRegistry, 10)

		// Load aggregate
		user, err := h.repo.FindByID(ctx, userID)
		if err != nil {
			return fmt.Errorf("finding user: %w", err)
		}

		// Execute business logic (adds UserUpdatedEvent internally)
		if err := user.UpdateProfile(name); err != nil {
			return fmt.Errorf("updating profile: %w", err)
		}

		// Persist changes
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
}
