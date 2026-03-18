package idempotent_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rai/clean-modularmonolith-go/modules/shared/idempotent"
)

// --- Once with hash ---

func TestOnce_WithHash_FirstCall_Executes(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	err := b.Once("op", "k", func() error {
		called++
		return nil
	}, b.HashInput("v1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected fn called once, got %d", called)
	}
}

func TestOnce_WithHash_SameHash_Skips(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	fn := func() error { called++; return nil }
	h := b.HashInput("v1")

	if err := b.Once("op", "k", fn, h); err != nil {
		t.Fatal(err)
	}
	if err := b.Once("op", "k", fn, h); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("expected fn called once, got %d", called)
	}
}

func TestOnce_WithHash_DifferentHash_Executes(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	fn := func() error { called++; return nil }

	if err := b.Once("op", "k", fn, b.HashInput("v1")); err != nil {
		t.Fatal(err)
	}
	if err := b.Once("op", "k", fn, b.HashInput("v2")); err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected fn called twice (different hash), got %d", called)
	}
}

func TestOnce_WithHash_ErrorNotCached(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	errBoom := errors.New("boom")
	h := b.HashInput("v1")

	err := b.Once("op", "k", func() error {
		called++
		return errBoom
	}, h)
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected errBoom, got %v", err)
	}

	// Retry should execute fn again because failure was not cached.
	err = b.Once("op", "k", func() error {
		called++
		return nil
	}, h)
	if err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected fn called twice (retry after error), got %d", called)
	}
}

func TestOnce_WithHash_Expired_Executes(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache(1 * time.Millisecond)
	defer cleanup()

	var called int
	fn := func() error { called++; return nil }
	h := b.HashInput("v1")

	if err := b.Once("op", "k", fn, h); err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Millisecond)

	if err := b.Once("op", "k", fn, h); err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected fn called twice (after TTL expiry), got %d", called)
	}
}

func TestOnce_WithHash_UpdatesHashOnReexecution(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	fn := func() error { called++; return nil }
	h1 := b.HashInput("v1")
	h2 := b.HashInput("v2")

	if err := b.Once("op", "k", fn, h1); err != nil {
		t.Fatal(err)
	}
	if err := b.Once("op", "k", fn, h2); err != nil {
		t.Fatal(err)
	}
	// h2 again → skip (cache now holds h2)
	if err := b.Once("op", "k", fn, h2); err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected fn called twice (h1, h2), got %d", called)
	}
}

// --- Once without hash ---

func TestOnce_NoHash_FirstCall_Executes(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	err := b.Once("op", "k", func() error {
		called++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Fatalf("expected fn called once, got %d", called)
	}
}

func TestOnce_NoHash_SecondCall_Blocked(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	fn := func() error { called++; return nil }

	if err := b.Once("op", "k", fn); err != nil {
		t.Fatal(err)
	}
	if err := b.Once("op", "k", fn); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("expected fn called once (second blocked), got %d", called)
	}
}

func TestOnce_NoHash_ErrorNotCached(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	errBoom := errors.New("boom")

	err := b.Once("op", "k", func() error {
		called++
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected errBoom, got %v", err)
	}

	err = b.Once("op", "k", func() error {
		called++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected fn called twice (retry after error), got %d", called)
	}
}

func TestOnce_NoHash_Expired_Executes(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache(1 * time.Millisecond)
	defer cleanup()

	var called int
	fn := func() error { called++; return nil }

	if err := b.Once("op", "k", fn); err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Millisecond)

	if err := b.Once("op", "k", fn); err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected fn called twice (after TTL expiry), got %d", called)
	}
}

func TestOnce_NoHash_DifferentKeys_Independent(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	fn := func() error { called++; return nil }

	if err := b.Once("op", "k1", fn); err != nil {
		t.Fatal(err)
	}
	if err := b.Once("op", "k2", fn); err != nil {
		t.Fatal(err)
	}
	if called != 2 {
		t.Fatalf("expected fn called twice (different keys), got %d", called)
	}
}

// --- HashInput ---

func TestHashInput_Deterministic(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	h1 := b.HashInput("a", "b", "c")
	h2 := b.HashInput("a", "b", "c")
	if h1 != h2 {
		t.Fatal("expected same hash for same inputs")
	}
}

func TestHashInput_DifferentArgs_DifferentHash(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	h1 := b.HashInput("a", "b")
	h2 := b.HashInput("a", "c")
	if h1 == h2 {
		t.Fatal("expected different hashes for different inputs")
	}
}

