# Unit of Work Pattern

## Overview

Unit of Work is a pattern that maintains a list of objects affected by a business transaction and coordinates the writing out of changes. Defined in Martin Fowler's [Patterns of Enterprise Application Architecture](https://martinfowler.com/eaaCatalog/unitOfWork.html).

## Current Design

This project **does not use Unit of Work**.
Transactions are encapsulated within individual repositories.

```go
// orders/infrastructure/persistence/spanner_repository.go
func (r *SpannerRepository) Save(ctx context.Context, order *domain.Order) error {
    _, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // Single aggregate save protected by transaction
    })
    return err
}
```

### Why This Design Works

1. **Module boundaries**: Modules are independent; inter-module communication uses events (eventual consistency)
2. **Aggregate-scoped transactions**: In DDD, the aggregate is the consistency boundary
3. **Simplicity**: Application layer doesn't need to know infrastructure details

## When Unit of Work Is Needed

When strong consistency is required across multiple repositories within the same module.

Example: Creating an Order while simultaneously updating Inventory

## Unit of Work Implementation

### Interface Definition

```go
// domain or application layer
type UnitOfWork interface {
    Execute(ctx context.Context, fn func(UoW) error) error
}

type UoW interface {
    Users() domain.UserRepository
    Orders() domain.OrderRepository
}
```

### Usage (Application Layer)

```go
func (h *TransferOrderHandler) Handle(ctx context.Context, cmd TransferOrderCommand) error {
    return h.uow.Execute(ctx, func(uow UoW) error {
        // Multiple repositories in single transaction
        order, _ := uow.Orders().FindByID(ctx, cmd.OrderID)
        order.TransferTo(cmd.NewUserID)

        oldUser, _ := uow.Users().FindByID(ctx, order.OldUserID())
        oldUser.DecrementOrderCount()

        newUser, _ := uow.Users().FindByID(ctx, cmd.NewUserID)
        newUser.IncrementOrderCount()

        // All commit or rollback together
        uow.Users().Save(ctx, oldUser)
        uow.Users().Save(ctx, newUser)
        return uow.Orders().Save(ctx, order)
    })
}
```

### Spanner Implementation

```go
// infrastructure layer
type spannerUnitOfWork struct {
    client *spanner.Client
}

func (u *spannerUnitOfWork) Execute(ctx context.Context, fn func(UoW) error) error {
    _, err := u.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        uow := &spannerUoW{txn: txn}
        return fn(uow)
    })
    return err
}

type spannerUoW struct {
    txn *spanner.ReadWriteTransaction
}

func (u *spannerUoW) Users() domain.UserRepository {
    return &txnUserRepository{txn: u.txn}  // Shared transaction
}

func (u *spannerUoW) Orders() domain.OrderRepository {
    return &txnOrderRepository{txn: u.txn}  // Shared transaction
}
```

### How Transaction Sharing Works

The key question: if `domain.UserRepository` has methods like `FindByID(ctx, id)` and `Save(ctx, entity)`, how does the transaction get passed without changing the interface?

The answer is **constructor injection** (not method parameter injection).

```go
// Transaction-aware repository implementation
// Implements domain.UserRepository but holds transaction internally
type txnUserRepository struct {
    txn *spanner.ReadWriteTransaction  // Injected at construction
}

func (r *txnUserRepository) FindByID(ctx context.Context, id types.UserID) (*domain.User, error) {
    // Uses r.txn internally - no txn parameter in method signature
    row, err := r.txn.ReadRow(ctx, "Users", spanner.Key{id.String()}, userColumns)
    // ...
}

func (r *txnUserRepository) Save(ctx context.Context, user *domain.User) error {
    // Uses r.txn internally
    return r.txn.BufferWrite([]*spanner.Mutation{...})
}
```

Compare with normal repository:

```go
// Normal repository - creates its own transaction per operation
type SpannerRepository struct {
    client *spanner.Client
}

func (r *SpannerRepository) Save(ctx context.Context, user *domain.User) error {
    _, err := r.client.ReadWriteTransaction(ctx, func(...) error {
        // Transaction scoped to this single operation
    })
    return err
}
```

| Aspect             | Normal Repository          | UoW Repository              |
| ------------------ | -------------------------- | --------------------------- |
| Transaction source | Creates own per operation  | Injected at construction    |
| Scope              | Single aggregate           | Shared across UoW           |
| Interface          | `domain.UserRepository`    | Same interface              |

The domain interface remains clean (no infrastructure concerns leaking). The infrastructure layer provides different implementations depending on whether UoW is used.

