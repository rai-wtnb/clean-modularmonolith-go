package events

import (
	"context"
	"sync"
	"testing"
	"time"
)

type testEvent struct {
	BaseEvent
}

func newTestEvent() testEvent {
	return testEvent{
		BaseEvent: NewBaseEvent("test.TestHappened"),
	}
}

func TestAddAndCollect(t *testing.T) {
	ctx := newContext(context.Background())

	e1 := newTestEvent()
	e2 := newTestEvent()
	Add(ctx, e1, e2)

	collected := collect(ctx)
	if len(collected) != 2 {
		t.Fatalf("expected 2 events, got %d", len(collected))
	}
}

func TestCollect_ClearsEvents(t *testing.T) {
	ctx := newContext(context.Background())
	Add(ctx, newTestEvent())

	_ = collect(ctx)
	second := collect(ctx)
	if len(second) != 0 {
		t.Fatalf("expected 0 events after second collect, got %d", len(second))
	}
}

func TestCollect_NoCollector_ReturnsNil(t *testing.T) {
	ctx := context.Background()
	collected := collect(ctx)
	if collected != nil {
		t.Fatalf("expected nil, got %v", collected)
	}
}

func TestAdd_NoCollector_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when Add is called without collector")
		}
	}()

	ctx := context.Background()
	Add(ctx, newTestEvent())
}

func TestNewContext_CreatesFreshCollector(t *testing.T) {
	ctx := newContext(context.Background())
	Add(ctx, newTestEvent())

	// Create a fresh context — old events should NOT be visible
	ctx2 := newContext(ctx)
	collected := collect(ctx2)
	if len(collected) != 0 {
		t.Fatalf("expected fresh collector with 0 events, got %d", len(collected))
	}
}

func TestDerivedContext_SharesCollector(t *testing.T) {
	ctx := newContext(context.Background())

	// Simulate errgroup-style derived context
	derived, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	Add(derived, newTestEvent())

	// Collect from the original ctx should see the event
	collected := collect(ctx)
	if len(collected) != 1 {
		t.Fatalf("expected 1 event via derived context, got %d", len(collected))
	}
}

func TestConcurrentAdd(t *testing.T) {
	ctx := newContext(context.Background())

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Add(ctx, newTestEvent())
		}()
	}
	wg.Wait()

	collected := collect(ctx)
	if len(collected) != 100 {
		t.Fatalf("expected 100 events, got %d", len(collected))
	}
}

func TestCaptureEvents(t *testing.T) {
	evts, err := CaptureEvents(context.Background(), func(ctx context.Context) error {
		Add(ctx, newTestEvent(), newTestEvent())
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
}
