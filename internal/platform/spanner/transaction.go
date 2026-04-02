package spanner

import (
	"context"
	"errors"
	"log/slog"

	"cloud.google.com/go/spanner"
)

// ErrNoReadWriteTransaction is returned when Write is called without a
// read-write transaction in the context.
var ErrNoReadWriteTransaction = errors.New("spanner.Write: no read-write transaction in context; use ReadWriteTransactionScope")

// ErrWriteInReadOnlyScope is returned when Write is called within a
// read-only transaction scope.
var ErrWriteInReadOnlyScope = errors.New("spanner.Write: cannot write within a read-only transaction scope")

// Write executes one or more DML statements within an existing read-write
// transaction from the context. Returns an error if no read-write transaction
// is active; all writes must go through a ReadWriteTransactionScope.
//
// For a single statement, tx.Update is used directly.
// For multiple statements, tx.BatchUpdate executes them in a single RPC.
func Write(ctx context.Context, stmts ...spanner.Statement) error {
	if len(stmts) == 0 {
		panic("spanner.Write: called with zero statements")
	}

	txn, ok := readWriteTxFromContext(ctx)
	if !ok {
		if _, ro := readOnlyTxFromContext(ctx); ro {
			return ErrWriteInReadOnlyScope
		}
		return ErrNoReadWriteTransaction
	}

	if len(stmts) == 1 {
		_, err := txn.Update(ctx, stmts[0])
		return err
	}
	_, err := txn.BatchUpdate(ctx, stmts)
	return err
}

// SingleRead executes fn with a read transaction from the context, or falls back to
// client.Single() for a one-shot read.
// Use this for operations that perform a single read call.
func SingleRead[T any](ctx context.Context, client *spanner.Client, logger *slog.Logger, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := readTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}

	finishLog := txLog(ctx, logger, TxSingleRead, "SingleRead")

	result, err := fn(ctx, client.Single())
	finishLog(err)
	return result, err
}

// ConsistentRead executes fn with a consistent read transaction from the context,
// or creates a new ReadOnlyTransaction for point-in-time consistent reads.
// Use this when performing multiple reads that must see a consistent snapshot
// (e.g., COUNT + SELECT, or reading from multiple tables).
func ConsistentRead[T any](ctx context.Context, client *spanner.Client, logger *slog.Logger, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := readTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}

	finishLog := txLog(ctx, logger, TxReadOnly, "ConsistentRead")

	roTx := client.ReadOnlyTransaction()
	defer roTx.Close()

	result, err := fn(ctx, roTx)
	finishLog(err)
	return result, err
}
