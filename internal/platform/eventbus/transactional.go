package eventbus

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rai/clean-modularmonolith-go/modules/shared/events"
)

// ErrEventProcessingDepthExceeded is returned when event handlers
// trigger too many nested events.
var ErrEventProcessingDepthExceeded = errors.New("event processing depth exceeded")

// TransactionalEventBus buffers events and processes them synchronously
// within a transaction scope. Create a new instance per transaction.
//
// IMPORTANT: For Spanner retry safety, create this inside the transaction
// closure to ensure clean state on each retry attempt.
//
// Example:
//
//	txScope.Execute(ctx, func(ctx context.Context) error {
//	    eventBus := eventbus.NewTransactional(registry, 10)
//	    // ... business logic ...
//	    eventBus.Publish(ctx, event)
//	    return eventBus.Flush(ctx)
//	})
type TransactionalEventBus struct {
	registry HandlerRegistry
	pending  []events.Event
	mu       sync.Mutex
	maxDepth int
}

// NewTransactional creates a TransactionalEventBus with the given registry.
// maxDepth limits nested event processing to prevent infinite loops (default: 10).
func NewTransactional(registry HandlerRegistry, maxDepth int) *TransactionalEventBus {
	if maxDepth <= 0 {
		maxDepth = 10
	}
	return &TransactionalEventBus{
		registry: registry,
		maxDepth: maxDepth,
	}
}

// Publish buffers an event for later processing.
// Events are not processed until Flush is called.
// Implements events.Publisher.
func (b *TransactionalEventBus) Publish(ctx context.Context, event events.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = append(b.pending, event)
	return nil
}

// Flush processes all buffered events synchronously.
// Handlers may publish additional events, which are processed in the same flush.
// Returns error if any handler fails (caller should rollback transaction).
func (b *TransactionalEventBus) Flush(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	depth := 0
	for len(b.pending) > 0 {
		if depth >= b.maxDepth {
			return ErrEventProcessingDepthExceeded
		}

		// Take the first event
		event := b.pending[0]
		b.pending = b.pending[1:]

		handlers := b.registry.HandlersFor(event.EventType())
		for _, handler := range handlers {
			// Unlock during handler execution to allow Publish calls from handlers
			b.mu.Unlock()
			err := handler.Handle(ctx, event)
			b.mu.Lock()

			if err != nil {
				return fmt.Errorf("handler failed for event %s: %w", event.EventType().String(), err)
			}
		}

		depth++
	}

	return nil
}

// PendingCount returns the number of buffered events (useful for testing).
func (b *TransactionalEventBus) PendingCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.pending)
}

// Compile-time interface check.
var _ events.Publisher = (*TransactionalEventBus)(nil)
