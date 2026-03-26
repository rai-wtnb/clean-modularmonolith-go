# Lightweight Domain Events Pattern

This document describes the lightweight domain events implementation pattern used in this codebase. It enables event handlers to participate in the same database transaction as the originating command, ensuring atomic consistency across module boundaries.

## Overview

### Problem

In a modular monolith, modules communicate via domain events. However, when an event handler performs database operations, those operations may need to be part of the same transaction as the original command:

```
  repo.Save()  →  [COMMIT]  →  publisher.Publish()  →  handler.Handle()
                     ↑
         Transaction boundary here - handler runs OUTSIDE transaction
```

If the handler fails, the original change is already committed, leading to inconsistent state.

### Solution

```
  txScope.Execute() {
      aggregate.BusinessMethod()   // Events collected internally
      repo.Save()                  // Same transaction
      eventBus.Publish()           // Handlers execute immediately in same transaction
  } // COMMIT - all or nothing
```

## Architecture

The implementation combines three patterns:

1. **Context-collected Events**: Aggregates add domain events to a context-bound collector during business operations
2. **Transaction-scoped Event Bus**: Events trigger handlers synchronously within a transaction
3. **Context-based Transaction Propagation**: Transaction is embedded in context for repositories

```
┌──────────────────────────────────────────────────────────────────────────┐
│                       ScopeWithDomainEvent                               │
│  ┌─────────────────┐                    ┌───────────────────────────┐   │
│  │   Aggregate     │  events.Add(ctx)   │       EventBus            │   │
│  │                 │ ─────────────────► │                           │   │
│  │  Delete(ctx)    │  (ctx collector)   │  Publish(ctx) drains      │   │
│  └─────────────────┘                    │  collector, dispatches    │   │
│         │                               │  handlers in parallel     │   │
│         │ Save()                        │            │              │   │
│         ▼                               │            ▼              │   │
│  ┌─────────────────┐                    │  ┌─────────────────────┐  │   │
│  │  Repository     │◄───────────────────│  │  Handler A          │  │   │
│  │  (TxFromCtx)    │                    │  │  (same ctx = tx)    │  │   │
│  └─────────────────┘                    │  └─────────────────────┘  │   │
│                                         │            │              │   │
│                                         │            ▼              │   │
│                                         │  ┌─────────────────────┐  │   │
│                                         │  │  Handler B          │  │   │
│                                         │  └─────────────────────┘  │   │
│                                         └───────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Context-based Event Collection

Aggregates call `events.Add(ctx, event)` directly in business methods. No embedded base struct is needed:

```go
// modules/shared/events/context.go
func Add(ctx context.Context, evts ...Event)                                         // Adds events to collector (public)
func CaptureEvents(ctx context.Context, fn func(ctx context.Context) error) ([]Event, error) // Test helper

// Internal (used by ScopeWithDomainEvent):
// newContext(ctx) — initializes fresh collector
// collect(ctx)   — drains and returns all events
```

Usage in aggregate:

```go
type User struct {
    id     UserID
    status Status
}

func (u *User) Delete(ctx context.Context) error {
    u.status = StatusDeleted
    events.Add(ctx, newUserDeletedEvent(u.id))  // Add to ctx collector
    return nil
}
```

### 2. transaction.Scope

Base interface for transaction lifecycle (`modules/shared/transaction/scope.go`):

```go
type Scope interface {
    Execute(ctx context.Context, fn func(ctx context.Context) error) error
}
```

### 3. transaction.ScopeWithDomainEvent

Interface defined in `modules/shared/transaction/event_scope.go`, implementation in `modules/shared/events/scope.go`:

```go
// modules/shared/transaction/event_scope.go — interface
type ScopeWithDomainEvent interface {
    ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error
}