func TestHashInput_BoundaryCollisionPrevented(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	h1 := b.HashInput("ab", "c")
	h2 := b.HashInput("a", "bc")
	if h1 == h2 {
		t.Fatal("expected different hashes: JSON array encoding should prevent boundary collision")
	}
}

func TestHashInput_LeadingEmptyStringDiffers(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	h1 := b.HashInput("", "x")
	h2 := b.HashInput("x")
	if h1 == h2 {
		t.Fatal("expected different hashes: leading empty string should produce different hash")
	}
}

func TestHashInput_MixedTypes(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	h1 := b.HashInput("user1", 42, true)
	h2 := b.HashInput("user1", 42, true)
	if h1 != h2 {
		t.Fatal("expected same hash for same mixed-type inputs")
	}

	h3 := b.HashInput("user1", 43, true)
	if h1 == h3 {
		t.Fatal("expected different hash when a value differs")
	}
}

func TestHashInput_Struct(t *testing.T) {
	type Doc struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Score int    `json:"score"`
	}

	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	h1 := b.HashInput(Doc{ID: "1", Name: "alice", Score: 100})
	h2 := b.HashInput(Doc{ID: "1", Name: "alice", Score: 100})
	if h1 != h2 {
		t.Fatal("expected same hash for same struct")
	}

	h3 := b.HashInput(Doc{ID: "1", Name: "alice", Score: 200})
	if h1 == h3 {
		t.Fatal("expected different hash when struct field differs")
	}
}

func TestHashInput_PanicsOnUnsupportedType(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unsupported type (channel)")
		}
	}()
	b.HashInput(make(chan int))
}

// --- Concurrency ---

func TestOnce_WithHash_ConcurrentSafe(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var count atomic.Int64
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h := b.HashInput("data")
			if i%2 == 0 {
				h = b.HashInput("other")
			}
			_ = b.Once("op", "k", func() error {
				count.Add(1)
				return nil
			}, h)
		}()
	}
	wg.Wait()

	if count.Load() == 0 {
		t.Fatal("expected at least one execution")
	}
}

// --- OnceResult ---

func TestOnceResult_FirstCall_ReturnsResult(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	result, err := idempotent.OnceResult(b, "op", "k", func() (string, error) {
		return "hello", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestOnceResult_SameKey_ReturnsCachedResult(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int
	fn := func() (int, error) {
		called++
		return 42, nil
	}

	r1, err := idempotent.OnceResult(b, "op", "k", fn)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := idempotent.OnceResult(b, "op", "k", fn)
	if err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("expected fn called once, got %d", called)
	}
	if r1 != 42 || r2 != 42 {
		t.Fatalf("expected 42 both times, got %d, %d", r1, r2)
	}
}

func TestOnceResult_WithHash_DifferentHash_Reexecutes(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var called int

	r1, err := idempotent.OnceResult(b, "op", "k", func() (string, error) {
		called++
		return "first", nil
	}, b.HashInput("v1"))
	if err != nil {
		t.Fatal(err)
	}

	r2, err := idempotent.OnceResult(b, "op", "k", func() (string, error) {
		called++
		return "second", nil
	}, b.HashInput("v2"))
	if err != nil {
		t.Fatal(err)
	}

	if called != 2 {
		t.Fatalf("expected fn called twice, got %d", called)
	}
	if r1 != "first" || r2 != "second" {
		t.Fatalf("expected 'first','second', got %q,%q", r1, r2)
	}
}

func TestOnceResult_ErrorNotCached(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	errBoom := errors.New("boom")

	_, err := idempotent.OnceResult(b, "op", "k", func() (string, error) {
		return "", errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected errBoom, got %v", err)
	}

	result, err := idempotent.OnceResult(b, "op", "k", func() (string, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "ok" {
		t.Fatalf("expected 'ok', got %q", result)
	}
}

func TestOnceResult_CachedResultPersistsThroughOnce(t *testing.T) {
	// Once and OnceResult share the same cache. If Once writes first,
	// OnceResult should still hit the cache (with zero-value result).
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()

	if err := b.Once("op", "k", func() error { return nil }); err != nil {
		t.Fatal(err)
	}

	result, err := idempotent.OnceResult(b, "op", "k", func() (string, error) {
		t.Fatal("fn should not be called")
		return "unreachable", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Fatalf("expected zero value, got %q", result)
	}
}

func TestOnce_NoHash_ConcurrentSafe(t *testing.T) {
	b, cleanup := idempotent.NewOutboundCache()
	defer cleanup()
	var count atomic.Int64
	var wg sync.WaitGroup

	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Once("op", "k", func() error {
				count.Add(1)
				return nil
			})
		}()
	}
	wg.Wait()

	if count.Load() == 0 {
		t.Fatal("expected at least one execution")
	}
}
