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
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events/contracts"
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
	Repository       domain.OrderRepository
	TransactionScope transaction.Scope
	Publisher        events.Publisher
	Subscriber       events.Subscriber
	Logger           *slog.Logger
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

	createOrderHandler := commands.NewCreateOrderHandler(cfg.Repository, cfg.TransactionScope, cfg.Publisher)
	addItemHandler := commands.NewAddItemHandler(cfg.Repository)
	removeItemHandler := commands.NewRemoveItemHandler(cfg.Repository)
	submitOrderHandler := commands.NewSubmitOrderHandler(cfg.Repository, cfg.TransactionScope, cfg.Publisher)
	cancelOrderHandler := commands.NewCancelOrderHandler(cfg.Repository, cfg.TransactionScope, cfg.Publisher)

	getOrderHandler := queries.NewGetOrderHandler(cfg.Repository)
	listUserOrdersHandler := queries.NewListUserOrdersHandler(cfg.Repository)

	if cfg.Subscriber != nil {
		userDeletedHandler := eventhandlers.NewUserDeletedHandler(cfg.Repository, logger)
		if err := cfg.Subscriber.Subscribe(contracts.UserDeletedEventType, userDeletedHandler); err != nil {
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
