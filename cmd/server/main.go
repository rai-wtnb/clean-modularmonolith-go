// Package main is the entry point for the modular monolith application.
// It wires together all modules and starts the HTTP server.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rai/clean-modularmonolith-go/internal/platform/eventbus"
	"github.com/rai/clean-modularmonolith-go/internal/platform/httpserver"
	"github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
	"github.com/rai/clean-modularmonolith-go/modules/notifications"
	"github.com/rai/clean-modularmonolith-go/modules/orders"
	orderspersistence "github.com/rai/clean-modularmonolith-go/modules/orders/infrastructure/persistence"
	"github.com/rai/clean-modularmonolith-go/modules/users"
	userspersistence "github.com/rai/clean-modularmonolith-go/modules/users/infrastructure/persistence"
)

func main() {
	// Initialize logger
	slogOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	slogJsonHandler := slog.NewJSONHandler(os.Stdout, slogOptions)
	logger := slog.New(slogJsonHandler)
	slog.SetDefault(logger)

	logger.Info("starting modular monolith application")

	// Initialize Spanner client
	ctx := context.Background()
	spannerCfg := spanner.Config{
		ProjectID:  getEnv("SPANNER_PROJECT_ID", "local-project"),
		InstanceID: getEnv("SPANNER_INSTANCE_ID", "local-instance"),
		DatabaseID: getEnv("SPANNER_DATABASE_ID", "app-db"),
	}

	spannerClient, err := spanner.NewClient(ctx, spannerCfg)
	if err != nil {
		logger.Error("failed to create spanner client", slog.Any("error", err))
		os.Exit(1)
	}
	defer spannerClient.Close()

	logger.Info("connected to spanner", slog.String("dsn", spannerCfg.DSN()))

	// Initialize event bus (for inter-module communication)
	eventBus := eventbus.New(logger)

	// Initialize repositories
	usersRepo := userspersistence.NewSpannerRepository(spannerClient)
	ordersRepo := orderspersistence.NewSpannerRepository(spannerClient)

	// Initialize modules
	// Each module subscribes to events it cares about internally
	usersCfg := users.Config{
		Repository:     usersRepo,
		EventPublisher: eventBus,
	}
	usersModule := users.New(usersCfg)

	ordersCfg := orders.Config{
		Repository:      ordersRepo,
		EventPublisher:  eventBus,
		EventSubscriber: eventBus,
		Logger:          logger,
	}
	ordersModule := orders.New(ordersCfg)

	notificationCfg := notifications.Config{
		EventSubscriber: eventBus,
		Logger:          logger,
	}
	_ = notifications.New(notificationCfg)

	// Build HTTP router
	router := buildRouter(usersModule, ordersModule)

	// Apply middleware
	handler := httpserver.Middleware(router, httpserver.Recovery(logger), httpserver.Logging(logger), httpserver.CORS([]string{"*"}))

	// Create and start server
	cfg := httpserver.DefaultConfig()
	server := httpserver.New(cfg, handler, logger)

	// Graceful shutdown
	go func() {
		if err := server.Start(); err != nil {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", slog.Any("error", err))
	}

	logger.Info("server stopped")
}

// buildRouter creates the main HTTP router with all module handlers.
func buildRouter(usersModule users.Module, ordersModule orders.Module) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// API version prefix
	mux.HandleFunc("GET /api/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"1.0.0"}`))
	})

	// Each module registers its own routes (same pattern as event subscriptions)
	usersModule.RegisterRoutes(mux)
	ordersModule.RegisterRoutes(mux)

	return mux
}

// getEnv returns the value of an environment variable or a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
