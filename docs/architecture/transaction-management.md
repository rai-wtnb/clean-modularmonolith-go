# Transaction Management

## Status

Accepted (2026-04-01)

## Context

This project uses Cloud Spanner as its database and manages transactions through `context.Context` propagation. The original design allowed `Write` to create a standalone read-write transaction when none existed in the context. This fallback silently broke atomicity when a repository method was accidentally called outside a transaction scope, creating an independent transaction instead of failing.

The decision was made to remove this fallback and require all writes to go through an explicit `transaction.Scope`.

## Decision

### Write operations require an explicit transaction scope

`Write` only joins an existing read-write transaction from the context. If no transaction is present, it returns an error. All write paths must go through `ReadWriteTransactionScope`.

### Read operations work both inside and outside a scope

`SingleRead` and `ConsistentRead` join an existing transaction if one is active, or create a standalone read transaction otherwise. This is safe because standalone reads return a point-in-time snapshot with no data integrity risk.

## Architecture

### Layers

```
Application Layer          transaction.Scope (port)
    |                      transaction.ScopeWithDomainEvent (port)
    |
    v
Infrastructure Layer       ReadWriteTransactionScope (adapter)
                           ReadOnlyTransactionScope (adapter)
                           Write, SingleRead, ConsistentRead (helpers)
```

- **Port** (`modules/shared/transaction/`): `Scope` and `ScopeWithDomainEvent` interfaces. Application layer depends on these.
- **Adapter** (`internal/platform/spanner/`): Spanner-specific implementations and data-access helpers.
- **Composition root** (`cmd/server/`): Wires adapters to ports.

### Transaction Flow

```
Command Handler
    +-> ScopeWithDomainEvent.ExecuteWithPublish(ctx, fn)
         +-> ReadWriteTransactionScope.Execute(ctx, fn)
              |-- Starts Spanner ReadWriteTransaction
              |-- Embeds tx in context via withReadWriteTx(ctx, tx)
              +-> fn(txCtx)
                   |-- repo.Save(ctx, entity)
                   |    +-> Write(ctx, stmts...)
                   |         +-> readWriteTxFromContext(ctx) -> joins tx
                   |-- repo.FindByID(ctx, id)
                   |    +-> SingleRead(ctx, ...) -> joins tx (read-your-writes)
                   +-> events.Add(ctx, event)
                        +-> Collected, published pre-commit
```

### Context-based Transaction Propagation

Transactions are stored in `context.Context` using unexported keys:

| Key | Stored type | Set by |
|-----|------------|--------|
| `readWriteTxKey{}` | `*spanner.ReadWriteTransaction` | `ReadWriteTransactionScope.Execute` |
| `readOnlyTxKey{}` | `*spanner.ReadOnlyTransaction` | `ReadOnlyTransactionScope.Execute` |

Extraction functions:

| Function | Checks | Used by |
|----------|--------|---------|
| `readWriteTxFromContext` | RW key only | `Write` |
| `readOnlyTxFromContext` | RO key only | `Write` (guard) |
| `readTransactionFromContext` | RW first, then RO | `SingleRead`, `ConsistentRead` |

`readTransactionFromContext` checks RW before RO so that reads within a write transaction get read-your-writes consistency.

### Nesting Prevention

Cloud Spanner does not support nested transactions. `withReadWriteTx` and `withReadOnlyTx` return `ErrNestedTransaction` if any transaction already exists in the context. Scope implementations handle this by joining the existing transaction instead of creating a new one:

```go
func (s *ReadWriteTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
    if _, ok := readWriteTxFromContext(ctx); ok {
        return fn(ctx) // Join existing transaction
    }
    // ... create new transaction
}
```

## Data-access Helper Semantics

### Write (join-only)

```go
func Write(ctx context.Context, stmts ...spanner.Statement) error
```

- RW tx in context: joins it
- RO tx in context: returns `ErrWriteInReadOnlyScope`
- No tx in context: returns `ErrNoReadWriteTransaction`

