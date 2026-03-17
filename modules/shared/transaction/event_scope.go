//go:generate mockgen -source=event_scope.go -destination=mocks/mock_event_scope.go -package=mocks

package transaction

import "context"

// ScopeWithDomainEvent manages domain event lifecycle within a transaction.
// It initializes event collectors, executes business logic within a transaction,
// and publishes collected domain events after successful execution.
//
// Command handlers that produce domain events should depend on this interface
// rather than the plain Scope interface.
//
// The implementation is provided by the events package (events.NewScopeWithDomainEvent),
// which encapsulates the event collection mechanism.
type ScopeWithDomainEvent interface {
	// ExecuteWithPublish runs fn within a transaction, collects domain events,
	// and publishes them before the transaction commits.
	//
	// Publishing is intentionally inside the transaction boundary: if publishing
	// fails (e.g. a handler returns an error), the transaction is rolled back,
	// preserving strong consistency between the write and its side effects.
	// This relies on the in-process event bus; once handlers are migrated to
	// Pub/Sub the atomicity guarantee must be provided by an outbox pattern instead.
	//
	// NOTE: The underlying transaction may be retried (e.g. Spanner Aborted),
	// so fn — and therefore all subscribed handlers — can be invoked more than
	// once per logical request. Handlers must account for this; see
	// idempotent.Base for guidance.
	ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error
}

// ExecuteWithPublishResult runs fn within a ScopeWithDomainEvent and returns the result.
// This is a generic helper that wraps ExecuteWithPublish for cases
// where the transaction needs to return a value.
func ExecuteWithPublishResult[T any](ctx context.Context, scope ScopeWithDomainEvent, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	err := scope.ExecuteWithPublish(ctx, func(ctx context.Context) error {
		var fnErr error
		result, fnErr = fn(ctx)
		return fnErr
	})
	return result, err
}
