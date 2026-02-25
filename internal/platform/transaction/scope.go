// Package transaction provides transaction management abstractions.
package transaction

import "context"

// TransactionScope manages the lifecycle of a transaction.
// It provides a clean abstraction for executing business logic
// within a transactional boundary.
type TransactionScope interface {
	// Execute runs the given function within a transaction.
	// The transaction is committed if fn returns nil, rolled back otherwise.
	// The ctx passed to fn contains the transaction for repositories to use.
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}
