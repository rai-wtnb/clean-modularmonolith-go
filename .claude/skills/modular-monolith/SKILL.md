---
name: modular-monolith
description: Guide for modular monolith architecture in Go. Use when designing module boundaries, organizing code by business domains, establishing inter-module communication patterns, or deciding between monolith and microservices. Covers module structure, public APIs, event-driven communication, and migration paths.
---

# Modular Monolith Architecture for Go

A modular monolith is a single deployable application where code is organized into self-contained, loosely coupled modules based on business domains.

## When This Applies

- Designing new Go applications
- Refactoring existing monoliths
- Evaluating architecture patterns
- Planning microservices migration paths

## Core Principles

### 1. Module = Business Domain (Bounded Context)

Each module represents a bounded context from Domain-Driven Design:

```
modules/
├── users/       # Authentication, profiles, permissions
├── orders/      # Order lifecycle, status tracking
├── inventory/   # Stock, warehousing, SKUs
├── payments/    # Payment processing, refunds
└── notifications/ # Email, SMS, push notifications
```

### 2. Module Independence

Modules should be:
- Independently developable by separate teams
- Testable in isolation
- Potentially extractable to microservices later

### 3. Explicit Module Boundaries

Module communication happens only through:
- Well-defined public APIs (Go interfaces)
- Shared events/messages
- **Never** through direct database access across modules

## Recommended Directory Structure

```
project/
├── go.work                    # Workspace file (gitignored)
├── cmd/
│   └── server/
│       ├── go.mod             # Module: myapp/cmd/server
│       └── main.go            # Application entry point
├── internal/
│   └── app/                   # Application wiring
│       ├── app.go
│       └── modules.go         # Module registration
├── modules/
│   ├── users/                 # User module
│   │   ├── go.mod             # Module: myapp/modules/users
│   │   ├── module.go          # Module entry point & public interface
│   │   ├── domain/
│   │   │   ├── entity/
│   │   │   └── repository.go
│   │   ├── application/
│   │   │   ├── command/
│   │   │   └── query/
│   │   ├── infrastructure/
│   │   │   └── postgres/
│   │   └── ports/
│   │       └── http/
│   ├── orders/
│   │   ├── go.mod             # Module: myapp/modules/orders
│   │   └── ...
│   └── shared/                # Shared kernel
│       ├── go.mod             # Module: myapp/modules/shared
│       ├── events/            # Domain events
│       ├── types/             # Shared value objects
│       └── errors/            # Common errors
└── pkg/                       # Truly reusable libraries
    ├── httputil/
    └── logging/
```

## Module Public API

### The Module Interface Pattern

Each module exposes a **single interface** as its public API:

```go
// modules/users/module.go
package users

import "context"

// Module is the public API for the users module.
// All inter-module communication must go through this interface.
type Module interface {
    // User retrieval
    GetUser(ctx context.Context, id UserID) (*User, error)
    GetUserByEmail(ctx context.Context, email string) (*User, error)

    // User operations
    CreateUser(ctx context.Context, cmd CreateUserCommand) (UserID, error)
    UpdateUser(ctx context.Context, cmd UpdateUserCommand) error

    // Permission checks
    HasPermission(ctx context.Context, userID UserID, perm Permission) (bool, error)
}

// User is a read-only view exposed to other modules.
// Internal domain model may have more fields.
type User struct {
    ID        UserID
    Email     string
    Name      string
    Status    UserStatus
    CreatedAt time.Time
}

// CreateUserCommand is the input for user creation.
type CreateUserCommand struct {
    Email string
    Name  string
}
```

### Module Entry Point

```go
// modules/users/module.go
package users

// New creates and initializes the users module.
func New(deps Dependencies) (Module, error) {
    repo := postgres.NewUserRepository(deps.DB)
    svc := service.NewUserService(repo, deps.EventBus)

    return &module{
        service: svc,
    }, nil
}

// Dependencies required by the users module.
type Dependencies struct {
    DB       *sql.DB
    EventBus events.Publisher
    Logger   *slog.Logger
}

type module struct {
    service *service.UserService
}

func (m *module) GetUser(ctx context.Context, id UserID) (*User, error) {
    return m.service.GetUser(ctx, id)
}

// ... implement other interface methods
```

