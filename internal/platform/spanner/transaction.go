package spanner

import (
	"context"

	"cloud.google.com/go/spanner"

	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
)

// TransactionScope implements transaction.TransactionScope for Cloud Spanner.
type TransactionScope struct {
	client *spanner.Client
}

// NewTransactionScope creates a new Spanner-backed transaction scope.
func NewTransactionScope(client *spanner.Client) *TransactionScope {
	return &TransactionScope{client: client}
}

// Execute runs fn within a Spanner ReadWriteTransaction.
// The transaction is embedded in ctx for repositories to access via TxFromContext.
//
// IMPORTANT: Spanner may retry fn on Aborted errors. Therefore:
//   - fn must be idempotent
//   - fn must NOT perform external side effects (email, API calls, etc.)
//   - Any state (like TransactionalPublisher) should be created inside fn
func (s *TransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	_, err := s.client.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		ctx = withTx(ctx, tx)
		return fn(ctx)
	})
	return err
}

var _ transaction.TransactionScope = (*TransactionScope)(nil)
