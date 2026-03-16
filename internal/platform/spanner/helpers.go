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
		return nil
	}

	exec := func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		if len(stmts) == 1 {
			_, err := tx.Update(ctx, stmts[0])
			return err
		}
		_, err := tx.BatchUpdate(ctx, stmts)
		return err
	}

	if txn, ok := ReadWriteTxFromContext(ctx); ok {
		return exec(ctx, txn)
	}
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return exec(ctx, txn)
	})
	return err
}

// Read executes fn with a read transaction from the context, or falls back to
// client.Single() for a one-shot read.
// Use this for operations that perform a single read call.
func Read[T any](ctx context.Context, client *spanner.Client, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := ReadTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}
	return fn(ctx, client.Single())
}

// ReadConsistent executes fn with a consistent read transaction from the context,
// or creates a new ReadOnlyTransaction for point-in-time consistent reads.
// Use this when performing multiple reads that must see a consistent snapshot
// (e.g., COUNT + SELECT, or reading from multiple tables).
func ReadConsistent[T any](ctx context.Context, client *spanner.Client, fn func(ctx context.Context, rtx ReadTransaction) (T, error)) (T, error) {
	if rtx, ok := ReadTransactionFromContext(ctx); ok {
		return fn(ctx, rtx)
	}
	roTx := client.ReadOnlyTransaction()
	defer roTx.Close()
	return fn(ctx, roTx)
}
