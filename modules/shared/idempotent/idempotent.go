package idempotent

import (
	"sync"
	"time"
)

// defaultOnceTTL is how long a deduplication key is retained.
// Sized to outlast Spanner Aborted retry windows (typically seconds) while
// bounding memory growth in long-running processes.
const defaultOnceTTL = 5 * time.Minute

// Base is an embeddable struct for external-call clients (email,
// HTTP, external APIs). Embed it in the client/sender, not in the
// DomainEventHandler itself, so that transactional DB operations within the
// same handler are unaffected and still re-run on Spanner retries.
//
// Wrap each outbound call with Once(key, fn) where key captures the call
// parameters. Same params → same key → skip. Different params (business
// logic changed between retries) → different key → execute.
//
// Example:
//
//	type EmailSender struct {
//	    idempotent.Base
//	    // ...
//	}
//
//	func (s *EmailSender) SendConfirmation(orderID string) error {
//	    return s.Once("send-confirmation:"+orderID, func() error {
//	        return s.client.Send(orderID)
//	    })
//	}
//
// Deduplication is in-memory, scoped to a single process instance and cleared
// on restart. Entries expire after defaultOnceTTL to bound memory growth.
// For stronger guarantees across restarts or instances, use a persistent
// backend instead.
type Base struct {
	seen sync.Map
}

// Once executes fn only if key has not been seen within the TTL window.
// Subsequent calls with the same key within the window return nil without
// invoking fn.
func (b *Base) Once(key string, fn func() error) error {
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
