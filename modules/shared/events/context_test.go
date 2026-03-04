package events_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

type testEvent struct {
	events.BaseEvent
}

func newTestEvent() testEvent {
	return testEvent{
		BaseEvent: events.NewBaseEvent("test.TestHappened", "agg-1"),
	}
}

func TestAddAndCollect(t *testing.T) {
	ctx := events.NewContext(context.Background())

	e1 := newTestEvent()
	e2 := newTestEvent()
	events.Add(ctx, e1, e2)

	collected := events.Collect(ctx)
	if len(collected) != 2 {
		t.Fatalf("expected 2 events, got %d", len(collected))
	}
}

func TestCollect_ClearsEvents(t *testing.T) {
	ctx := events.NewContext(context.Background())
	events.Add(ctx, newTestEvent())

	_ = events.Collect(ctx)
	second := events.Collect(ctx)
	if len(second) != 0 {
		t.Fatalf("expected 0 events after second collect, got %d", len(second))
	}
}

func TestCollect_NoCollector_ReturnsNil(t *testing.T) {
	ctx := context.Background()
	collected := events.Collect(ctx)
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
	events.Add(ctx, newTestEvent())
}

func TestNewContext_CreatesFreshCollector(t *testing.T) {
	ctx := events.NewContext(context.Background())
	events.Add(ctx, newTestEvent())

	// Create a fresh context — old events should NOT be visible
	ctx2 := events.NewContext(ctx)
	collected := events.Collect(ctx2)
	if len(collected) != 0 {
		t.Fatalf("expected fresh collector with 0 events, got %d", len(collected))
	}
}

func TestDerivedContext_SharesCollector(t *testing.T) {
	ctx := events.NewContext(context.Background())

	// Simulate errgroup-style derived context
	derived, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	events.Add(derived, newTestEvent())

	// Collect from the original ctx should see the event
	collected := events.Collect(ctx)
	if len(collected) != 1 {
		t.Fatalf("expected 1 event via derived context, got %d", len(collected))
	}
}

func TestConcurrentAdd(t *testing.T) {
	ctx := events.NewContext(context.Background())

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			events.Add(ctx, newTestEvent())
		}()
	}
	wg.Wait()

	collected := events.Collect(ctx)
	if len(collected) != 100 {
		t.Fatalf("expected 100 events, got %d", len(collected))
	}
}
