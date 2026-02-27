# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
make build              # Build server binary to bin/server
make run                # Build and run the server
make test               # Run tests across all modules
make lint               # Run golangci-lint on all modules
make tidy               # Run go mod tidy on all modules

# Single module operations
go test ./modules/users/...                    # Test specific module
go test -run TestUserCreate ./modules/users/...  # Run specific test
```

## Architecture

This is a **Go modular monolith** using Clean Architecture principles with CQRS and event-driven inter-module communication.

### Multi-Module Workspace

Uses `go.work` to manage independent Go modules:
- `modules/users` - User management bounded context
- `modules/orders` - Order management bounded context
- `modules/notifications` - Notification handling (event-driven)
- `modules/shared` - Shared kernel (events only)
- `internal/platform` - Infrastructure (HTTP server, event bus)
- `cmd/server` - Application entry point

### Module Structure

Each business module follows this structure:
```
modules/{name}/
├── module.go              # Public API (Module interface + New factory)
├── domain/                # Entities, value objects, repository interfaces, events
├── application/
│   ├── commands/          # Write use cases (change state)
│   ├── queries/           # Read use cases (return data)
│   └── eventhandlers/     # Cross-module event handlers
└── infrastructure/
    ├── http/              # HTTP handlers
    └── persistence/       # Repository implementations
```

### Key Patterns

**Module Interface**: Each module exports a minimal `Module` interface. Modules interact through this interface and domain events, not by importing internal packages.

**Domain Encapsulation**: Aggregates use private fields with factory functions (`NewUser`) and reconstitution (`Reconstitute`) for persistence hydration.

**Event-Driven Communication**: Modules publish domain events via `events.Publisher`. Other modules subscribe via `events.Subscriber`. The `EventBus` struct implements both interfaces:
- `Subscribe`: Register handlers for event types (called at module initialization)
- `Publish`: Execute handlers synchronously within the current transaction context

**CQRS**: Commands (writes) and queries (reads) are separate handlers in the application layer. Commands typically return only IDs; queries return DTOs.

### Dependency Rule

Dependencies point inward: `infrastructure → application → domain`. The domain layer has no external dependencies. Repository interfaces are defined in domain but implemented in infrastructure.

### Inter-Module Events

Events flow: `orders` subscribes to `UserDeleted` from `users`; `notifications` subscribes to `OrderSubmitted` from `orders`.

## Conventions

- Typed IDs: Use `domain.UserID`, `domain.OrderID` instead of raw strings (each module owns its ID type)
- Value objects validate on construction (`NewEmail`, `NewName`)
- Aggregate roots control their invariants through business methods
