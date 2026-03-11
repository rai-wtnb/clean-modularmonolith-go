package transaction

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// ScopeWithDomainEvent manages domain event lifecycle within a transaction.
// It initializes event collectors, executes business logic within a transaction,
// and publishes collected domain events after successful execution.
//
// Command handlers that produce domain events should depend on this interface
// rather than the plain Scope interface.
type ScopeWithDomainEvent interface {
	// ExecuteWithPublish runs the given function within a transaction,
	// collecting domain events and publishing them after successful execution.
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

// scopeWithDomainEventImpl wraps a transaction Scope with automatic domain event
// collection and publishing. It initializes a fresh event collector in the
// context before running the business logic, and publishes all collected
// events after the function returns successfully (but before commit).
//
// This removes the need for command handlers to manage event publishing
// explicitly, preventing missed event publishing structurally.
type scopeWithDomainEventImpl struct {
	inner     Scope
	publisher events.Publisher
}

var _ ScopeWithDomainEvent = (*scopeWithDomainEventImpl)(nil)

func NewScopeWithDomainEvent(inner Scope, publisher events.Publisher) ScopeWithDomainEvent {
	return &scopeWithDomainEventImpl{inner: inner, publisher: publisher}
}

func (s *scopeWithDomainEventImpl) ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.inner.Execute(ctx, func(ctx context.Context) error {
		// Fresh collector per invocation — important for Spanner retries
		// where the same function may be called multiple times.
		ctx = events.NewContext(ctx)
		if err := fn(ctx); err != nil {
			return err
		}
		return s.publisher.Publish(ctx)
	})
}
