# Module Boundary Rules

This document defines the rules for communication between modules in the modular monolith.

## Core Principle

Modules are autonomous bounded contexts. They must be loosely coupled to enable:
- Independent development and testing
- Future extraction to microservices
- Clear ownership boundaries

## Rules

### 1. No Internal Package Imports

Modules must NOT import internal packages from other modules.

```go
// BAD: Importing internal package from another module
import (
    "github.com/example/app/modules/users/domain"           // internal!
    "github.com/example/app/modules/users/infrastructure"   // internal!
)

// GOOD: Import only the module's public API
import (
    "github.com/example/app/modules/users"  // module.go only
)
```

**Why:** Internal packages are implementation details. Depending on them creates tight coupling and prevents independent evolution.

### 2. Module Interface Only

Cross-module communication must use the public `Module` interface defined in `module.go`.

```go
// modules/users/module.go
type Module interface {
    RegisterRoutes(mux *http.ServeMux)
}
```

**Why:** The Module interface is the explicit contract. It can be versioned and maintained independently.

### 3. Domain Events for Reactions

When Module A needs to react to changes in Module B, use domain events.

```
Module B (Publisher)              Module A (Subscriber)
┌──────────────────┐              ┌──────────────────┐
│  User Deleted    │──event──────▶│  Cancel Orders   │
└──────────────────┘              └──────────────────┘
```

```go
// GOOD: Subscribe to events via the handler's self-declared EventType
cfg.Subscriber.Subscribe(handler.EventType(), handler)

// BAD: Call other module's internal methods
usersModule.DeleteUserOrders(userID)  // Tight coupling!
```

**Why:** Events enable loose coupling. Publishers don't know about subscribers.

### 4. Cross-Module Events in `domain/events/` Sub-package

Events consumed by other modules are defined in a dedicated `domain/events/` sub-package within the publishing module. Internal events stay in `domain/events.go`:

```go
// modules/users/domain/events/user_deleted.go — importable by other modules
const UserDeletedEventType events.EventType = "users.UserDeleted"
type UserDeletedEvent struct { ... }

// modules/users/domain/events.go — internal to users module
const UserCreatedEventType events.EventType = "users.UserCreated"
type UserCreatedEvent struct { ... }
```

### 5. Shared Kernel Exception

The `modules/shared` package is the only cross-cutting dependency. It contains:
- Domain event infrastructure (`events.Event`, `events.Publisher`, `events.Subscriber`)
- Transaction abstractions (`transaction.Scope`, `transaction.ScopeWithDomainEvent`)
- Idempotency utilities (`idempotent.OutboundCache`)

```go
// OK: Shared kernel imports
import "github.com/example/app/modules/shared/events"
```

**Why:** Some infrastructure must be shared to enable communication.

### 6. No Circular Dependencies

Module dependencies must form a directed acyclic graph (DAG).

```
// BAD: Circular dependency
users → orders → users

// GOOD: One-way dependencies via events
users ──event──▶ orders
```

**Why:** Circular dependencies make the system impossible to understand and test.

### 7. Typed IDs Across Boundaries

When passing IDs across module boundaries (e.g., in event contracts), use primitive types (`string`) in the contract structs. Parse to typed IDs at the receiving module boundary:

```go
// modules/users/domain/events/user_deleted.go
type UserDeletedEvent struct {
    events.BaseEvent
    UserID string  // Primitive — no module coupling
}

// In the event handler (orders module)
userID, err := domain.ParseUserID(e.UserID)  // Parse at boundary
```

**Why:** Typed IDs prevent mixing up IDs from different aggregates.

## Allowed vs Prohibited

| Action | Allowed | Prohibited |
|--------|:-------:|:----------:|
| Import `modules/xxx/module.go` | ✓ | |
| Import `modules/xxx/domain/events/` (cross-module events) | ✓ | |
| Import `modules/xxx/domain/*.go` (internal types) | | ✗ |
| Import `modules/xxx/application/*.go` | | ✗ |
| Import `modules/xxx/infrastructure/*.go` | | ✗ |
| Import `modules/shared/*` | ✓ | |
| Call Module interface methods | ✓ | |
| Subscribe to domain events | ✓ | |
| Direct database queries to other module's tables | | ✗ |

## Verification

Use Go's build constraints or linting tools to enforce these rules:

```bash
# Check for prohibited imports (example using grep)
grep -r "modules/users/domain" modules/orders/
```

## See Also

- [Dependency Rule](dependency-rule.md)
- [Domain Event Handlers](domain-event-handlers.md)