// modules/shared/events/scope.go — factory and implementation
func NewScopeWithDomainEvent(inner transaction.Scope, publisher Publisher) transaction.ScopeWithDomainEvent
```

The implementation:

```go
func (s *scopeWithDomainEventImpl) ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error {
    return s.inner.Execute(ctx, func(ctx context.Context) error {
        ctx = events.NewContext(ctx)  // Fresh collector per invocation (Spanner retry safe)
        if err := fn(ctx); err != nil {
            return err
        }
        return s.publisher.Publish(ctx)  // Publish after fn succeeds, before commit
    })
}
```

### 4. Transaction-aware Repository

Repositories extract the transaction from context to participate in the active transaction. The context-extraction functions are private to the `internal/platform/spanner` package:

```go
func (r *SpannerRepository) Save(ctx context.Context, user *domain.User) error {
    if txn, ok := readWriteTxFromContext(ctx); ok {
        return txn.BufferWrite(mutations)  // Use existing transaction
    }
    _, err := r.client.Apply(ctx, mutations)  // Standalone write
    return err
}
```

### 5. EventBus

Implements both `events.Publisher` and `events.Subscriber`. `Publish(ctx, evts)` dispatches events synchronously to registered handlers. The **drain loop** (collecting cascading events from handlers) lives in `ScopeWithDomainEvent.ExecuteWithPublish`, not in the EventBus itself:

```go
// internal/platform/eventbus/eventbus.go
func (b *EventBus) Publish(ctx context.Context, evts []events.Event) error {
    for _, event := range evts {
        if err := b.processEvent(ctx, event); err != nil {
            return err
        }
    }
    return nil
}

// modules/shared/events/scope.go — drain loop lives here
func (s *scopeWithDomainEventImpl) ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error {
    return s.inner.Execute(ctx, func(ctx context.Context) error {
        ctx = newContext(ctx)
        if err := fn(ctx); err != nil {
            return err
        }
        for evts := collect(ctx); len(evts) > 0; evts = collect(ctx) {
            if err := s.publisher.Publish(ctx, evts); err != nil {
                return err
            }
        }
        return nil
    })
}
```

Handlers for the same event type run in parallel (`errgroup`). Depth is tracked per-context (max 10 levels), so concurrent transactions are isolated and infinite loops are prevented.

### 6. Command Handler Integration

Command handlers depend on `ScopeWithDomainEvent` — no publisher dependency needed:

```go
type DeleteUserHandler struct {
    repo    domain.UserRepository
    txScope transaction.ScopeWithDomainEvent
}

func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
    userID, _ := domain.ParseUserID(cmd.UserID)

    return h.txScope.ExecuteWithPublish(ctx, func(ctx context.Context) error {
        // 1. Load aggregate
        user, err := h.repo.FindByID(ctx, userID)
        if err != nil {
            return err
        }

        // 2. Execute business logic (adds event to ctx collector)
        if err := user.Delete(ctx); err != nil {
            return err
        }

        // 3. Persist aggregate
        return h.repo.Save(ctx, user)
        // ScopeWithDomainEvent publishes events after this fn returns nil
    })
}
```

## Cloud Spanner Considerations

When using Cloud Spanner as the transactional store, several characteristics require careful handling.

### 1. Transaction Retry and External Side Effects

Spanner's `ReadWriteTransaction` automatically retries the callback function on `Aborted` errors (e.g., lock contention). This has critical implications:

**DO NOT** perform irreversible side effects inside event handlers:

- Email/SMS sending
- External API calls
- Pub/Sub publishing
- File writes

These operations cannot be rolled back and will be duplicated on retry.

**Event handlers within transactions should be limited to database operations only.**

For external side effects (notifications, etc.), use a separate async pattern (Pub/Sub subscription) outside the transaction. See the `notifications` module for an example approach.

### 2. Context-based Depth Tracking

The `EventBus` tracks event processing depth via context, not instance state. This means:

- A single `EventBus` instance is shared across the application
- Each transaction context starts with depth 0
- Spanner retries automatically reset the context, so depth is correctly reset

### 3. BufferWrite vs DML (Read-Your-Writes)

Spanner's `BufferWrite` mutations are not visible until commit. If an event handler needs to read data written earlier in the same transaction, you have two options:

**Option A: Use DML instead of BufferWrite**

```go
// In repository - use DML for read-your-writes support
_, err := txn.Update(ctx, spanner.Statement{
    SQL: `UPDATE users SET status = @status WHERE id = @id`,
    Params: map[string]interface{}{
        "id":     user.ID().String(),
        "status": string(user.Status()),
    },
})
```

**Option B: Pass data through the event payload**

Include all necessary data in the event itself so handlers don't need to re-read:

```go
event := NewUserDeletedEvent(user.ID(), user.Email(), user.Name())
// Handler uses event payload directly, no DB read needed
```

### 4. Infinite Loop Prevention

Event handlers may publish new events (depth-first execution). The `EventBus` tracks call stack depth via context (max 10 levels by default).

If a handler publishes an event, its handlers execute immediately (nested). If the nesting depth exceeds the limit, an `"event processing depth exceeded"` error is returned and the transaction rolls back.

## Event Handler Guidelines

### Handlers Running Within Transactions

For handlers that participate in the originating transaction:

```go
type UserDeletedHandler struct {
    orderRepo domain.OrderRepository  // Use repository directly
    logger    *slog.Logger
}

