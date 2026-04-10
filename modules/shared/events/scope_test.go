package events

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// retryScope simulates Spanner Aborted retries by calling fn multiple times.
type retryScope struct {
	attempts int
}

func (s retryScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	for i := 0; i < s.attempts-1; i++ {
		_ = fn(ctx) // simulate aborted retry, discard result
	}
	return fn(ctx)
}

// fakeScope is a minimal transaction.Scope that simply calls fn.
type fakeScope struct{}

func (fakeScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakePublisher records published events and optionally adds new events
// to the collector (simulating a handler that emits cascading events).
type fakePublisher struct {
	published []Event
	// onPublish is called for each Publish invocation, allowing tests to
	// inject cascading events via events.Add.
	onPublish func(ctx context.Context, evts []Event)
}

func (p *fakePublisher) Publish(ctx context.Context, evts []Event) error {
	p.published = append(p.published, evts...)
	if p.onPublish != nil {
		p.onPublish(ctx, evts)
	}
	return nil
}

// fakePostCommitPublisher records whether PublishPostCommit was called.
type fakePostCommitPublisher struct {
	events []Event
}

func (p *fakePostCommitPublisher) PublishPostCommit(_ context.Context, evts []Event) {
	p.events = append(p.events, evts...)
}

func TestExecuteWithPublish_NormalFlow(t *testing.T) {
	pub := &fakePublisher{}
	postPub := &fakePostCommitPublisher{}
	scope := NewScopeWithDomainEvent(fakeScope{}, pub, postPub)

	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, newTestEvent())
		Add(ctx, newTestEvent())
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.published) != 2 {
		t.Fatalf("expected 2 published events, got %d", len(pub.published))
	}
	if len(postPub.events) != 2 {
		t.Fatalf("expected 2 post-commit events, got %d", len(postPub.events))
	}
}

func TestExecuteWithPublish_FnError_NoPublish(t *testing.T) {
	pub := &fakePublisher{}
	scope := NewScopeWithDomainEvent(fakeScope{}, pub, nil)

	wantErr := errors.New("business error")
	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, newTestEvent())
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected 0 published events on error, got %d", len(pub.published))
	}
}

func TestExecuteWithPublish_PostCommitNil(t *testing.T) {
	pub := &fakePublisher{}
	scope := NewScopeWithDomainEvent(fakeScope{}, pub, nil)

	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, newTestEvent())
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecuteWithPublish_NoEvents_NoPostCommit(t *testing.T) {
	pub := &fakePublisher{}
	postPub := &fakePostCommitPublisher{}
	scope := NewScopeWithDomainEvent(fakeScope{}, pub, postPub)

	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(postPub.events) != 0 {
		t.Fatalf("expected no post-commit events, got %d", len(postPub.events))
	}
}

func TestExecuteWithPublish_NestedScope_PostCommitOnlyOnOutermost(t *testing.T) {
	eventA := EventType("test.EventA")
	eventB := EventType("test.EventB")

	outerPub := &fakePublisher{}
	innerPub := &fakePublisher{}
	postPub := &fakePostCommitPublisher{}

	innerScope := NewScopeWithDomainEvent(fakeScope{}, innerPub, postPub)

	// When the outer scope publishes, simulate a pre-commit handler
	// that calls its own ExecuteWithPublish (nested scope).
	outerPub.onPublish = func(ctx context.Context, evts []Event) {
		_ = innerScope.ExecuteWithPublish(ctx, func(ctx context.Context) error {
			Add(ctx, testEvent{BaseEvent: NewBaseEvent(eventB)})
			return nil
		})
	}

	outerScope := NewScopeWithDomainEvent(fakeScope{}, outerPub, postPub)

	err := outerScope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, testEvent{BaseEvent: NewBaseEvent(eventA)})
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Outer published EventA, inner published EventB.
	if len(outerPub.published) != 1 {
		t.Fatalf("outer: expected 1 published event, got %d", len(outerPub.published))
	}
	if len(innerPub.published) != 1 {
		t.Fatalf("inner: expected 1 published event, got %d", len(innerPub.published))
	}

	// PostCommitPublish should be called exactly once (by outermost scope)
	// with both events in chronological order: EventA (outer) then EventB (inner).
	if len(postPub.events) != 2 {
		t.Fatalf("expected 2 post-commit events, got %d", len(postPub.events))
	}
	if postPub.events[0].EventType() != eventA {
		t.Fatalf("expected first post-commit event to be %s, got %s", eventA, postPub.events[0].EventType())
	}
	if postPub.events[1].EventType() != eventB {
		t.Fatalf("expected second post-commit event to be %s, got %s", eventB, postPub.events[1].EventType())
	}
}

func TestExecuteWithPublish_OrphanedEvents_Error(t *testing.T) {
	pub := &fakePublisher{}
	// Handler calls events.Add directly without ExecuteWithPublish.
	pub.onPublish = func(ctx context.Context, evts []Event) {
		Add(ctx, newTestEvent())
	}

	scope := NewScopeWithDomainEvent(fakeScope{}, pub, nil)

	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, newTestEvent())
		return nil
	})

	if err == nil {
		t.Fatal("expected error for orphaned events, got nil")
	}
	if !strings.Contains(err.Error(), "orphaned events") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestExecuteWithPublish_SpannerRetry_FreshAccumulator(t *testing.T) {
	attempt := 0
	pub := &fakePublisher{}
	postPub := &fakePostCommitPublisher{}
	scope := NewScopeWithDomainEvent(retryScope{attempts: 2}, pub, postPub)

	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		attempt++
		Add(ctx, newTestEvent())
		if attempt == 2 {
			Add(ctx, newTestEvent()) // second attempt adds an extra event
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Post-commit should only contain events from the final (second) attempt.
	if len(postPub.events) != 2 {
		t.Fatalf("expected 2 post-commit events (from final attempt), got %d", len(postPub.events))
	}
}

func TestExecuteWithPublish_NestedScope_ErrorPropagation(t *testing.T) {
	innerPub := &fakePublisher{}
	postPub := &fakePostCommitPublisher{}

	wantErr := errors.New("inner error")
	innerScope := NewScopeWithDomainEvent(fakeScope{}, innerPub, postPub)

	outerScope := NewScopeWithDomainEvent(fakeScope{}, &errorOnNestedPublisher{
		innerScope: innerScope,
		innerErr:   wantErr,
	}, postPub)

	err := outerScope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, newTestEvent())
		return nil
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
	if len(postPub.events) != 0 {
		t.Fatalf("expected no post-commit events on error, got %d", len(postPub.events))
	}
}

// errorOnNestedPublisher is a Publisher that invokes a nested ExecuteWithPublish
// which returns innerErr.
type errorOnNestedPublisher struct {
	published  []Event
	innerScope interface {
		ExecuteWithPublish(ctx context.Context, fn func(ctx context.Context) error) error
	}
	innerErr error
}

func (p *errorOnNestedPublisher) Publish(ctx context.Context, evts []Event) error {
	p.published = append(p.published, evts...)
	return p.innerScope.ExecuteWithPublish(ctx, func(ctx context.Context) error {
		return p.innerErr
	})
}
