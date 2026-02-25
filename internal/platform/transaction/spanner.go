package transaction

import (
	"context"

	"cloud.google.com/go/spanner"
)

// SpannerTransactionScope implements TransactionScope for Cloud Spanner.
type SpannerTransactionScope struct {
	client *spanner.Client
}

// NewSpannerTransactionScope creates a new Spanner-backed transaction scope.
func NewSpannerTransactionScope(client *spanner.Client) *SpannerTransactionScope {
	return &SpannerTransactionScope{client: client}
}

// Execute runs fn within a Spanner ReadWriteTransaction.
// The transaction is embedded in ctx for repositories to access via TxFromContext.
//
// IMPORTANT: Spanner may retry fn on Aborted errors. Therefore:
//   - fn must be idempotent
//   - fn must NOT perform external side effects (email, API calls, etc.)
//   - Any state (like TransactionalEventBus) should be created inside fn
func (s *SpannerTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Embed transaction in context for repositories
		ctx = WithTx(ctx, txn)
		return fn(ctx)
	})
	return err
}

// Compile-time interface check.
var _ TransactionScope = (*SpannerTransactionScope)(nil)
