# Lightweight Domain Events Pattern

This document describes the lightweight domain events implementation pattern used in this codebase. It supports two phases of event handling:

- **Pre-commit handlers**: Run inside the transaction boundary, ensuring atomic consistency across module boundaries. Failures roll back the transaction.
- **Post-commit handlers**: Run after the transaction commits successfully. Used for external side effects (notifications, search index updates). Failures are logged but do not affect the caller.

## Overview

### Problem

In a modular monolith, modules communicate via domain events. Different handlers have different consistency requirements:

1. **Transactional handlers** (e.g., cancelling orders when a user is deleted) must participate in the same transaction as the originating command — if the handler fails, the entire operation should roll back.
2. **Side-effect handlers** (e.g., sending emails, updating search indices) must run *after* the transaction commits — they should not block or roll back the originating transaction, and must not execute inside a retryable transaction (Spanner retries would duplicate side effects).

### Solution

```
  txScope.ExecuteWithPublish() {
      aggregate.BusinessMethod()   // Events collected internally
      repo.Save()                  // Same transaction
      eventBus.Publish()           // Pre-commit handlers execute in same transaction
  } // COMMIT - all or nothing
  // After commit:
  eventBus.PublishPostCommit()     // Post-commit handlers run asynchronously
```

## Architecture

The implementation combines four patterns:

1. **Context-collected Events**: Aggregates add domain events to a context-bound collector during business operations
2. **Pre-commit Event Bus**: Events trigger handlers synchronously within a transaction
3. **Post-commit Event Bus**: Events are dispatched to handlers after a successful commit
4. **Context-based Transaction Propagation**: Transaction is embedded in context for repositories

```
┌──────────────────────────────────────────────────────────────────────────┐
│                       ScopeWithDomainEvent                               │
│  ┌─────────────────┐                    ┌───────────────────────────┐   │
│  │   Aggregate     │  events.Add(ctx)   │       EventBus            │   │
│  │                 │ ─────────────────► │                           │   │
│  │  Delete(ctx)    │  (ctx collector)   │  Publish(ctx) dispatches  │   │
│  └─────────────────┘                    │  collected events to      │   │
│         │                               │  pre-commit handlers      │   │
│         │ Save()                        │            │              │   │
│         ▼                               │            ▼              │   │
│  ┌─────────────────┐                    │  ┌─────────────────────┐  │   │
│  │  Repository     │◄───────────────────│  │  Pre-commit Handler │  │   │
│  │  (TxFromCtx)    │                    │  │  (same ctx = tx)    │  │   │
│  └─────────────────┘                    │  └─────────────────────┘  │   │
│                                         └───────────────────────────┘   │
│  ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ COMMIT ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─   │
│                                         ┌───────────────────────────┐   │
│                                         │  PublishPostCommit()      │   │
│                                         │  (detached goroutine)     │   │
│                                         │            │              │   │
│                                         │            ▼              │   │
│                                         │  ┌─────────────────────┐  │   │
│                                         │  │ Post-commit Handler │  │   │
│                                         │  │ (email, ES index)   │  │   │
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
// collect(ctx)   — atomically returns and resets all events
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
func NewScopeWithDomainEvent(inner transaction.Scope, publisher Publisher, postCommitPublisher PostCommitPublisher) transaction.ScopeWithDomainEvent
```

The implementation orchestrates two phases:

```go
func (s *scopeWithDomainEventImpl) ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error {
    var publishedEvents []Event

    innerFn := func(ctx context.Context) error {
        if !isNested {
            acc = &postCommitAccumulator{}       // fresh on each retry
            ctx = contextWithAccumulator(ctx, acc)
        } else {
            acc, _ = accumulatorFromContext(ctx)  // reuse parent's
        }
        ctx = newContext(ctx)
        if err := fn(ctx); err != nil {
            return err
        }

        // Phase 1: Pre-commit — collect and publish events inside the transaction.
        evts := collect(ctx)
        if len(evts) == 0 {
            return nil
        }
        acc.add(evts)
        if err := s.publisher.Publish(ctx, evts); err != nil {
            return err
        }
        return nil
    }
    if err := s.inner.Execute(ctx, innerFn); err != nil {
        return err
    }

    // Phase 2: Post-commit — only the outermost scope fires handlers
    // after the actual transaction commit.
    if !isNested && s.postCommitPublisher != nil {
        if allEvents := acc.drain(); len(allEvents) > 0 {
            s.postCommitPublisher.PublishPostCommit(ctx, allEvents)
        }
    }
    return nil
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

Implements four interfaces: `events.Publisher`, `events.Subscriber`, `events.PostCommitPublisher`, and `events.PostCommitSubscriber`.

**Pre-commit**: `Publish(ctx, evts)` dispatches events synchronously to registered pre-commit handlers. Each handler that needs to emit its own events must use its own `ExecuteWithPublish` scope, which joins the existing transaction and contributes events to the outermost scope's post-commit accumulator.

**Post-commit**: `PublishPostCommit(ctx, evts)` dispatches events to post-commit handlers asynchronously in a separate goroutine with a detached context (not cancelled when the caller's HTTP request completes). Errors are logged but not propagated.

```go
// internal/platform/eventbus/eventbus.go