## Inter-Module Communication

### Pattern 1: Direct Call (Synchronous)

For queries and commands that need immediate response:

```go
// In orders module - depends on users module
type OrderService struct {
    usersModule users.Module  // Injected interface
    repo        OrderRepository
}

func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) error {
    // Call users module through interface
    user, err := s.usersModule.GetUser(ctx, cmd.UserID)
    if err != nil {
        return fmt.Errorf("get user: %w", err)
    }

    if user.Status != users.StatusActive {
        return ErrInactiveUser
    }

    // Continue with order creation...
    order := NewOrder(user.ID, cmd.Items)
    return s.repo.Save(ctx, order)
}
```

### Pattern 2: Events (Asynchronous)

For loose coupling when modules don't need immediate response:

```go
// modules/shared/events/user_events.go
package events

// UserCreated is published when a new user is created.
type UserCreated struct {
    UserID    string    `json:"user_id"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

func (UserCreated) EventName() string { return "user.created" }
```

```go
// In users module - publishing events
func (s *UserService) CreateUser(ctx context.Context, cmd CreateUserCommand) (UserID, error) {
    user, err := s.repo.Create(ctx, cmd)
    if err != nil {
        return "", err
    }

    // Publish event for other modules
    s.eventBus.Publish(ctx, events.UserCreated{
        UserID:    user.ID.String(),
        Email:     user.Email,
        CreatedAt: user.CreatedAt,
    })

    return user.ID, nil
}
```

```go
// In notifications module - subscribing to events
func (m *notificationModule) RegisterHandlers(bus events.Subscriber) {
    bus.Subscribe("user.created", m.handleUserCreated)
}

func (m *notificationModule) handleUserCreated(ctx context.Context, evt events.UserCreated) error {
    return m.emailService.SendWelcomeEmail(ctx, evt.Email)
}
```

### Pattern 3: Shared Kernel

Minimal shared types that multiple modules depend on:

```go
// modules/shared/types/money.go
package types

// Money represents a monetary value.
type Money struct {
    Amount   int64  // Cents to avoid floating point
    Currency string // ISO 4217 code
}

func (m Money) Add(other Money) (Money, error) {
    if m.Currency != other.Currency {
        return Money{}, ErrCurrencyMismatch
    }
    return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}
```

## Module Registration & Application Wiring

```go
// internal/app/app.go
package app

type App struct {
    Users    users.Module
    Orders   orders.Module
    Payments payments.Module

    eventBus *eventbus.Bus
    db       *sql.DB
}

func New(cfg Config) (*App, error) {
    db, err := sql.Open("postgres", cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("open database: %w", err)
    }

    logger := slog.Default()
    eventBus := eventbus.NewEventBus(logger)

    // Initialize modules in dependency order
    usersModule, err := users.New(users.Dependencies{
        DB:        db,
        Publisher: eventBus,
        Logger:    logger.With("module", "users"),
    })
    if err != nil {
        return nil, fmt.Errorf("init users module: %w", err)
    }

    ordersModule, err := orders.New(orders.Dependencies{
        DB:          db,
        Publisher:   eventBus,
        Subscriber:  eventBus,
        UsersModule: usersModule, // Inject users module
        Logger:      logger.With("module", "orders"),
    })
    if err != nil {
        return nil, fmt.Errorf("init orders module: %w", err)
    }

    return &App{
        Users:    usersModule,
        Orders:   ordersModule,
        eventBus: eventBus,
        db:       db,
    }, nil
}

func (a *App) Close() error {
    return a.db.Close()
}
```

## Database Strategy

### Option A: Shared Database, Separate Schemas

```sql
-- Each module owns its schema
CREATE SCHEMA users;
CREATE SCHEMA orders;
CREATE SCHEMA payments;