Single statement uses `tx.Update`; multiple statements use `tx.BatchUpdate` (single RPC).

### SingleRead (REQUIRED propagation)

```go
func SingleRead[T any](ctx context.Context, client *spanner.Client, logger *slog.Logger, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error)
```

- Any tx in context: joins it
- No tx in context: falls back to `client.Single()` (cheapest one-shot read)

### ConsistentRead (REQUIRED propagation)

```go
func ConsistentRead[T any](ctx context.Context, client *spanner.Client, logger *slog.Logger, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error)
```

- Any tx in context: joins it
- No tx in context: creates a `ReadOnlyTransaction` for snapshot consistency

## CQRS Transaction Patterns

### Commands: ScopeWithDomainEvent (required)

```go
func (h *CreateUserHandler) Handle(ctx context.Context, cmd CreateUserCommand) (string, error) {
    return transaction.ExecuteWithPublishResult(ctx, h.txScope, func(ctx context.Context) (string, error) {
        // All reads and writes here join the same RW transaction
        exists, _ := h.repo.Exists(ctx, email)
        user := domain.NewUser(ctx, email, name) // events.Add(ctx, event) inside
        h.repo.Save(ctx, user)                   // Write(ctx, stmt) inside
        return user.ID().String(), nil
    })
}
```

### Queries: ReadOnlyTransactionScope or no scope

```go
// Multiple reads needing snapshot consistency -> use ReadOnlyTransactionScope
func (h *ListUsersHandler) Handle(ctx context.Context, query ListUsersQuery) (*UserListDTO, error) {
    return transaction.ExecuteWithResult(ctx, h.txScope, func(ctx context.Context) (*UserListDTO, error) {
        users, total, _ := h.repo.FindAll(ctx, offset, limit) // COUNT + SELECT in same snapshot
        return toDTO(users, total), nil
    })
}

// Single read -> no scope needed (SingleRead falls back to client.Single())
func (h *GetUserHandler) Handle(ctx context.Context, query GetUserQuery) (*UserDTO, error) {
    user, err := h.repo.FindByID(ctx, query.ID) // SingleRead creates its own one-shot read
    return toDTO(user), err
}
```

## Enforcement

Three layers enforce these rules:

| Layer | Mechanism | What it prevents |
|-------|-----------|-----------------|
| `spannercheck` linter | Static analysis | Direct use of `client.Apply`, `client.Single`, `client.ReadWriteTransaction`, `tx.BufferWrite` in persistence packages |
| Transaction scopes | Context propagation | Missing transaction boundaries (scopes embed tx in context) |
| `Write` error return | Runtime guard | Writes outside a scope (`ErrNoReadWriteTransaction`), writes inside a read-only scope (`ErrWriteInReadOnlyScope`) |

## Rationale

### Why error return instead of panic?

Although a missing scope is a programming error (incorrect wiring), `Write` returns a sentinel error (`ErrNoReadWriteTransaction`, `ErrWriteInReadOnlyScope`) rather than panicking. This allows callers to propagate the error through the normal error chain, making it visible in logs and test output without crashing the process. Repository methods already return `error`, so the misuse surfaces naturally as a failed operation.

### Why asymmetric rules for reads and writes?

Standalone reads are safe: they return a point-in-time snapshot with no side effects. Standalone writes are dangerous: they create an independent transaction that breaks atomicity with other operations in the same business flow and bypasses domain event collection.

### Why context-based propagation instead of explicit tx parameter?

- Repository interfaces (`domain.UserRepository`) stay clean with no infrastructure types
- Transaction boundaries are decided by the application layer (command/query handlers), not the infrastructure layer (repositories)
- Repositories are reusable across different transactional contexts without code changes

## See Also

- [Domain Events](domain-events.md)
- [Domain Event Handler Constraints](../rules/domain-event-handlers.md)
- [Unit of Work](../patterns/unit-of-work.md)
- [Module Boundaries](../rules/module-boundaries.md)
