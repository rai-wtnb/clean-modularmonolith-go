# Outbox Pattern

## Overview

The Outbox Pattern ensures reliable event publishing by storing domain events in the same transaction as business data. Events are written to an "outbox" table within the aggregate's transaction, then a separate process reads and publishes them. This guarantees at-least-once delivery without distributed transactions.

## Problem

Without the Outbox pattern, two operations happen separately:

```go
func (h *Handler) Handle(ctx context.Context, cmd Command) error {
    order.Submit()

    // Step 1: Save to database
    if err := h.repo.Save(ctx, order); err != nil {
        return err  // Database failed, no event published ✓
    }

    // Step 2: Publish event
    if err := h.publisher.Publish(ctx, OrderSubmittedEvent{...}); err != nil {
        return err  // Database succeeded, but event lost! ✗
    }
    return nil
}
```

Failure scenarios:
| Scenario | Result | Problem |
|----------|--------|---------|
| DB fails, publish not called | Consistent | None |
| DB succeeds, publish fails | Inconsistent | Event lost, subscribers never notified |
| DB succeeds, app crashes before publish | Inconsistent | Event lost |

## Solution

Write events to the database in the same transaction as the aggregate:

```
┌─────────────────────────────────────────────────────────┐
│                  Single Transaction                      │
│                                                          │
│   ┌──────────────────┐    ┌──────────────────┐          │
│   │  Orders Table    │    │  Outbox Table    │          │
│   │                  │    │                  │          │
│   │  UPDATE order    │    │  INSERT event    │          │
│   │  SET status=...  │    │  (OrderSubmitted)│          │
│   └──────────────────┘    └──────────────────┘          │
│                                                          │
│   Both succeed or both fail (ACID)                       │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Outbox Processor (async)                    │
│                                                          │
│   1. Read unpublished events from Outbox                │
│   2. Publish to event bus                               │
│   3. Mark as published (or delete)                      │
└─────────────────────────────────────────────────────────┘
```

## Implementation

### Outbox Table Schema

```sql
CREATE TABLE Outbox (
    id STRING(36) NOT NULL,
    aggregate_type STRING(50) NOT NULL,
    aggregate_id STRING(36) NOT NULL,
    event_type STRING(100) NOT NULL,
    payload BYTES(MAX) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    published_at TIMESTAMP,
) PRIMARY KEY (id);
```

### Domain Event Collection

Aggregates collect events during business operations:

```go
// domain/order.go
type Order struct {
    // ... fields
    events []events.Event
}

func (o *Order) Submit() error {
    if o.status != StatusDraft {
        return ErrInvalidStatus
    }
    o.status = StatusSubmitted

    // Collect event (not published yet)
    o.events = append(o.events, events.OrderSubmittedEvent{
        OrderID:   o.id,
        UserID:    o.userID,
        Total:     o.total,
        Timestamp: time.Now(),
    })
    return nil
}

func (o *Order) Events() []events.Event {
    return o.events
}

func (o *Order) ClearEvents() {
    o.events = nil
}
```

### Repository with Outbox

```go
// infrastructure/persistence/spanner_repository.go
func (r *SpannerRepository) Save(ctx context.Context, order *domain.Order) error {
    _, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // 1. Save aggregate
        orderMutation := spanner.InsertOrUpdate("Orders", orderColumns, orderValues(order))

        // 2. Save events to outbox (same transaction)
        var mutations []*spanner.Mutation
        mutations = append(mutations, orderMutation)

        for _, event := range order.Events() {
            payload, _ := json.Marshal(event)
            outboxMutation := spanner.Insert("Outbox", outboxColumns, []interface{}{
                uuid.New().String(),
                "Order",
                order.ID().String(),
                event.EventType(),
                payload,
                spanner.CommitTimestamp,
                nil, // published_at
            })
            mutations = append(mutations, outboxMutation)
        }

        return txn.BufferWrite(mutations)
    })

    if err == nil {
        order.ClearEvents()
    }
    return err
}
```

### Outbox Processor

A background worker polls the outbox and publishes events:

```go
// infrastructure/outbox/processor.go
type Processor struct {
    client    *spanner.Client
    publisher events.Publisher
    interval  time.Duration
}

func (p *Processor) Run(ctx context.Context) {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.processOutbox(ctx)
        }
    }
}

func (p *Processor) processOutbox(ctx context.Context) error {
    // Read unpublished events
    stmt := spanner.Statement{
        SQL: `SELECT id, event_type, payload FROM Outbox
              WHERE published_at IS NULL
              ORDER BY created_at LIMIT 100`,
    }

    var eventsToPublish []outboxEvent
    iter := p.client.Single().Query(ctx, stmt)
    defer iter.Stop()
    // ... collect events

    for _, e := range eventsToPublish {
        event := deserializeEvent(e.EventType, e.Payload)

        if err := p.publisher.Publish(ctx, event); err != nil {
            // Will retry next tick
            continue
        }

        // Mark as published
        _, err := p.client.Apply(ctx, []*spanner.Mutation{
            spanner.Update("Outbox", []string{"id", "published_at"}, []interface{}{
                e.ID, spanner.CommitTimestamp,
            }),
        })
        // Handle error...
    }
    return nil
}
```

## Considerations

### Idempotency

Since the Outbox pattern guarantees at-least-once delivery, events may be delivered more than once. Subscribers must be idempotent:

```go
// Event handler must handle duplicates
func (h *NotificationHandler) Handle(ctx context.Context, event OrderSubmittedEvent) error {
    // Check if already processed
    if h.alreadySent(ctx, event.OrderID) {
        return nil  // Idempotent: skip duplicate
    }

    if err := h.sendEmail(ctx, event); err != nil {
        return err
    }

    return h.markAsSent(ctx, event.OrderID)
}
```

### Ordering

Events from the same aggregate are ordered by `created_at`. Cross-aggregate ordering is not guaranteed.

### Cleanup

Old published events should be cleaned up:

```sql
DELETE FROM Outbox
WHERE published_at IS NOT NULL
  AND published_at < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 7 DAY)
```

## Alternative: Change Data Capture (CDC)

Instead of polling, use database CDC features:

| Approach | Pros | Cons |
|----------|------|------|
| Polling | Simple, portable | Latency, load |
| CDC (Spanner Change Streams) | Real-time, efficient | Vendor-specific |

## Comparison with Direct Publishing

| Aspect | Direct Publish | Outbox Pattern |
|--------|---------------|----------------|
| Atomicity | No (two operations) | Yes (single transaction) |
| Delivery guarantee | At-most-once | At-least-once |
| Failure handling | Complex | Simple (retry) |
| Latency | Immediate | Slight delay (polling interval) |
| Complexity | Low | Moderate |

## When to Use

- **Use Outbox**: When event delivery reliability is critical (payments, orders)
- **Skip Outbox**: When occasional event loss is acceptable (analytics, logs)

## References

- [Microservices Patterns](https://microservices.io/patterns/data/transactional-outbox.html) - Chris Richardson
- [Reliable Microservices Data Exchange With the Outbox Pattern](https://debezium.io/blog/2019/02/19/reliable-microservices-data-exchange-with-the-outbox-pattern/)
