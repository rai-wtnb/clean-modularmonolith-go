// Package orders provides order management functionality.
// This is the public API for the orders bounded context.
package orders

import (
	"log/slog"
	"net/http"

	"github.com/rai/clean-modularmonolith-go/modules/orders/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/orders/application/eventhandlers"
	"github.com/rai/clean-modularmonolith-go/modules/orders/application/queries"
	"github.com/rai/clean-modularmonolith-go/modules/orders/domain"
	httphandler "github.com/rai/clean-modularmonolith-go/modules/orders/infrastructure/http"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	userdomain "github.com/rai/clean-modularmonolith-go/modules/users/domain" // FIXME: this shouldn't be imported
)

// Module is the public API for the orders bounded context.
// External communication: HTTP API (RegisterRoutes)
// Cross-module communication: Domain Events (subscribed internally)
type Module interface {
	// RegisterRoutes registers the module's HTTP routes to the given mux.
	RegisterRoutes(mux *http.ServeMux)
}

// Config holds the module configuration.
type Config struct {
	Repository      domain.OrderRepository
	EventPublisher  events.Publisher
	EventSubscriber events.Subscriber
	Logger          *slog.Logger
}

type module struct {
	createOrderHandler *commands.CreateOrderHandler
	addItemHandler     *commands.AddItemHandler
	removeItemHandler  *commands.RemoveItemHandler
	submitOrderHandler *commands.SubmitOrderHandler
	cancelOrderHandler *commands.CancelOrderHandler
	getOrderHandler    *queries.GetOrderHandler
	listUserOrders     *queries.ListUserOrdersHandler
}

// New creates a new orders module.
func New(cfg Config) Module {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("module", "orders")

	createOrderHandler := commands.NewCreateOrderHandler(cfg.Repository, cfg.EventPublisher)
	addItemHandler := commands.NewAddItemHandler(cfg.Repository)
	removeItemHandler := commands.NewRemoveItemHandler(cfg.Repository)
	submitOrderHandler := commands.NewSubmitOrderHandler(cfg.Repository, cfg.EventPublisher)
	cancelOrderHandler := commands.NewCancelOrderHandler(cfg.Repository, cfg.EventPublisher)

	getOrderHandler := queries.NewGetOrderHandler(cfg.Repository)
	listUserOrdersHandler := queries.NewListUserOrdersHandler(cfg.Repository)

	// Subscribe to cross-module events
	if cfg.EventSubscriber != nil {
		userDeletedHandler := eventhandlers.NewUserDeletedHandler(cancelOrderHandler, logger)
		if err := cfg.EventSubscriber.Subscribe(userdomain.UserDeletedEventType, userDeletedHandler); err != nil {
			logger.Error("failed to subscribe to user deleted event", slog.Any("error", err))
		}
	}

	return &module{
		createOrderHandler: createOrderHandler,
		addItemHandler:     addItemHandler,
		removeItemHandler:  removeItemHandler,
		submitOrderHandler: submitOrderHandler,
		cancelOrderHandler: cancelOrderHandler,
		getOrderHandler:    getOrderHandler,
		listUserOrders:     listUserOrdersHandler,
	}
}

func (m *module) RegisterRoutes(mux *http.ServeMux) {
	httphandler.RegisterRoutes(mux, m.createOrderHandler, m.addItemHandler, m.removeItemHandler, m.submitOrderHandler, m.cancelOrderHandler, m.getOrderHandler, m.listUserOrders)
}
