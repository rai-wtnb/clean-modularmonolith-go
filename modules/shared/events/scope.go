package events

import (
	"context"

	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// scopeWithDomainEventImpl composes a transaction.Scope with event collection
// and publishing. It is the sole orchestrator of the event lifecycle:
// CaptureEvents (collector + drain), publish, and repeat for cascading events.
//
// It supports two phases of event handling:
//   - Pre-commit: handlers run inside the transaction (via publisher). Failures roll back the transaction.
//   - Post-commit: handlers run after the transaction commits (via postCommitPublisher). Failures are logged, not propagated.
type scopeWithDomainEventImpl struct {
	inner               transaction.Scope
	publisher           Publisher
	postCommitPublisher PostCommitPublisher
}

var _ transaction.ScopeWithDomainEvent = (*scopeWithDomainEventImpl)(nil)

// NewScopeWithDomainEvent creates a new ScopeWithDomainEvent that wraps
// the given transaction.Scope with automatic event collection and publishing.
// postCommitPublisher may be nil if no post-commit handlers are needed.
func NewScopeWithDomainEvent(inner transaction.Scope, publisher Publisher, postCommitPublisher PostCommitPublisher) transaction.ScopeWithDomainEvent {
	return &scopeWithDomainEventImpl{inner: inner, publisher: publisher, postCommitPublisher: postCommitPublisher}
}

func (s *scopeWithDomainEventImpl) ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error {
	var publishedEvents []Event

	innerFn := func(ctx context.Context) error {
		// Reset on each attempt (Execute may retry).
		publishedEvents = nil
		ctx = newContext(ctx)
		if err := fn(ctx); err != nil {
			return err
		}

		for evts := collect(ctx); len(evts) > 0; evts = collect(ctx) {
			if err := s.publisher.Publish(ctx, evts); err != nil {
				return err
			}
			publishedEvents = append(publishedEvents, evts...)
		}

		return nil
	}
	err := s.inner.Execute(ctx, innerFn)
	if err != nil {
		return err
	}

	// Transaction committed successfully — fire post-commit handlers.
	if len(publishedEvents) > 0 && s.postCommitPublisher != nil {
		s.postCommitPublisher.PublishPostCommit(ctx, publishedEvents)
	}
	return nil
}
