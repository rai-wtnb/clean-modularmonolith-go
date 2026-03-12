package notifications

import (
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/notifications/application/eventhandlers"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// Module represents the notification module entry point.
type Module struct{}

type Config struct {
	EventSubscriber events.Subscriber
	Logger          *slog.Logger
}

// New initializes the notification module and subscribes to events.
func New(cfg Config) *Module {
	logger := cfg.Logger.With("module", "notifications")

	// Initialize event handlers
	sender := eventhandlers.NewNotificationSender(logger)
	orderSubmittedHandler := eventhandlers.NewOrderSubmittedHandler(sender)

	// Subscribe to events
	if err := cfg.EventSubscriber.Subscribe(orderSubmittedHandler.EventType(), orderSubmittedHandler); err != nil {
		logger.Error("failed to subscribe to order submitted event", slog.Any("error", err))
		// specific error handling strategy (panic vs log) depends on requirements
	}

	return &Module{}
}
