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

1. **Aggregate-collected Events**: Aggregates collect domain events internally during business operations
2. **Transaction-scoped Event Bus**: Events trigger handlers synchronously within a transaction
3. **Context-based Transaction Propagation**: Transaction is embedded in context for repositories

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           TransactionScope                               │
│  ┌─────────────────┐                    ┌───────────────────────────┐   │
│  │   Aggregate     │  PopDomainEvents() │       EventBus            │   │
│  │  ┌───────────┐  │ ─────────────────► │                           │   │
│  │  │ events[]  │  │                    │  Publish() executes       │   │
│  │  └───────────┘  │                    │  handlers immediately     │   │
│  └─────────────────┘                    │            │              │   │
│         │                               │            ▼              │   │
│         │ Save()                        │  ┌─────────────────────┐  │   │
│         ▼                               │  │  Handler A          │  │   │
│  ┌─────────────────┐                    │  │  (same ctx = tx)    │  │   │
│  │  Repository     │◄───────────────────│  └─────────────────────┘  │   │
│  │  (TxFromCtx)    │                    │            │              │   │
│  └─────────────────┘                    │            ▼              │   │
│                                         │  ┌─────────────────────┐  │   │
│                                         │  │  Handler B          │  │   │
│                                         │  └─────────────────────┘  │   │
│                                         └───────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────────┘
```

## Components

### 1. AggregateRoot Base Structure

Embed `AggregateRoot` in domain aggregates to collect events during business operations:

```go
// modules/shared/domain/aggregate.go
type AggregateRoot struct {
    domainEvents []events.Event
}

func (a *AggregateRoot) AddDomainEvent(event events.Event)
func (a *AggregateRoot) PopDomainEvents() []events.Event
```

Usage in aggregate:

```go
type User struct {
    domain.AggregateRoot  // Embed base
    id     UserID
    status Status
}

func (u *User) Delete() error {
    u.status = StatusDeleted
    u.AddDomainEvent(NewUserDeletedEvent(u.id))  // Collect event
    return nil
}
```

### 2. TransactionScope

Manages transaction lifecycle with a clean functional interface:

```go
// internal/platform/transaction/scope.go
type TransactionScope interface {
    Execute(ctx context.Context, fn func(ctx context.Context) error) error
}
```

The Spanner implementation embeds the transaction in context:

```go
func (s *SpannerTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
    _, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        ctx = WithTx(ctx, txn)  // Embed transaction in context
        return fn(ctx)
    })
    return err
}
```

### 3. Transaction-aware Repository

Repositories check context for existing transaction:

```go
func (r *SpannerRepository) Save(ctx context.Context, user *domain.User) error {
    mutations := userToMutations(user)

    if txn, ok := transaction.TxFromContext(ctx); ok {
        return txn.BufferWrite(mutations)  // Use existing transaction
    }
    _, err := r.client.Apply(ctx, mutations)  // Standalone transaction
    return err
}
```

### 4. EventBus

Implements both `events.Publisher` and `events.Subscriber`. Executes handlers synchronously when `Publish()` is called, with depth tracking via context:

```go
// internal/platform/eventbus/eventbus.go
type EventBus struct {
    mu       sync.RWMutex
    handlers map[events.EventType][]events.Handler
    logger   *slog.Logger
    maxDepth int
}

func (b *EventBus) Publish(ctx context.Context, events ...events.Event) error {
    depth := b.depthFromContext(ctx)
    if depth >= b.maxDepth {
        return ErrEventProcessingDepthExceeded
    }
    ctx = context.WithValue(ctx, depthKey{}, depth+1)

    for _, handler := range b.handlersFor(event.EventType()) {
        if err := handler.Handle(ctx, event); err != nil {
            return err  // Caller should rollback
        }
    }
    return nil
}
```

Depth is tracked per-context, so concurrent transactions are isolated. A single `EventBus` instance is shared across the application.

### 5. Command Handler Integration

Putting it all together:

```go
type DeleteUserHandler struct {
    repo      domain.UserRepository
    txScope   transaction.TransactionScope
    publisher events.Publisher  // EventBus injected here
}

func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
    return h.txScope.Execute(ctx, func(ctx context.Context) error {
        // 1. Load aggregate
        user, err := h.repo.FindByID(ctx, userID)
        if err != nil {
            return err
        }

        // 2. Execute business logic (adds event internally)
        if err := user.Delete(); err != nil {
            return err
        }

        // 3. Persist aggregate
        if err := h.repo.Save(ctx, user); err != nil {
            return err
        }

        // 4. Publish events (handlers execute immediately within same transaction)
        if err := h.publisher.Publish(ctx, user.PopDomainEvents()...); err != nil {
            return err
        }

        return nil
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

If a handler publishes an event, its handlers execute immediately (nested). If the nesting depth exceeds the limit, `ErrEventProcessingDepthExceeded` is returned and the transaction rolls back.

## Event Handler Guidelines

### Handlers Running Within Transactions

For handlers that participate in the originating transaction:

```go
type UserDeletedHandler struct {
    orderRepo domain.OrderRepository  // Use repository directly
    logger    *slog.Logger
}

func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    // ctx contains the transaction - repository operations join it
    orders, _, err := h.orderRepo.FindByUserID(ctx, userID, 0, 1000)
    if err != nil {
        return err  // Triggers rollback
    }

    for _, order := range orders {
        order.Cancel()
        h.orderRepo.Save(ctx, order)  // Same transaction
    }
    return nil
}
```

Key points:

- Use repository directly (not through command handlers)
- Pass context through - it contains the transaction
- Return errors to trigger rollback
- **No external API calls or side effects**

### Handlers Running Outside Transactions

For handlers with external side effects (email, Pub/Sub, etc.):

```go
// This handler runs via EventBus AFTER transaction commit
type OrderSubmittedHandler struct {
    logger *slog.Logger
}

func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
    // Safe to perform external operations here
    h.logger.Info("sending confirmation email", slog.String("order_id", event.AggregateID()))
    return nil
}
```

**Future migration path**: These handlers will move to Pub/Sub subscriptions where:

- Events are published to a message queue after commit
- Handlers process asynchronously with idempotency (event ID deduplication)

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

| Component          | Responsibility                                    |
| ------------------ | ------------------------------------------------- |
| `AggregateRoot`    | Collect events during business operations         |
| `TransactionScope` | Manage transaction lifecycle                      |
| `TxFromContext`    | Enable repositories to join existing transactions |
| `EventBus`         | Subscribe handlers and publish events synchronously |

This pattern achieves:

- **Atomic consistency** across module boundaries
- **Clean separation** between domain logic and infrastructure
- **Testability** through interface-based design
- **Migration path** to full event sourcing / outbox pattern