### Instance Lifecycle

With constructor injection, repository instances are created per transaction:

```go
func (u *spannerUnitOfWork) Execute(ctx context.Context, fn func(UoW) error) error {
    _, err := u.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
        // New spannerUoW created for each Execute() call
        uow := &spannerUoW{txn: txn}
        return fn(uow)
    })
    return err
}

func (u *spannerUoW) Users() domain.UserRepository {
    // New repository instance bound to this transaction
    return &txnUserRepository{txn: u.txn}
}
```

| Instance             | Lifetime             | Created when      |
| -------------------- | -------------------- | ----------------- |
| `spannerUnitOfWork`  | Application lifetime | DI at startup     |
| `spannerUoW`         | Per transaction      | `Execute()` call  |
| `txnUserRepository`  | Per transaction      | `Users()` call    |

After the transaction completes (commit or rollback), `spannerUoW` and `txnUserRepository` are discarded (eligible for GC). These are lightweight structs, so the per-transaction allocation overhead is negligible.

## Testing

Unit of Work adds some test setup overhead but enables verification of cross-repository coordination.

### Test Helper

Create a test-specific UoW that executes without actual transactions:

```go
type testUnitOfWork struct {
    users  domain.UserRepository
    orders domain.OrderRepository
}

func NewTestUoW(users domain.UserRepository, orders domain.OrderRepository) UnitOfWork {
    return &testUnitOfWork{users: users, orders: orders}
}

func (u *testUnitOfWork) Execute(ctx context.Context, fn func(UoW) error) error {
    return fn(u)  // Execute directly without transaction wrapper
}

func (u *testUnitOfWork) Users() domain.UserRepository  { return u.users }
func (u *testUnitOfWork) Orders() domain.OrderRepository { return u.orders }
```

### Verifying Operation Sequence

Track repository calls to verify operations happen in correct order:

```go
func TestTransferOrder_OperationSequence(t *testing.T) {
    var calls []string

    mockUsers := &MockUserRepository{
        FindByIDFn: func(ctx context.Context, id UserID) (*User, error) {
            calls = append(calls, "users.FindByID:"+id.String())
            return &User{}, nil
        },
        SaveFn: func(ctx context.Context, u *User) error {
            calls = append(calls, "users.Save:"+u.ID().String())
            return nil
        },
    }

    mockOrders := &MockOrderRepository{
        FindByIDFn: func(ctx context.Context, id OrderID) (*Order, error) {
            calls = append(calls, "orders.FindByID")
            return &Order{}, nil
        },
        SaveFn: func(ctx context.Context, o *Order) error {
            calls = append(calls, "orders.Save")
            return nil
        },
    }

    uow := NewTestUoW(mockUsers, mockOrders)
    handler := NewTransferOrderHandler(uow)

    err := handler.Handle(ctx, TransferOrderCommand{OrderID: "order-1", NewUserID: "new-user"})

    require.NoError(t, err)
    assert.Equal(t, []string{
        "orders.FindByID",
        "users.FindByID:old-user",
        "users.FindByID:new-user",
        "users.Save:old-user",
        "users.Save:new-user",
        "orders.Save",
    }, calls)
}
```

### What Can Be Verified

| Verification Target | How to Test |
| ------------------- | ----------- |
| Call sequence | Record calls in a slice, assert order |
| All repos in same UoW | Assert `Execute` is called exactly once |
| Rollback on error | Return error from mock, verify subsequent Saves are not called |
| Correct arguments | Capture arguments in mock, assert values |
| Atomicity guarantee | Structurally guaranteed - all operations inside `Execute` callback |

### Comparison: Testing Without UoW

Without UoW, each repository is injected separately:

```go
// Simple but cannot verify cross-repository atomicity
handler := NewHandler(mockUserRepo, mockOrderRepo)
```

With UoW, the `Execute` scope structurally guarantees all operations occur within the same transaction boundary - this is verifiable by checking that `Execute` is called once and all repository operations happen inside its callback.

## Comparison

| Aspect           | Repository-scoped TX (Current) | Unit of Work                     |
| ---------------- | ------------------------------ | -------------------------------- |
| Scope            | Single aggregate               | Multiple aggregates/repositories |
| Complexity       | Simple                         | Moderate                         |
| Use case         | Independent operations         | Cross-cutting operations         |
| Infra dependency | Hidden in repository           | Abstracted at application layer  |

## References

- [P of EAA: Unit of Work](https://martinfowler.com/eaaCatalog/unitOfWork.html)
