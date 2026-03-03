package spanner

import (
	"context"

	"cloud.google.com/go/spanner"
)

// ReadWriteTransactionScope manages the lifecycle of a Spanner read-write transaction.
type ReadWriteTransactionScope struct {
	client *spanner.Client
}

// NewReadWriteTransactionScope creates a new Spanner-backed transaction scope.
// It should be called once per application startup in main.
func NewReadWriteTransactionScope(client *spanner.Client) *ReadWriteTransactionScope {
	return &ReadWriteTransactionScope{client: client}
}

// Execute runs fn within a Spanner ReadWriteTransaction.
// If a ReadWriteTransaction already exists in ctx, fn joins that transaction
// instead of creating a new one (REQUIRED propagation semantics).
// The transaction is committed if fn returns nil, rolled back otherwise.
// The ctx passed to fn contains the transaction for repositories to access via ReadWriteTxFromContext.
//
// IMPORTANT: Spanner may retry fn on Aborted errors. Therefore:
//   - fn must be idempotent
//   - fn must NOT perform external side effects (email, API calls, etc.)
//   - Any state (like TransactionalPublisher) should be created inside fn
func (s *ReadWriteTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ReadWriteTxFromContext(ctx); ok {
		return fn(ctx)
	}
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		txCtx, err := withReadWriteTx(ctx, tx)
		if err != nil {
			return err
		}
		return fn(txCtx)
	})
	return err
}

// ReadOnlyTransactionScope manages the lifecycle of a Spanner read-only transaction.
// Use this when you need consistent reads across multiple queries without writes.
type ReadOnlyTransactionScope struct {
	client *spanner.Client
}

// NewReadOnlyTransactionScope creates a new Spanner-backed read-only transaction scope.
func NewReadOnlyTransactionScope(client *spanner.Client) *ReadOnlyTransactionScope {
	return &ReadOnlyTransactionScope{client: client}
}

// Execute runs fn within a Spanner ReadOnlyTransaction.
// If a ReadTransaction (read-write or read-only) already exists in ctx, fn joins
// that transaction instead of creating a new one (REQUIRED propagation semantics).
// The ctx passed to fn contains the transaction for repositories to access via ReadTransactionFromContext.
// The transaction is closed automatically when Execute returns.
func (s *ReadOnlyTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ReadTransactionFromContext(ctx); ok {
		return fn(ctx)
	}
	tx := s.client.ReadOnlyTransaction()
	defer tx.Close()

	txCtx, err := withReadOnlyTx(ctx, tx)
	if err != nil {
		return err
	}
	return fn(txCtx)
}
