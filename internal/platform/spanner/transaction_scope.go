package spanner

import (
	"context"
	"log/slog"

	"cloud.google.com/go/spanner"
)

// ReadWriteTransactionScope manages the lifecycle of a Spanner read-write transaction.
type ReadWriteTransactionScope struct {
	client *spanner.Client
	logger *slog.Logger
}

// NewReadWriteTransactionScope creates a new Spanner-backed transaction scope.
// It should be called once per application startup in main.
func NewReadWriteTransactionScope(client *spanner.Client, logger *slog.Logger) *ReadWriteTransactionScope {
	return &ReadWriteTransactionScope{client: client, logger: logger}
}

// Execute runs fn within a Spanner ReadWriteTransaction.
// If a ReadWriteTransaction already exists in ctx, fn joins that transaction
// instead of creating a new one (REQUIRED propagation semantics).
// The transaction is committed if fn returns nil, rolled back otherwise.
// The ctx passed to fn contains the transaction for repositories to use via Write/SingleRead/ConsistentRead.
//
// IMPORTANT: Spanner may retry fn on Aborted errors. Therefore:
//   - fn must be idempotent
//   - fn must NOT perform external side effects (email, API calls, etc.)
//   - Any state (like TransactionalPublisher) should be created inside fn
func (s *ReadWriteTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := readWriteTxFromContext(ctx); ok {
		return fn(ctx)
	}

	finishLog := txLog(ctx, s.logger, TxReadWrite, "ReadWriteScope")

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		txCtx, err := withReadWriteTx(ctx, tx)
		if err != nil {
			return err
		}
		return fn(txCtx)
	})
	finishLog(err)
	return err
}

// ReadOnlyTransactionScope manages the lifecycle of a Spanner read-only transaction.
// Use this when you need consistent reads across multiple queries without writes.
type ReadOnlyTransactionScope struct {
	client *spanner.Client
	logger *slog.Logger
}

// NewReadOnlyTransactionScope creates a new Spanner-backed read-only transaction scope.
func NewReadOnlyTransactionScope(client *spanner.Client, logger *slog.Logger) *ReadOnlyTransactionScope {
	return &ReadOnlyTransactionScope{client: client, logger: logger}
}

// Execute runs fn within a Spanner ReadOnlyTransaction.
// If a ReadTransaction (read-write or read-only) already exists in ctx, fn joins
// that transaction instead of creating a new one (REQUIRED propagation semantics).
// The ctx passed to fn contains the transaction for repositories to use via SingleRead/ConsistentRead.
// The transaction is closed automatically when Execute returns.
func (s *ReadOnlyTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := readTransactionFromContext(ctx); ok {
		return fn(ctx)
	}

	finishLog := txLog(ctx, s.logger, TxReadOnly, "ReadOnlyScope")

	tx := s.client.ReadOnlyTransaction()
	defer tx.Close()

	txCtx, err := withReadOnlyTx(ctx, tx)
	if err != nil {
		finishLog(err)
		return err
	}

	err = fn(txCtx)
	finishLog(err)
	return err
}
