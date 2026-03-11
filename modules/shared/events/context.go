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

// NewContext returns a new context with a fresh event collector.
// Call this at the start of each transaction to ensure a clean slate
// (important for Spanner retries).
func NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, collectorKey{}, &eventCollector{})
}

// Add appends domain events to the collector in ctx.
// Panics if no collector is present — this is a programming error
// indicating that NewContext was not called (typically via ScopeWithDomainEvent).
func Add(ctx context.Context, evts ...Event) {
	c, ok := ctx.Value(collectorKey{}).(*eventCollector)
	if !ok {
		panic("events.Add: no event collector in context; ensure events.NewContext was called (typically via transaction.ScopeWithDomainEvent)")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evts...)
}

// Collect atomically drains and returns all events from the collector.
// Returns nil if no collector is present in the context.
func Collect(ctx context.Context) []Event {
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
