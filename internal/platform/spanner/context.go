package spanner

import (
	"context"
	"errors"

	"cloud.google.com/go/spanner"
)

// ErrNestedTransaction is returned when attempting to start a transaction
// inside an already-active transaction scope.
// Cloud Spanner does not support nested transactions — nesting would silently
// create an independent transaction, breaking atomicity guarantees.
var ErrNestedTransaction = errors.New("nested transaction detected: Cloud Spanner does not support nested transactions")

// readWriteTxKey is the context key for storing Spanner ReadWriteTransaction.
type readWriteTxKey struct{}

// withReadWriteTx embeds a Spanner ReadWriteTransaction in the context.
// Returns ErrNestedTransaction if a transaction already exists in the context.
func withReadWriteTx(ctx context.Context, tx *spanner.ReadWriteTransaction) (context.Context, error) {
	if _, ok := readTransactionFromContext(ctx); ok {
		return nil, ErrNestedTransaction
	}
	return context.WithValue(ctx, readWriteTxKey{}, tx), nil
}

// readWriteTxFromContext extracts a Spanner ReadWriteTransaction from context.
// Returns (nil, false) if no transaction is present.
func readWriteTxFromContext(ctx context.Context) (*spanner.ReadWriteTransaction, bool) {
	tx, ok := ctx.Value(readWriteTxKey{}).(*spanner.ReadWriteTransaction)
	return tx, ok
}

// readOnlyTxKey is the context key for storing Spanner ReadOnlyTransaction.
type readOnlyTxKey struct{}

// withReadOnlyTx embeds a Spanner ReadOnlyTransaction in the context.
// Returns ErrNestedTransaction if a transaction already exists in the context.
func withReadOnlyTx(ctx context.Context, tx *spanner.ReadOnlyTransaction) (context.Context, error) {
	if _, ok := readTransactionFromContext(ctx); ok {
		return nil, ErrNestedTransaction
	}
	return context.WithValue(ctx, readOnlyTxKey{}, tx), nil
}

// readOnlyTxFromContext extracts a Spanner ReadOnlyTransaction from context.
// Returns (nil, false) if no transaction is present.
func readOnlyTxFromContext(ctx context.Context) (*spanner.ReadOnlyTransaction, bool) {
	tx, ok := ctx.Value(readOnlyTxKey{}).(*spanner.ReadOnlyTransaction)
	return tx, ok
}

// ReadTransaction is the common interface for Spanner read operations.
// Both *spanner.ReadWriteTransaction and *spanner.ReadOnlyTransaction implement this.
type ReadTransaction interface {
	Read(ctx context.Context, table string, keys spanner.KeySet, columns []string) *spanner.RowIterator
	ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
	Query(ctx context.Context, statement spanner.Statement) *spanner.RowIterator
}

var _ ReadTransaction = (*spanner.ReadWriteTransaction)(nil)
var _ ReadTransaction = (*spanner.ReadOnlyTransaction)(nil)

// readTransactionFromContext extracts a ReadTransaction from context.
// It checks ReadWriteTransaction first (for read-your-writes within a write tx),
// then ReadOnlyTransaction.
// Returns (nil, false) if no transaction is present.
func readTransactionFromContext(ctx context.Context) (ReadTransaction, bool) {
	if rwTx, ok := readWriteTxFromContext(ctx); ok {
		return rwTx, true
	}
	if roTx, ok := readOnlyTxFromContext(ctx); ok {
		return roTx, true
	}
	return nil, false
}
