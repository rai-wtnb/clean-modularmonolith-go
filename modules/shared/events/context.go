package events

import (
	"context"
	"sync"
)

type collectorKey struct{}

// eventCollector accumulates domain events within a request/transaction scope.
// It is stored as a pointer in context, so derived contexts share the same collector.
type eventCollector struct {
	mu     sync.Mutex
	events []Event
}

// newContext returns a new context with a fresh event collector.
// Call this at the start of each transaction to ensure a clean slate
// (important for Spanner retries).
func newContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, collectorKey{}, &eventCollector{})
}

// Add appends domain events to the collector in ctx.
// Panics if no collector is present — this is a programming error
// indicating that ScopeWithDomainEvent was not used.
func Add(ctx context.Context, evts ...Event) {
	c, ok := ctx.Value(collectorKey{}).(*eventCollector)
	if !ok {
		panic("events.Add: no event collector in context; ensure ScopeWithDomainEvent is used")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evts...)
}

// collect atomically drains and returns all events from the collector.
// Returns nil if no collector is present in the context.
func collect(ctx context.Context) []Event {
	c, ok := ctx.Value(collectorKey{}).(*eventCollector)
	if !ok {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	evts := c.events
	c.events = nil
	return evts
}

// CaptureEvents creates a fresh event collector, runs fn, and returns
// all domain events added during execution. Used by tests to verify
// that domain logic emits the expected events (see eventstest package).
func CaptureEvents(ctx context.Context, fn func(ctx context.Context) error) ([]Event, error) {
	ctx = newContext(ctx)
	if err := fn(ctx); err != nil {
		return nil, err
	}
	return collect(ctx), nil
}

// accumulatorKey is the context key for the shared post-commit accumulator.
type accumulatorKey struct{}

// postCommitAccumulator collects published events across all nested
// ExecuteWithPublish scopes within a single transaction. Only the outermost
// scope drains the accumulator for PostCommitPublish.
type postCommitAccumulator struct {
	mu     sync.Mutex
	events []Event
}

func (a *postCommitAccumulator) reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = nil
}

func (a *postCommitAccumulator) add(evts []Event) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.events = append(a.events, evts...)
}

func (a *postCommitAccumulator) drain() []Event {
	a.mu.Lock()
	defer a.mu.Unlock()
	evts := a.events
	a.events = nil
	return evts
}

func accumulatorFromContext(ctx context.Context) (*postCommitAccumulator, bool) {
	a, ok := ctx.Value(accumulatorKey{}).(*postCommitAccumulator)
	return a, ok
}

func newContextWithAccumulator(ctx context.Context, a *postCommitAccumulator) context.Context {
	return context.WithValue(ctx, accumulatorKey{}, a)
}
