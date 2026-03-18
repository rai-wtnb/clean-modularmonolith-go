// Package users provides user management functionality.
// This file defines the module's public API - the single interface
// that other modules use to interact with the users bounded context.
package users

import (
	"log/slog"
	"net/http"

	"github.com/rai/clean-modularmonolith-go/internal/platform/elasticsearch"
	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/eventhandlers"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/queries"
	"github.com/rai/clean-modularmonolith-go/modules/users/domain"
	httphandler "github.com/rai/clean-modularmonolith-go/modules/users/infrastructure/http"
)

// Module is the public API for the users bounded context.
// External communication: HTTP API (RegisterRoutes)
// Cross-module communication: Domain Events (subscribed internally)
type Module interface {
	// RegisterRoutes registers the module's HTTP routes to the given mux.
	RegisterRoutes(mux *http.ServeMux)
}

// Config holds the module configuration.
type Config struct {
	Repository                domain.UserRepository
	ReadWriteTransactionScope transaction.Scope
	ReadOnlyTransactionScope  transaction.Scope
	Publisher                 events.Publisher
	Subscriber                events.Subscriber
	ESClient                  elasticsearch.Client
	Logger                    *slog.Logger
}

// module implements the Module interface.
type module struct {
	createUserHandler  *commands.CreateUserHandler
	updateUserHandler  *commands.UpdateUserHandler
	deleteUserHandler  *commands.DeleteUserHandler
	getUserHandler     *queries.GetUserHandler
	listUsersHandler   *queries.ListUsersHandler
	searchUsersHandler *queries.SearchUsersHandler
}

// New creates a new users module with all dependencies wired.
func New(cfg Config) Module {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("module", "users")

	// Wrap the transaction scope with ScopeWithDomainEvent that automatically
	// collects domain events from context and publishes them after success.
	txScope := events.NewScopeWithDomainEvent(cfg.ReadWriteTransactionScope, cfg.Publisher)

	// Wire up command handlers (no publisher needed — ScopeWithDomainEvent handles it)
	createUserHandler := commands.NewCreateUserHandler(cfg.Repository, txScope)
	updateUserHandler := commands.NewUpdateUserHandler(cfg.Repository, txScope)
	deleteUserHandler := commands.NewDeleteUserHandler(cfg.Repository, txScope)

	// Wire up query handlers
	getUserHandler := queries.NewGetUserHandler(cfg.Repository)
	listUsersHandler := queries.NewListUsersHandler(cfg.Repository, cfg.ReadOnlyTransactionScope)
	searchUsersHandler := queries.NewSearchUsersHandler(cfg.ESClient)

	// Subscribe to domain events for Elasticsearch sync
	if cfg.Subscriber != nil && cfg.ESClient != nil {
		indexer := eventhandlers.NewUserIndexer(cfg.ESClient, logger)
		handlers := []events.Handler{
			eventhandlers.NewUserCreatedHandler(indexer),
			eventhandlers.NewUserUpdatedHandler(indexer),
			eventhandlers.NewUserDeletedHandler(indexer),
		}
		for _, h := range handlers {
			if err := cfg.Subscriber.Subscribe(h.EventType(), h); err != nil {
				logger.Error("failed to subscribe to event",
					slog.String("event_type", h.EventType().String()),
					slog.Any("error", err),
				)
			}
		}
	}

	return &module{
		createUserHandler:  createUserHandler,
		updateUserHandler:  updateUserHandler,
		deleteUserHandler:  deleteUserHandler,
		getUserHandler:     getUserHandler,
		listUsersHandler:   listUsersHandler,
		searchUsersHandler: searchUsersHandler,
	}
}

func (m *module) RegisterRoutes(mux *http.ServeMux) {
	httphandler.RegisterRoutes(mux, m.createUserHandler, m.updateUserHandler, m.deleteUserHandler, m.getUserHandler, m.listUsersHandler, m.searchUsersHandler)
}
