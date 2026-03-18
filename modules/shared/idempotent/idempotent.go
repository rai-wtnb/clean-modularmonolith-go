package idempotent

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

// defaultTTL is how long a deduplication entry is retained.
// Sized to outlast Spanner Aborted retry windows (typically seconds) while
// bounding memory growth in long-running processes.
const defaultTTL = 5 * time.Minute

// hash is a content hash produced by HashInput.
// Defined as a named type so that callers cannot pass a plain string literal
// without an explicit conversion, making misuse visible in code review.
type hash string

// entry stores per-key deduplication state.
type entry struct {
	hash   string
	result any // nil for Once, typed value for OnceResult
}

// OutboundCache provides deduplication for outbound calls to external
// services (Elasticsearch, email, HTTP APIs). Embed or store in the
// client/sender, not in the DomainEventHandler itself, so that transactional
// DB operations within the same handler are unaffected and still re-run on
// Spanner retries.
//
// Wrap each outbound call with Once(key, fn) or Once(key, fn, inputHash):
//
//   - Without hash: blocks any re-execution once the key is recorded.
//     Use for irreversible side effects (email, payment, SMS) or operations
//     where input never varies (ES delete by ID).
//
//   - With hash: skips only when the input hash matches the cached value.
//     If input changes, fn re-executes. Use for idempotent operations
//     (ES upsert, file write). Since OutboundCache is composed into the
//     external API client, the caller (DomainEventHandler) can run business
//     logic to compute the request and pass the result's hash — this also
//     covers the case where the output of business logic determines the
//     external API request.
//
// Both modes use store-on-success: the cache entry is written only after
// fn returns nil. Failed operations are never cached.
//
// Deduplication is in-memory, scoped to a single process instance and cleared
// on restart. Entries expire after the configured TTL to bound memory growth.
type OutboundCache struct {
	cache *ttlcache.Cache[string, entry]
}

// NewOutboundCache creates an OutboundCache.
// If ttl is provided, it overrides the default (5 min).
// A background goroutine is started to evict expired entries; call the
// returned stop function to release the goroutine when the cache is no
// longer needed.
func NewOutboundCache(ttl ...time.Duration) (_ *OutboundCache, cleanup func()) {
	d := defaultTTL
	if len(ttl) > 0 {
		d = ttl[0]
	}
	c := ttlcache.New(
		ttlcache.WithTTL[string, entry](d),
	)
	go c.Start()
	return &OutboundCache{cache: c}, c.Stop
}

// HashInput computes a SHA-256 hex digest from the given values.
// Values are JSON-serialized to produce a deterministic byte sequence,
// which naturally handles structs, slices, and nested types.
//
// Callers may pass individual fields or a struct directly:
//
//	b.HashInput(userID, email, age)       // multiple primitives
//	b.HashInput(customerIndex)            // struct
//
// Caveats:
//   - Unexported struct fields are ignored by json.Marshal.
//   - Fields tagged with `json:",omitempty"` are omitted when zero-valued,
//     which may change the hash if the tag is added/removed.
//   - Types implementing json.Marshaler use that custom encoding;
//     ensure it is deterministic and lossless for hashing purposes.
func (b *OutboundCache) HashInput(values ...any) hash {
	data, err := json.Marshal(values)
	if err != nil {
		panic(fmt.Sprintf("idempotent.HashInput: unsupported type: %v", err))
	}
	h := sha256.Sum256(data)
	return hash(fmt.Sprintf("%x", h))
}

// execute is the shared deduplication logic for Once and OnceResult.
func (b *OutboundCache) execute(operation, key string, fn func() (any, error), inputHash ...hash) (any, bool, error) {
	var h string
	if len(inputHash) > 0 {
		h = string(inputHash[0])
	}

	cacheKey := operation + ":" + key
	if item := b.cache.Get(cacheKey); item != nil {
		e := item.Value()
		if h == "" || e.hash == h {
			return e.result, true, nil // cache hit
		}
		// hash mismatch: input changed, fall through to execute
	}

	result, err := fn()
	if err != nil {
		return nil, false, err
	}

	b.cache.Set(cacheKey, entry{
		hash:   h,
		result: result,
	}, ttlcache.DefaultTTL)
	return result, false, nil
}

// Once executes fn at most once per operation+key within the TTL window.
//
//   - Once(op, key, fn): blocks any re-execution for the same operation+key.
//   - Once(op, key, fn, inputHash): skips only when the hash matches; if the
//     input changed, fn re-executes.
//
// fn is NOT cached on failure, so retries after errors will re-execute.
func (b *OutboundCache) Once(operation, key string, fn func() error, inputHash ...hash) error {
	_, _, err := b.execute(operation, key, func() (any, error) {
		return nil, fn()
	}, inputHash...)
	return err
}

// OnceResult executes fn at most once per operation+key within the TTL window
// and returns the cached result on subsequent calls. This is the generic
// counterpart of Once for functions that return a value.
//
// Go does not allow generic methods, so this is a package-level function.
// See Once for deduplication semantics.
func OnceResult[T any](b *OutboundCache, operation, key string, fn func() (T, error), inputHash ...hash) (T, error) {
	result, cached, err := b.execute(operation, key, func() (any, error) {
		return fn()
	}, inputHash...)
	if err != nil {
		var zero T
		return zero, err
	}
	if cached || result != nil {
		if v, ok := result.(T); ok {
			return v, nil
		}
	}
	var zero T
	return zero, nil
}
