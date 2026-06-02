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

// Read executes fn with a read transaction from the context, or falls back to
// a standalone ReadOnlyTransaction for a point-in-time consistent snapshot.
//
// Like Write, Read defaults to joining the transaction already in the context
// (RW preferred for read-your-writes, then RO) — participation is the normal
// case. The name describes only the standalone strategy: when no transaction is
// active, Read uses a ReadOnlyTransaction so multiple reads see one consistent
// snapshot (e.g., COUNT + SELECT, or reading from multiple tables). This is the
// safe default; use ReadOrSingle to opt into the cheaper one-shot fallback.
func Read[T any](ctx context.Context, client *spanner.Client, logger *slog.Logger, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := readTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}

	finishLog := txLog(ctx, logger, TxReadOnly, "Read")

	roTx := client.ReadOnlyTransaction()
	defer roTx.Close()

	result, err := fn(ctx, roTx)
	finishLog(err)
	return result, err
}

// ReadOrSingle behaves like Read but, when standalone, falls back to
// client.Single() — a one-shot read — instead of a multi-read snapshot.
//
// Like Read, it defaults to joining the transaction already in the context
// (RW preferred, then RO). The name describes only the standalone strategy:
// use ReadOrSingle when the operation performs a single read call (ReadRow,
// Query) and does not need a consistent snapshot across multiple reads; it is
// the cheapest option. Prefer Read when in doubt.
func ReadOrSingle[T any](ctx context.Context, client *spanner.Client, logger *slog.Logger, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := readTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}

	finishLog := txLog(ctx, logger, TxSingleRead, "ReadOrSingle")

	result, err := fn(ctx, client.Single())
	finishLog(err)
	return result, err
}
