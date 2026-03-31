package spanner

import (
	"context"
	"log/slog"
	"runtime"
	"time"

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
	_, file, line, _ := runtime.Caller(1)
	caller := slog.Group("caller", slog.String("file", file), slog.Int("line", line))

	if _, ok := readWriteTxFromContext(ctx); ok {
		s.logger.DebugContext(ctx, "joining existing read-write transaction", caller)
		return fn(ctx)
	}

	s.logger.DebugContext(ctx, "transaction starting", slog.String("type", "read-write"), caller)
	start := time.Now()

	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		txCtx, err := withReadWriteTx(ctx, tx)
		if err != nil {
			return err
		}
		return fn(txCtx)
	})

	duration := time.Since(start)
	if err != nil {
		s.logger.ErrorContext(ctx, "transaction rolled back", slog.String("type", "read-write"), slog.Duration("duration", duration), slog.Any("error", err), caller)
	} else {
		s.logger.DebugContext(ctx, "transaction committed", slog.String("type", "read-write"), slog.Duration("duration", duration), caller)
	}
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
	_, file, line, _ := runtime.Caller(1)
	caller := slog.Group("caller", slog.String("file", file), slog.Int("line", line))

	if _, ok := readTransactionFromContext(ctx); ok {
		s.logger.DebugContext(ctx, "joining existing read transaction", caller)
		return fn(ctx)
	}

	s.logger.DebugContext(ctx, "transaction starting", slog.String("type", "read-only"), caller)
	start := time.Now()

	tx := s.client.ReadOnlyTransaction()
	defer tx.Close()

	txCtx, err := withReadOnlyTx(ctx, tx)
	if err != nil {
		s.logger.ErrorContext(ctx, "transaction setup failed", slog.String("type", "read-only"), slog.Any("error", err), caller)
		return err
	}

	err = fn(txCtx)
	duration := time.Since(start)

	if err != nil {
		s.logger.ErrorContext(ctx, "transaction failed", slog.String("type", "read-only"), slog.Duration("duration", duration), slog.Any("error", err), caller)
	} else {
		s.logger.DebugContext(ctx, "transaction closed", slog.String("type", "read-only"), slog.Duration("duration", duration), caller)
	}
	return err
}