// Pre-commit: synchronous, errors propagated (→ rollback)
func (b *EventBus) Publish(ctx context.Context, evts []events.Event) error {
    for _, event := range evts {
        if err := b.processEvent(ctx, event); err != nil {
            return err
        }
    }
    return nil
}

// Post-commit: asynchronous, errors logged only (best-effort)
func (b *EventBus) PublishPostCommit(ctx context.Context, evts []events.Event) {
    detachedCtx := detachContext(ctx) // carries trace span, not cancellation
    go func() {
        for _, event := range evts {
            b.processPostCommitEvent(detachedCtx, event)
        }
    }()
}
```

Pre-commit depth is tracked per-context (max 10 levels), so concurrent transactions are isolated and infinite loops are prevented.

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

**DO NOT** perform irreversible side effects inside **pre-commit** event handlers:

- Email/SMS sending
- External API calls
- Pub/Sub publishing
- File writes

These operations cannot be rolled back and will be duplicated on retry.

**Pre-commit event handlers should be limited to database operations only.**

For external side effects, use **post-commit handlers** via `SubscribePostCommit`. These run after the transaction commits successfully and are not subject to Spanner retries. See the `notifications` module (`OrderSubmittedHandler`) and the `users` module (`UserIndexer` for Elasticsearch sync) for examples.

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

### 4. Nested Scope and Depth Protection

Pre-commit handlers that need to emit their own domain events must use their own `ExecuteWithPublish` scope. The nested scope joins the existing transaction and contributes its events to the outermost scope's post-commit accumulator — only the outermost scope fires `PostCommitPublish` after the actual commit.

The `EventBus` tracks call stack depth via context (max 10 levels by default). If the nesting depth exceeds the limit, an `"event processing depth exceeded"` error is returned and the transaction rolls back.

## Event Handler Guidelines

### Pre-commit Handlers (Transactional)

For handlers that participate in the originating transaction. Registered via `Subscribe()`:

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
    // Events added by Cancel(ctx) must be published via the handler's own
    // ExecuteWithPublish scope, which joins the outer transaction.
}
```

Key points:

- Use repository directly (not through command handlers)
- Pass `ctx` through — it contains both the transaction and the event collector
- Return errors to trigger rollback
- **No external API calls or side effects** (Spanner may retry)

### Post-commit Handlers (After Transaction)

For handlers that perform external side effects. Registered via `SubscribePostCommit()`. These run after the transaction commits successfully in a separate goroutine with a detached context:

```go
// Runs AFTER the transaction commits — safe for external side effects
type OrderSubmittedHandler struct {
    sender *NotificationSender
}

func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
    e, ok := event.(orderevents.OrderSubmittedEvent)
    if !ok {
        return fmt.Errorf("unexpected event type: %T", event)
    }
    return h.sender.SendOrderConfirmation(e.OrderID)
}
```

Key points:

- External side effects (email, search index, etc.) are safe — the transaction has already committed
- Errors are logged but do not affect the caller (best-effort delivery)
- Context is detached from the original HTTP request — not cancelled when the request completes
- Handlers share the same `events.Handler` interface as pre-commit handlers; only the subscription method differs

## Migration Path to Outbox Pattern

The lightweight domain events pattern is designed to migrate smoothly to a full outbox pattern:

### Current (Lightweight)

```
Transaction {
    Save aggregate
    Publisher.Publish()               →  Pre-commit handlers execute in same tx
}
// After commit:
PostCommitPublisher.PublishPostCommit()  →  Post-commit handlers (email, ES, etc.)
```

### Future (Outbox)

```
Transaction {
    Save aggregate
    OutboxPublisher.Publish()  →  Save events to outbox table
}
// Later: Change stream / polling reads outbox and publishes to Pub/Sub
// Pub/Sub subscribers replace both pre-commit and post-commit handlers
```

The migration requires:

1. Replace `Publisher` with `OutboxPublisher` implementation
2. Add outbox table and change stream / polling worker
3. Update handlers for idempotency (event ID tracking)
4. Migrate post-commit handlers to Pub/Sub subscribers

**No changes required** in:

- Aggregate event collection (`events.Add(ctx, event)`)
- Command handler structure (`ExecuteWithPublish`)
- TransactionScope usage

## Summary

| Component                    | Responsibility                                                       |
| ---------------------------- | -------------------------------------------------------------------- |
| `events.Add(ctx, event)`     | Collect events during business operations                            |
| `transaction.Scope`          | Manage transaction lifecycle                                         |
| `ScopeWithDomainEvent`       | Initialize collector, execute fn, publish pre-commit + post-commit   |
| Context-embedded transaction | Enable repositories to join existing transactions                    |
| `EventBus` (pre-commit)      | Subscribe and publish events synchronously inside the transaction    |
| `EventBus` (post-commit)     | Subscribe and publish events asynchronously after successful commit  |

This pattern achieves:

- **Atomic consistency** across module boundaries (pre-commit handlers)
- **Safe external side effects** after commit (post-commit handlers)
- **Clean separation** between domain logic and infrastructure
- **Testability** through interface-based design
- **Migration path** to full outbox pattern