func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    // ctx contains the transaction - repository operations join it automatically
    orders, _, err := h.orderRepo.FindByUserID(ctx, userID, 0, 1000)
    if err != nil {
        return err  // Triggers rollback of the entire transaction
    }

    for _, order := range orders {
        order.Cancel(ctx)         // Adds OrderCancelledEvent to ctx
        h.orderRepo.Save(ctx, order)  // Same transaction
    }
    return nil
    // Events added by Cancel(ctx) are drained by the outer EventBus drain loop
}
```

Key points:

- Use repository directly (not through command handlers)
- Pass `ctx` through — it contains both the transaction and the event collector
- Return errors to trigger rollback
- **No external API calls or side effects**

### Handlers Intended for Future Async Processing

Some handlers are designed to eventually run outside the transaction (e.g., sending emails). Currently, **all handlers run synchronously** within the same transaction via the in-process EventBus:

```go
// Currently runs in-process (same transaction context), but designed for future async migration
type OrderSubmittedHandler struct {
    logger *slog.Logger
}

func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
    // Currently: avoid external side effects (handler may retry with the transaction)
    h.logger.Info("sending confirmation email", slog.String("order_id", event.EventID()))
    return nil
}
```

**Future migration path**: These handlers will move to Pub/Sub subscriptions where:

- Events are published to a message queue after commit (via outbox pattern)
- Handlers process asynchronously with idempotency (event ID deduplication)
- External side effects (email, SMS) become safe

## Migration Path to Outbox Pattern

The lightweight domain events pattern is designed to migrate smoothly to a full outbox pattern:

### Current (Lightweight)

```
Transaction {
    Save aggregate
    TransactionalPublisher.Publish()  →  Handler executes in same tx
}
```

### Future (Outbox)

```
Transaction {
    Save aggregate
    OutboxPublisher.Publish()  →  Save events to outbox table
}
// Later: Change stream / polling reads outbox and publishes to Pub/Sub
```

The migration requires:

1. Replace `TransactionalPublisher` with `OutboxPublisher` implementation
2. Add outbox table and change stream / polling worker
3. Update handlers for idempotency (event ID tracking)

**No changes required** in:

- Aggregate event collection
- Command handler structure
- TransactionScope usage

## Summary

| Component                  | Responsibility                                       |
| -------------------------- | ---------------------------------------------------- |
| `events.Add(ctx, event)`   | Collect events during business operations            |
| `transaction.Scope`        | Manage transaction lifecycle                         |
| `ScopeWithDomainEvent`     | Initialize collector, execute fn, publish events     |
| Context-embedded transaction | Enable repositories to join existing transactions  |
| `EventBus`                 | Subscribe handlers and publish events synchronously  |

This pattern achieves:

- **Atomic consistency** across module boundaries
- **Clean separation** between domain logic and infrastructure
- **Testability** through interface-based design
- **Migration path** to full event sourcing / outbox pattern
