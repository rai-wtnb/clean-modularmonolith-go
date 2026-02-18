package notifications

import (
	"log/slog"

	"github.com/rai/clean-modularmonolith-go/modules/notifications/application/eventhandlers"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// Module represents the notification module entry point.
// Since this module primarily reacts to events, it might not have a large public API.
type Module struct{}

type Config struct {
	EventSubscriber events.Subscriber
	Logger          *slog.Logger
}

// New initializes the notification module and subscribes to events.
func New(cfg Config) *Module {
	logger := cfg.Logger.With("module", "notifications")

	// Initialize event handlers
	orderSubmittedHandler := eventhandlers.NewOrderSubmittedHandler(logger)

	// Subscribe to events
	// Note: We use the string constant from orders domain to ensure we subscribe to the correct event
	// In a looser coupling scenario, we might duplicate the string constant here to avoid importing orders domain
	if err := cfg.EventSubscriber.Subscribe(domain.OrderSubmittedEventType, orderSubmittedHandler); err != nil {
		logger.Error("failed to subscribe to order submitted event", slog.Any("error", err))
		// specific error handling strategy (panic vs log) depends on requirements
	}

	return &Module{}
}
