package events

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// scopeWithDomainEventImpl composes a transaction.Scope with event collection
// and publishing. It is the sole orchestrator of the event lifecycle:
// CaptureEvents (collector + drain), publish, and repeat for cascading events.
type scopeWithDomainEventImpl struct {
	inner     transaction.Scope
	publisher Publisher
}

var _ transaction.ScopeWithDomainEvent = (*scopeWithDomainEventImpl)(nil)

// NewScopeWithDomainEvent creates a new ScopeWithDomainEvent that wraps
// the given transaction.Scope with automatic event collection and publishing.
func NewScopeWithDomainEvent(inner transaction.Scope, publisher Publisher) transaction.ScopeWithDomainEvent {
	return &scopeWithDomainEventImpl{inner: inner, publisher: publisher}
}

func (s *scopeWithDomainEventImpl) ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.inner.Execute(ctx, func(ctx context.Context) error {
		ctx = newContext(ctx)
		if err := fn(ctx); err != nil {
			return err
		}
		for evts := collect(ctx); len(evts) > 0; evts = collect(ctx) {
			if err := s.publisher.Publish(ctx, evts); err != nil {
				return err
			}
		}
		return nil
	})
}
