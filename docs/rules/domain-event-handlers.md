# Domain Event Handler Constraints

This document describes critical constraints that domain event subscribers (handlers) must follow.

## Handler Phases

Domain event handlers run in one of two phases, determined by how they are subscribed:

| Phase | Subscription | Example | Characteristics |
|-------|-------------|---------|-----------------|
| **Pre-commit** | `Subscribe()` | `UserDeletedHandler` in Orders | Runs inside the Spanner transaction. Errors roll back the transaction. |
| **Post-commit** | `SubscribePostCommit()` | `OrderSubmittedHandler` in Notifications, `UserIndexer` in Users | Runs after successful commit in a detached goroutine. Errors are logged, not propagated. |

## Constraints for Pre-commit Handlers

### 1. No External Side Effects

Spanner's `ReadWriteTransaction` automatically retries the callback function when `Aborted` errors occur (e.g., lock contention, deadline exceeded).

```
Command Execute
    └─> Transaction Start
         ├─> Business Logic
         ├─> Event Publish (pre-commit handlers execute)  ← May be retried!
         └─> Commit (or Retry from Start)
    └─> Post-commit handlers (not retried)
```

**Prohibited actions in pre-commit handlers:**

- Sending emails, SMS, push notifications
- Calling external APIs (payment, shipping, etc.)
- Writing to external message queues
- Logging audit events to external systems
- Any action that cannot be rolled back

**Why:** If the transaction retries, these side effects will execute multiple times. Use **post-commit handlers** (`SubscribePostCommit`) for external side effects instead.

```go
// BAD: External side effect in pre-commit handler
func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    h.emailService.Send("user-deleted@example.com", "...") // Sent on every retry!
    return h.orderRepo.CancelUserOrders(ctx, userID)
}

// GOOD: Only database operations in pre-commit handler
func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    return h.orderRepo.CancelUserOrders(ctx, userID) // Rolled back on retry
}

// GOOD: External side effects in post-commit handler (registered via SubscribePostCommit)
func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
    return h.sender.SendOrderConfirmation(orderID) // Runs after commit, not retried
}
```

### 2. Use DML, Not Mutations

Spanner's `BufferWrite` (Mutation-based writes) are **not applied until commit**. If an event handler reads data that was just written in the same transaction, it will see stale data.

```
Command Execute
    ├─> repo.Save(aggregate)        ← Uses BufferWrite (not yet visible)
    ├─> eventBus.Publish(event)
    ├─> Handler: repo.FindByID()    ← Returns OLD data!
    └─> Commit                       ← Mutations applied here
```

**Solution:** Use DML (`tx.Update`, `tx.Insert`) instead of `tx.BufferWrite` for writes that must be visible to subsequent reads within the same transaction.

```go
// BAD: Mutation-based write (invisible until commit)
func (r *SpannerRepository) Save(ctx context.Context, order *Order) error {
    m := spanner.InsertOrUpdateMap("orders", map[string]interface{}{...})
    return r.client.BufferWrite([]*spanner.Mutation{m})
}

// GOOD: DML-based write (immediately visible)
func (r *SpannerRepository) Save(ctx context.Context, order *Order) error {
    stmt := spanner.Statement{
        SQL: `INSERT OR UPDATE INTO orders (...) VALUES (...)`,
        Params: map[string]interface{}{...},
    }
    _, err := r.txn.Update(ctx, stmt)
    return err
}
```

### 3. Context-based Depth Tracking

The `EventBus` tracks event processing depth via context, not instance state. This ensures:

- Spanner retries automatically reset depth (new context)
- Concurrent transactions are isolated
- A single `EventBus` instance is safely shared

```go
// Command handlers use ScopeWithDomainEvent (not plain Scope)
type DeleteUserHandler struct {
    repo    domain.UserRepository
    txScope transaction.ScopeWithDomainEvent
}

func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
    return h.txScope.ExecuteWithPublish(ctx, func(ctx context.Context) error {
        // ... business logic — events added to ctx collector via events.Add(ctx, ...) ...
        // ScopeWithDomainEvent calls EventBus.Publish(ctx) after fn succeeds
        return nil
    })
}
```

### 4. Handler Errors Cause Transaction Rollback

If a pre-commit handler returns an error, the entire transaction is rolled back.

```go
func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    if err := h.orderRepo.CancelUserOrders(ctx, userID); err != nil {
        return err // Transaction rolls back, user deletion is also undone
    }
    return nil
}
```

**Consider:** Whether the handler failure should block the originating command. If not, register it as a post-commit handler instead.

## Constraints for Post-commit Handlers

Post-commit handlers run after the transaction commits successfully, in a separate goroutine with a detached context.

### 1. Best-effort Delivery

Post-commit handler errors are **logged but not propagated** to the caller. The originating transaction has already committed. If a handler fails, the side effect is lost unless the handler itself implements retry logic.

### 2. Idempotency Recommended

Although the current in-process `EventBus` delivers post-commit events exactly once per successful commit, designing handlers to be idempotent prepares for future migration to Pub/Sub (at-least-once delivery). Use event ID for deduplication where appropriate.

### 3. No Transaction Context

Post-commit handlers do **not** have access to the originating transaction. If they need to perform database operations, they must create their own transaction or use standalone operations.

### 4. Detached Context

The context passed to post-commit handlers is detached from the original HTTP request context. This means:

- The handler is not cancelled when the HTTP request completes
- Trace spans are preserved for observability
- Deadlines and cancellation signals from the original request are not inherited

### 5. Eventual Consistency

Post-commit handlers see data after the originating transaction commits. The system is eventually consistent for post-commit side effects.

```
Transaction Commit → PublishPostCommit() → Handler runs (sees committed data)
```

## Quick Reference

| Constraint | Pre-commit | Post-commit |
|------------|:----------:|:-----------:|
| No external side effects | **Required** | Allowed (this is the intended use) |
| Use DML (not Mutations) | **Required** | N/A (no transaction context) |
| Fresh event collector per retry | Automatic | N/A |
| Handler error = rollback | Yes | No (logged only) |
| Idempotency | Nice to have | Recommended |
| Data consistency | Strong | Eventual |
| Runs in transaction | Yes | No |

## See Also

- [Lightweight Domain Events](../patterns/lightweight-domain-events.md)
- [Unit of Work](../patterns/unit-of-work.md)
