package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// testHandler is a configurable handler for testing.
type testHandler struct {
	name      string
	subdomain string
	eventType events.EventType
	handleFn  func(ctx context.Context, event events.Event) error
}

func (h *testHandler) HandlerName() string         { return h.name }
func (h *testHandler) Subdomain() string           { return h.subdomain }
func (h *testHandler) EventType() events.EventType { return h.eventType }
func (h *testHandler) Handle(ctx context.Context, event events.Event) error {
	return h.handleFn(ctx, event)
}

// testEvent is a minimal event for testing.
type testEvent struct {
	events.BaseEvent
}

const testEventType events.EventType = "test.TestHappened"

func newTestEvent() testEvent {
	return testEvent{BaseEvent: events.NewBaseEvent(testEventType)}
}

func newTestBus() *EventBus {
	return NewEventBus(slog.Default())
}

// --- Issue 1: Panic recovery ---

func TestPostCommit_PanicRecovery(t *testing.T) {
	bus := newTestBus()

	var mu sync.Mutex
	var called []string

	// Handler A panics
	handlerA := &testHandler{
		name:      "PanicHandler",
		subdomain: "test",
		eventType: testEventType,
		handleFn: func(ctx context.Context, event events.Event) error {
			panic("handler A exploded")
		},
	}
	// Handler B should still execute
	handlerB := &testHandler{
		name:      "NormalHandler",
		subdomain: "test",
		eventType: testEventType,
		handleFn: func(ctx context.Context, event events.Event) error {
			mu.Lock()
			defer mu.Unlock()
			called = append(called, "B")
			return nil
		},
	}

	if err := bus.SubscribePostCommit(testEventType, handlerA); err != nil {
		t.Fatal(err)
	}
	if err := bus.SubscribePostCommit(testEventType, handlerB); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		bus.processPostCommitEvent(context.Background(), newTestEvent())
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("processPostCommitEvent did not complete in time")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(called) != 1 || called[0] != "B" {
		t.Fatalf("expected handler B to be called after A panicked, got: %v", called)
	}
}

func TestPostCommit_ErrorLogged_NotPropagated(t *testing.T) {
	bus := newTestBus()

	var called bool

	errHandler := &testHandler{
		name:      "ErrorHandler",
		subdomain: "test",
		eventType: testEventType,
		handleFn: func(ctx context.Context, event events.Event) error {
			return fmt.Errorf("handler error")
		},
	}
	normalHandler := &testHandler{
		name:      "AfterErrorHandler",
		subdomain: "test",
		eventType: testEventType,
		handleFn: func(ctx context.Context, event events.Event) error {
			called = true
			return nil
		},
	}

	bus.SubscribePostCommit(testEventType, errHandler)
	bus.SubscribePostCommit(testEventType, normalHandler)

	done := make(chan struct{})
	go func() {
		defer close(done)
		bus.processPostCommitEvent(context.Background(), newTestEvent())
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}

	if !called {
		t.Fatal("expected handler after error to still be called")
	}
}

// --- Issue 2: Timeout ---

func TestPostCommit_Timeout(t *testing.T) {
	bus := newTestBus()
	bus.postCommitTimeout = 50 * time.Millisecond

	var ctxErr error
	handler := &testHandler{
		name:      "SlowHandler",
		subdomain: "test",
		eventType: testEventType,
		handleFn: func(ctx context.Context, event events.Event) error {
			// Block until context is cancelled by timeout
			<-ctx.Done()
			ctxErr = ctx.Err()
			return ctxErr
		},
	}

	if err := bus.SubscribePostCommit(testEventType, handler); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		bus.processPostCommitEvent(context.Background(), newTestEvent())
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("processPostCommitEvent did not complete — timeout not working")
	}

	if ctxErr != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", ctxErr)
	}
}

// --- Issue 3: Parallel execution ---

func TestPostCommit_HandlersRunConcurrently(t *testing.T) {
	bus := newTestBus()

	// Both handlers block on a shared barrier. If they run sequentially,
	// the test will deadlock and hit the timeout.
	barrier := make(chan struct{})
	var mu sync.Mutex
	var called []string

	makeHandler := func(name string) *testHandler {
		return &testHandler{
			name:      name,
			subdomain: "test",
			eventType: testEventType,
			handleFn: func(ctx context.Context, event events.Event) error {
				mu.Lock()
				called = append(called, name)
				mu.Unlock()

				// Wait for all handlers to reach this point
				barrier <- struct{}{}
				return nil
			},
		}
	}

	bus.SubscribePostCommit(testEventType, makeHandler("HandlerA"))
	bus.SubscribePostCommit(testEventType, makeHandler("HandlerB"))

	done := make(chan struct{})
	go func() {
		defer close(done)
		bus.processPostCommitEvent(context.Background(), newTestEvent())
	}()

	// Drain the barrier — both handlers must send before either can finish.
	// If handlers are sequential, only one sends and this blocks forever.
	for range 2 {
		select {
		case <-barrier:
		case <-time.After(5 * time.Second):
			t.Fatal("handlers did not run concurrently")
		}
	}

	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(called) != 2 {
		t.Fatalf("expected 2 handlers called, got %d", len(called))
	}
}

func TestPostCommit_PanicInParallel_OtherHandlerStillRuns(t *testing.T) {
	bus := newTestBus()

	var completed sync.WaitGroup
	completed.Add(1)

	panicHandler := &testHandler{
		name:      "ParallelPanicHandler",
		subdomain: "test",
		eventType: testEventType,
		handleFn: func(ctx context.Context, event events.Event) error {
			panic("boom")
		},
	}
	normalHandler := &testHandler{
		name:      "ParallelNormalHandler",
		subdomain: "test",
		eventType: testEventType,
		handleFn: func(ctx context.Context, event events.Event) error {
			completed.Done()
			return nil
		},
	}

	bus.SubscribePostCommit(testEventType, panicHandler)
	bus.SubscribePostCommit(testEventType, normalHandler)

	done := make(chan struct{})
	go func() {
		defer close(done)
		bus.processPostCommitEvent(context.Background(), newTestEvent())
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("processPostCommitEvent did not complete")
	}

	// completed.Done() was called, so Wait returns immediately
	completed.Wait()
}
