package notifications

import (
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/notifications/application/eventhandlers"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// Module represents the notification module entry point.
type Module struct{}

type Config struct {
	PostCommitEventSubscriber events.PostCommitSubscriber
	Logger                    *slog.Logger
}

// New initializes the notification module and subscribes to events.
// The returned cleanup function releases background resources.
func New(cfg Config) (_ *Module, cleanup func()) {
	logger := cfg.Logger.With("module", "notifications")

	// Initialize event handlers
	sender, cleanup := eventhandlers.NewNotificationSender(logger)
	orderSubmittedHandler := eventhandlers.NewOrderSubmittedHandler(sender)

	// Subscribe to events (post-commit: exterSubscribePostCommitnal side effects like email should not be in DB transactions)
	if err := cfg.PostCommitEventSubscriber.SubscribePostCommit(orderSubmittedHandler.EventType(), orderSubmittedHandler); err != nil {
		logger.Error("failed to subscribe to order submitted event", slog.Any("error", err))
		// specific error handling strategy (panic vs log) depends on requirements
	}

	return &Module{}, cleanup
}
