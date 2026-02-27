package transaction

import "context"

// Scope manages the lifecycle of a transaction.
// It provides a clean abstraction for executing business logic
// within a transactional boundary.
//
// Implementations (e.g., Spanner read-write, read-only) handle
// the concrete transaction lifecycle: begin, commit/rollback, and retry.
type Scope interface {
	// Execute runs the given function within a transaction.
	// The transaction is committed if fn returns nil, rolled back otherwise.
	// The ctx passed to fn contains the transaction for repositories to use.
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

// ExecuteWithResult runs fn within a transaction and returns the result.
// This is a generic helper that wraps Scope.Execute for cases
// where the transaction needs to return a value.
func ExecuteWithResult[T any](ctx context.Context, scope Scope, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	err := scope.Execute(ctx, func(ctx context.Context) error {
		var fnErr error
		result, fnErr = fn(ctx)
		return fnErr
	})
	return result, err
}
