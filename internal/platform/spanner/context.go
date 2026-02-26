package spanner

import (
	"context"

	"cloud.google.com/go/spanner"
)

// txKey is the context key for storing Spanner transactions.
type txKey struct{}

// withTx embeds a Spanner ReadWriteTransaction in the context.
func withTx(ctx context.Context, tx *spanner.ReadWriteTransaction) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext extracts a Spanner ReadWriteTransaction from context.
// Returns (nil, false) if no transaction is present.
func TxFromContext(ctx context.Context) (*spanner.ReadWriteTransaction, bool) {
	tx, ok := ctx.Value(txKey{}).(*spanner.ReadWriteTransaction)
	return tx, ok
}