-- Tables in respective schemas
CREATE TABLE users.accounts (...);
CREATE TABLE users.permissions (...);
CREATE TABLE orders.orders (...);
CREATE TABLE orders.order_items (...);
```

### Option B: Logical Separation with Table Prefixes

```sql
-- Prefix tables with module name
CREATE TABLE user_accounts (...);
CREATE TABLE user_permissions (...);
CREATE TABLE order_orders (...);
CREATE TABLE order_items (...);
```

### Critical Rule: No Cross-Module Database Access

```go
// WRONG - Direct database access across modules
func (r *OrderRepo) GetOrderWithUser(ctx context.Context, id string) (*OrderWithUser, error) {
    query := `
        SELECT o.*, u.name, u.email
        FROM orders.orders o
        JOIN users.accounts u ON o.user_id = u.id  -- Violates boundary!
        WHERE o.id = $1
    `
    // ...
}

// CORRECT - Use module API
func (s *OrderService) GetOrderWithUser(ctx context.Context, id string) (*OrderWithUser, error) {
    order, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Get user through module interface
    user, err := s.usersModule.GetUser(ctx, order.UserID)
    if err != nil {
        return nil, fmt.Errorf("get user: %w", err)
    }

    return &OrderWithUser{Order: order, User: user}, nil
}
```

## Anti-Patterns to Avoid

### 1. Circular Dependencies

```
// BAD: Circular dependency
Users Module ──depends on──> Orders Module
Orders Module ──depends on──> Users Module

// GOOD: Unidirectional dependency
Users Module <──depends on── Orders Module

// GOOD: Event-based (no direct dependency)
Users Module ──publishes──> Events <──subscribes── Orders Module
```

### 2. Leaky Abstractions

```go
// BAD: Exposing internal types
type Module interface {
    GetUserRepo() *postgres.UserRepository // Leaks implementation!
    GetDB() *sql.DB                        // Leaks infrastructure!
}

// GOOD: Expose only domain concepts
type Module interface {
    GetUser(ctx context.Context, id UserID) (*User, error)
}
```

### 3. Shared Mutable State

```go
// BAD: Global state
var activeUsers = make(map[string]*User)

func GetActiveUser(id string) *User {
    return activeUsers[id]
}

// GOOD: Encapsulated state within module
type module struct {
    cache *userCache // Private to module
}

func (m *module) GetUser(ctx context.Context, id UserID) (*User, error) {
    if cached := m.cache.Get(id); cached != nil {
        return cached, nil
    }
    // ...
}
```

### 4. Too Many Module Dependencies

```go
// BAD: Module depends on too many others
type OrderModule struct {
    users       users.Module
    payments    payments.Module
    inventory   inventory.Module
    shipping    shipping.Module
    analytics   analytics.Module
    audit       audit.Module
    // Signs of poor module boundaries
}

// Consider: Are these really separate bounded contexts?
// Maybe some should be merged or communication should be event-based.
```

## Migration to Microservices

When a module is ready for extraction:

1. Module already has clean API (interface)
2. Communication already uses events
3. Database schema is isolated
4. No shared in-memory state

### Extraction Process

```go
// BEFORE: In-process module
usersModule, _ := users.New(deps)

// AFTER: Remote service client implementing same interface
type usersClient struct {
    conn *grpc.ClientConn
}

func NewUsersClient(addr string) (users.Module, error) {
    conn, err := grpc.Dial(addr)
    if err != nil {
        return nil, err
    }
    return &usersClient{conn: conn}, nil
}

func (c *usersClient) GetUser(ctx context.Context, id users.UserID) (*users.User, error) {
    // gRPC call to remote service
    resp, err := c.client.GetUser(ctx, &pb.GetUserRequest{Id: id.String()})
    if err != nil {
        return nil, err
    }
    return mapProtoToUser(resp), nil
}

// Rest of the app doesn't change - same interface!
```

## Checklist for New Modules

- [ ] Module has its own `go.mod`
- [ ] Module exposes single interface (`module.go`)
- [ ] Module owns its database tables/schema
- [ ] No imports from other modules' internal packages
- [ ] Inter-module communication via interface or events only
- [ ] Module can be tested in isolation
- [ ] Module has clear bounded context (single responsibility)
- [ ] Dependencies are injected, not created internally

## Related Skills

- `/clean-architecture` - Layer design within modules
- `/cohesion-coupling` - Module design principles
- `/go-workspace` - Multi-module development setup
- `/philosophy-software-design` - Deep module design
