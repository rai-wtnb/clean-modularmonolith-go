// Package users provides user management functionality.
// This file defines the module's public API - the single interface
// that other modules use to interact with the users bounded context.
package users

import (
	"net/http"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/commands"
	"github.com/rai/clean-modularmonolith-go/modules/users/application/queries"
	httphandler "github.com/rai/clean-modularmonolith-go/modules/users/infrastructure/http"
	"github.com/rai/clean-modularmonolith-go/modules/users/infrastructure/persistence"
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
	EventPublisher events.Publisher
}

// module implements the Module interface.
type module struct {
	createUserHandler *commands.CreateUserHandler
	updateUserHandler *commands.UpdateUserHandler
	deleteUserHandler *commands.DeleteUserHandler
	getUserHandler    *queries.GetUserHandler
	listUsersHandler  *queries.ListUsersHandler
}

// New creates a new users module with all dependencies wired.
func New(cfg Config) Module {
	repository := persistence.NewInMemoryRepository()

	// Wire up command handlers
	createUserHandler := commands.NewCreateUserHandler(repository, cfg.EventPublisher)
	updateUserHandler := commands.NewUpdateUserHandler(repository, cfg.EventPublisher)
	deleteUserHandler := commands.NewDeleteUserHandler(repository, cfg.EventPublisher)

	// Wire up query handlers
	getUserHandler := queries.NewGetUserHandler(repository)
	listUsersHandler := queries.NewListUsersHandler(repository)

	return &module{
		createUserHandler: createUserHandler,
		updateUserHandler: updateUserHandler,
		deleteUserHandler: deleteUserHandler,
		getUserHandler:    getUserHandler,
		listUsersHandler:  listUsersHandler,
	}
}

func (m *module) RegisterRoutes(mux *http.ServeMux) {
	httphandler.RegisterRoutes(mux, m.createUserHandler, m.updateUserHandler, m.deleteUserHandler, m.getUserHandler, m.listUsersHandler)
}
