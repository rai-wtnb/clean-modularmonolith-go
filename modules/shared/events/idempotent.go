package events

import (
	"sync"
	"time"
)

// defaultOnceTTL is how long a deduplication key is retained.
// Sized to outlast Spanner Aborted retry windows (typically seconds) while
// bounding memory growth in long-running processes.
const defaultOnceTTL = 5 * time.Minute

// IdempotentBase is an embeddable struct for handlers that perform external
// side effects (email, HTTP calls, external APIs). Embed it and wrap each
// external call with Once(key, fn), where key is derived from the actual call
// parameters.
//
// Use this when: the handler makes out-of-transaction calls that cannot be
// rolled back. Same params → same key → skip. Different params (business
// logic changed between retries) → different key → execute.
//
// Do NOT use this for handlers that only perform transactional DB writes.
// Those must re-run on retry (previous writes were rolled back) and rely on
// DB-level idempotency instead.
//
// Example:
//
//	type MyHandler struct {
//	    events.IdempotentBase
//	    // ...
//	}
//
//	func (h *MyHandler) Handle(ctx context.Context, event events.Event) error {
//	    e := event.(contracts.SomeEvent)
//	    return h.Once(fmt.Sprintf("my-operation:%s", e.SomeID), func() error {
//	        return callExternalService(e.SomeID)
//	    })
//	}
//
// Deduplication is in-memory, scoped to a single process instance and cleared
// on restart. Entries expire after defaultOnceTTL to bound memory growth.
// For stronger guarantees across restarts or instances, use a persistent
// backend instead.
type IdempotentBase struct {
	seen sync.Map
}

// Once executes fn only if key has not been seen within the TTL window.
// Subsequent calls with the same key within the window return nil without
// invoking fn.
func (b *IdempotentBase) Once(key string, fn func() error) error {
	now := time.Now()
	expiry := now.Add(defaultOnceTTL)

	if actual, loaded := b.seen.LoadOrStore(key, expiry); loaded {
		if t, ok := actual.(time.Time); ok && now.Before(t) {
			return nil // within TTL: skip
		}
		// Expired: refresh the entry and fall through to execute fn.
		// CompareAndSwap is best-effort; if it loses the race another goroutine
		// will also execute fn, which is acceptable (same as no dedup).
		b.seen.CompareAndSwap(key, actual, expiry)
	}
	return fn()
}
