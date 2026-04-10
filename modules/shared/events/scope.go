package events

import (
	"context"
	"fmt"

	"github.com/rai/clean-modularmonolith-go/modules/shared/transaction"
)

// scopeWithDomainEventImpl composes a transaction.Scope with event collection
// and publishing. It is the sole orchestrator of the event lifecycle:
// collect events from business logic, publish to pre-commit handlers, and
// fire post-commit handlers after the transaction commits.
//
// It supports two phases of event handling:
//   - Pre-commit: handlers run inside the transaction (via publisher). Failures roll back the transaction.
//   - Post-commit: handlers run after the transaction commits (via postCommitPublisher). Failures are logged, not propagated.
//
// Nested scopes (e.g., a pre-commit handler that calls ExecuteWithPublish) join
// the existing transaction and share a post-commit accumulator. Only the outermost
// scope fires PostCommitPublish, ensuring post-commit handlers run after the actual commit.
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

// ExecuteWithPublish runs fn within a transaction, collects domain events via
// events.Add, and publishes them in two phases (pre-commit / post-commit).
//
// Nesting: when called from within a pre-commit handler that already has an
// active ExecuteWithPublish scope, the nested scope joins the existing
// transaction (via inner.Execute's REQUIRED propagation) and shares the
// outermost scope's postCommitAccumulator. The nested scope skips
// PostCommitPublish — only the outermost scope fires it after the actual commit.
//
// Pre-commit handlers that emit domain events MUST use their own
// ExecuteWithPublish scope. Calling events.Add directly inside a handler
// without ExecuteWithPublish is detected as an error (orphaned events).
func (s *scopeWithDomainEventImpl) ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error {
	_, isNested := accumulatorFromContext(ctx)
	acc := &postCommitAccumulator{}

	innerFn := func(ctx context.Context) error {
		if isNested {
			// Nested scope: reuse the parent's accumulator.
			acc, _ = accumulatorFromContext(ctx)
		} else {
			// Outermost scope: reset the accumulator on each attempt
			// so that Spanner retries start with a clean slate.
			acc.reset()
			ctx = newContextWithAccumulator(ctx, acc)
		}
		ctx = newContext(ctx)

		if err := fn(ctx); err != nil {
			return err
		}

		evts := collect(ctx)
		if len(evts) == 0 {
			return nil
		}

		// Accumulate before Publish so that parent events precede child events
		// in chronological order.
		acc.add(evts)

		if err := s.publisher.Publish(ctx, evts); err != nil {
			return err
		}

		// Safety: detect handlers that called events.Add without ExecuteWithPublish.
		if orphaned := collect(ctx); len(orphaned) > 0 {
			return fmt.Errorf("events.Add called during Publish without ExecuteWithPublish: %d orphaned events", len(orphaned))
		}

		return nil
	}

	if err := s.inner.Execute(ctx, innerFn); err != nil {
		return err
	}

	// Only the outermost scope fires post-commit handlers,
	// ensuring they run after the actual transaction commit.
	// IMPORTANT: isNested must be checked BEFORE acc.drain(), because nested
	// scopes share the parent's accumulator — draining here would discard
	// the parent's collected events.
	if isNested || s.postCommitPublisher == nil {
		return nil
	}

	allEvents := acc.drain()
	if len(allEvents) == 0 {
		return nil
	}

	s.postCommitPublisher.PublishPostCommit(ctx, allEvents)
	return nil
}
