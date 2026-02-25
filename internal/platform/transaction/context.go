package transaction

import (
	"context"

	"cloud.google.com/go/spanner"
)

// ctxKey is the context key for storing transactions.
type ctxKey struct{}

// WithTx embeds a Spanner ReadWriteTransaction in the context.
func WithTx(ctx context.Context, txn *spanner.ReadWriteTransaction) context.Context {
	return context.WithValue(ctx, ctxKey{}, txn)
}

// TxFromContext extracts a Spanner ReadWriteTransaction from context.
// Returns (nil, false) if no transaction is present.
func TxFromContext(ctx context.Context) (*spanner.ReadWriteTransaction, bool) {
	txn, ok := ctx.Value(ctxKey{}).(*spanner.ReadWriteTransaction)
	return txn, ok
}
