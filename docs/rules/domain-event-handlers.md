# Domain Event Handler Constraints

This document describes critical constraints that domain event subscribers (handlers) must follow.

## Transaction Context

Domain event handlers in this system can run in two contexts:

| Context | Example | Characteristics |
|---------|---------|-----------------|
| **Transactional** | `UserDeletedHandler` in Orders | Runs within the same Spanner transaction as the command |
| **Non-transactional** | `OrderSubmittedHandler` in Notifications | Runs outside the transaction (async-ready) |

## Constraints for Transactional Handlers

### 1. No External Side Effects

Spanner's `ReadWriteTransaction` automatically retries the callback function when `Aborted` errors occur (e.g., lock contention, deadline exceeded).

```
Command Execute
    └─> Transaction Start
         ├─> Business Logic
         ├─> Event Publish (handlers execute immediately)  ← May be retried!
         └─> Commit (or Retry from Start)
```

**Prohibited actions in transactional handlers:**

- Sending emails, SMS, push notifications
- Calling external APIs (payment, shipping, etc.)
- Writing to external message queues
- Logging audit events to external systems
- Any action that cannot be rolled back

**Why:** If the transaction retries, these side effects will execute multiple times.

```go
// BAD: External side effect in transactional handler
func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    h.emailService.Send("user-deleted@example.com", "...") // Sent on every retry!
    return h.orderRepo.CancelUserOrders(ctx, userID)
}

// GOOD: Only database operations
func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    return h.orderRepo.CancelUserOrders(ctx, userID) // Rolled back on retry
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
// EventBus is injected into command handlers
type DeleteUserHandler struct {
    repo      domain.UserRepository
    txScope   transaction.TransactionScope
    publisher events.Publisher  // EventBus
}

func (h *DeleteUserHandler) Handle(ctx context.Context, cmd DeleteUserCommand) error {
    return h.txScope.Execute(ctx, func(ctx context.Context) error {
        // ... business logic ...
        return h.publisher.Publish(ctx, events...) // Depth tracked via ctx
    })
}
```

### 4. Handler Errors Cause Transaction Rollback

If a transactional handler returns an error, the entire transaction is rolled back.

```go
func (h *UserDeletedHandler) Handle(ctx context.Context, event events.Event) error {
    if err := h.orderRepo.CancelUserOrders(ctx, userID); err != nil {
        return err // Transaction rolls back, user deletion is also undone
    }
    return nil
}
```

**Consider:** Whether the handler failure should block the originating command. If not, the handler should be non-transactional.

## Constraints for Non-Transactional Handlers

### 1. Idempotency Required

Non-transactional handlers (future Pub/Sub subscribers) may receive the same event multiple times due to:
- At-least-once delivery guarantees
- Network issues causing redelivery
- Consumer restarts

**Solution:** Use event ID for deduplication.

```go
func (h *OrderSubmittedHandler) Handle(ctx context.Context, event events.Event) error {
    // Check if already processed
    if h.processedEvents.Contains(event.EventID()) {
        return nil // Skip duplicate
    }

    // Process event
    if err := h.sendConfirmationEmail(event); err != nil {
        return err
    }

    // Mark as processed
    h.processedEvents.Add(event.EventID())
    return nil
}
```

### 2. Eventual Consistency

Non-transactional handlers see data after the originating transaction commits. The system is eventually consistent.

```
Transaction Commit → Event Published → Handler Reads (sees committed data)
```

## Quick Reference

| Constraint | Transactional | Non-Transactional |
|------------|:-------------:|:-----------------:|
| No external side effects | **Required** | Allowed |
| Use DML (not Mutations) | **Required** | N/A |
| Fresh publisher per retry | **Required** | N/A |
| Handler error = rollback | Yes | No |
| Idempotency | Nice to have | **Required** |
| Data consistency | Strong | Eventual |

## See Also

- [Lightweight Domain Events](../patterns/lightweight-domain-events.md)
- [Unit of Work](../patterns/unit-of-work.md)
