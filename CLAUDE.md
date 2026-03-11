# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build   # Build server binary to bin/server
make run     # Build and run the server
make test    # Run tests across all modules
make lint    # Run golangci-lint on all modules
make tidy    # Run go mod tidy on all modules

go test ./modules/users/...                      # Test specific module
go test -run TestUserCreate ./modules/users/...  # Run specific test
```

## Workspace Modules

- `modules/users` — User management bounded context
- `modules/orders` — Order management bounded context
- `modules/notifications` — Notification handling (event-driven)
- `modules/shared` — Shared kernel: `events`, `transaction`
- `internal/platform` — Infrastructure: event bus, HTTP server, Spanner
- `cmd/server` — Composition root

## Key Patterns

**Event collection**: Aggregates call `events.Add(ctx, event)` inside business methods. `ScopeWithDomainEvent.ExecuteWithPublish` collects and publishes events automatically after successful execution.

**Transaction scope**: `transaction.Scope` (port, in `modules/shared/transaction`) wraps business logic. `transaction.ScopeWithDomainEvent` adds automatic event publishing. Concrete implementations are in `internal/platform/spanner`.

**CQRS**: Commands use `ScopeWithDomainEvent`; queries use `transaction.Scope` (read-only) or no scope.

**Module public API**: Each module exposes only `RegisterRoutes(mux *http.ServeMux)`. Cross-module communication uses domain events via contracts in `modules/shared/events/contracts`.
