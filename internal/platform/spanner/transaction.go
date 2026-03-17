package spanner

import (
	"context"

	"cloud.google.com/go/spanner"
)

// Write executes one or more DML statements within an existing read-write
// transaction from the context, or creates a new transaction if none is active.
// DML provides read-your-writes (RYW) consistency within a transaction.
//
// For a single statement, tx.Update is used directly.
// For multiple statements, tx.BatchUpdate executes them in a single RPC.
func Write(ctx context.Context, client *spanner.Client, stmts ...spanner.Statement) error {
	if len(stmts) == 0 {
		panic("spanner.Write: called with zero statements")
	}

	exec := func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		if len(stmts) == 1 {
			_, err := tx.Update(ctx, stmts[0])
			return err
		}
		_, err := tx.BatchUpdate(ctx, stmts)
		return err
	}

	if txn, ok := readWriteTxFromContext(ctx); ok {
		return exec(ctx, txn)
	}
	// Guard: if a read-only transaction is active, writing is a programming error.
	// Without this check, a new independent RW transaction would be silently created,
	// breaking atomicity with the surrounding read-only scope.
	if _, ok := readOnlyTxFromContext(ctx); ok {
		panic("spanner.Write: cannot write within a read-only transaction scope")
	}
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return exec(ctx, txn)
	})
	return err
}

// SingleRead executes fn with a read transaction from the context, or falls back to
// client.Single() for a one-shot read.
// Use this for operations that perform a single read call.
func SingleRead[T any](ctx context.Context, client *spanner.Client, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := readTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}
	return fn(ctx, client.Single())
}

// ConsistentRead executes fn with a consistent read transaction from the context,
// or creates a new ReadOnlyTransaction for point-in-time consistent reads.
// Use this when performing multiple reads that must see a consistent snapshot
// (e.g., COUNT + SELECT, or reading from multiple tables).
func ConsistentRead[T any](ctx context.Context, client *spanner.Client, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := readTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}
	roTx := client.ReadOnlyTransaction()
	defer roTx.Close()
	return fn(ctx, roTx)
}
