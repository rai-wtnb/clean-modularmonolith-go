package transaction

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// EventAwareScope wraps a transaction Scope with automatic domain event
// collection and publishing. It initializes a fresh event collector in the
// context before running the business logic, and publishes all collected
// events after the function returns successfully (but before commit).
//
// This removes the need for command handlers to manage event publishing
// explicitly, preventing missed event publishing structurally.
type EventAwareScope struct {
	inner     Scope
	publisher events.Publisher
}

var _ Scope = (*EventAwareScope)(nil)

func NewEventAwareScope(inner Scope, publisher events.Publisher) *EventAwareScope {
	return &EventAwareScope{inner: inner, publisher: publisher}
}

func (s *EventAwareScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
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
