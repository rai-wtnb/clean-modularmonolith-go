package events

import (
	"context"
	"errors"
	"strings"
	"testing"
)

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

func TestExecuteWithPublish_CascadingEvents(t *testing.T) {
	eventA := EventType("test.EventA")
	eventB := EventType("test.EventB")

	cascaded := false
	pub := &fakePublisher{}
	pub.onPublish = func(ctx context.Context, evts []Event) {
		// First iteration: handler for EventA emits EventB
		for _, e := range evts {
			if e.EventType() == eventA && !cascaded {
				cascaded = true
				Add(ctx, testEvent{BaseEvent: NewBaseEvent(eventB)})
			}
		}
	}

	scope := NewScopeWithDomainEvent(fakeScope{}, pub, nil)

	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, testEvent{BaseEvent: NewBaseEvent(eventA)})
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// EventA published in iteration 1, EventB published in iteration 2
	if len(pub.published) != 2 {
		t.Fatalf("expected 2 published events (cascading), got %d", len(pub.published))
	}
}

func TestExecuteWithPublish_CycleDetected(t *testing.T) {
	eventA := EventType("test.EventA")
	eventB := EventType("test.EventB")

	pub := &fakePublisher{}
	pub.onPublish = func(ctx context.Context, evts []Event) {
		for _, e := range evts {
			switch e.EventType() {
			case eventA:
				Add(ctx, testEvent{BaseEvent: NewBaseEvent(eventB)})
			case eventB:
				Add(ctx, testEvent{BaseEvent: NewBaseEvent(eventA)})
			}
		}
	}

	scope := NewScopeWithDomainEvent(fakeScope{}, pub, nil)

	err := scope.ExecuteWithPublish(context.Background(), func(ctx context.Context) error {
		Add(ctx, testEvent{BaseEvent: NewBaseEvent(eventA)})
		return nil
	})

	if err == nil {
		t.Fatal("expected error for event cycle, got nil")
	}
	if !strings.Contains(err.Error(), "event drain loop exceeded") {
		t.Fatalf("unexpected error message: %v", err)
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
