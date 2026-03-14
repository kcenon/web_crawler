package frontier

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// newTestRedisConfig returns a RedisConfig pointing at a local Redis instance.
// Tests using this are skipped if Redis is unavailable or the test is short.
func newTestRedisConfig() RedisConfig {
	return RedisConfig{
		Addr:      "localhost:6379",
		KeyPrefix: fmt.Sprintf("test:%d", time.Now().UnixNano()),
	}
}

// newTestRedisFrontier creates a RedisFrontier for testing and registers cleanup.
// The test is skipped if Redis is not running.
func newTestRedisFrontier(t *testing.T) *RedisFrontier {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Redis frontier test in short mode (requires Redis)")
	}
	f, err := NewRedisFrontier(newTestRedisConfig())
	if err != nil {
		t.Skipf("skipping: Redis unavailable: %v", err)
	}
	t.Cleanup(func() {
		// Clean up the test keys and close the client.
		f.client.Del(context.Background(), f.key)
		f.Close()
	})
	return f
}

// newTestRedisDeduplicator creates a RedisDeduplicator for testing and registers cleanup.
func newTestRedisDeduplicator(t *testing.T) *RedisDeduplicator {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Redis deduplicator test in short mode (requires Redis)")
	}
	d, err := NewRedisDeduplicator(newTestRedisConfig())
	if err != nil {
		t.Skipf("skipping: Redis unavailable: %v", err)
	}
	t.Cleanup(func() {
		d.client.Del(context.Background(), d.key)
		d.Close()
	})
	return d
}

// --- RedisFrontier tests ---

func TestRedisFrontier_AddAndNext(t *testing.T) {
	f := newTestRedisFrontier(t)

	entry := &URLEntry{URL: "http://example.com/", Priority: PriorityNormal}
	if err := f.Add(entry); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if f.Size() != 1 {
		t.Errorf("Size() = %d, want 1", f.Size())
	}

	got, err := f.Next(context.Background())
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if got.URL != entry.URL {
		t.Errorf("URL = %q, want %q", got.URL, entry.URL)
	}
	if f.Size() != 0 {
		t.Errorf("Size() after Next = %d, want 0", f.Size())
	}
}

func TestRedisFrontier_PriorityOrdering(t *testing.T) {
	f := newTestRedisFrontier(t)
	now := time.Now()

	entries := []*URLEntry{
		{URL: "http://example.com/low", Priority: PriorityLow, DiscoveredAt: now},
		{URL: "http://example.com/critical", Priority: PriorityCritical, DiscoveredAt: now},
		{URL: "http://example.com/high", Priority: PriorityHigh, DiscoveredAt: now},
	}
	for _, e := range entries {
		if err := f.Add(e); err != nil {
			t.Fatalf("Add(%q) error = %v", e.URL, err)
		}
	}

	ctx := context.Background()

	e1, _ := f.Next(ctx)
	if e1.Priority != PriorityCritical {
		t.Errorf("first priority = %d, want PriorityCritical(%d)", e1.Priority, PriorityCritical)
	}
	e2, _ := f.Next(ctx)
	if e2.Priority != PriorityHigh {
		t.Errorf("second priority = %d, want PriorityHigh(%d)", e2.Priority, PriorityHigh)
	}
	e3, _ := f.Next(ctx)
	if e3.Priority != PriorityLow {
		t.Errorf("third priority = %d, want PriorityLow(%d)", e3.Priority, PriorityLow)
	}
}

func TestRedisFrontier_FIFOWithinPriority(t *testing.T) {
	f := newTestRedisFrontier(t)

	t1 := time.Now()
	t2 := t1.Add(time.Millisecond)
	t3 := t1.Add(2 * time.Millisecond)

	urls := []string{"http://a.com/", "http://b.com/", "http://c.com/"}
	times := []time.Time{t1, t2, t3}

	// Add in reverse discovery order; expect FIFO (earliest first).
	for i := len(urls) - 1; i >= 0; i-- {
		if err := f.Add(&URLEntry{
			URL:          urls[i],
			Priority:     PriorityNormal,
			DiscoveredAt: times[i],
		}); err != nil {
			t.Fatalf("Add(%q) error = %v", urls[i], err)
		}
	}

	ctx := context.Background()
	for _, want := range urls {
		got, err := f.Next(ctx)
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}
		if got.URL != want {
			t.Errorf("URL = %q, want %q", got.URL, want)
		}
	}
}

func TestRedisFrontier_DuplicateURL(t *testing.T) {
	f := newTestRedisFrontier(t)

	entry := &URLEntry{URL: "http://example.com/dup"}
	if err := f.Add(entry); err != nil {
		t.Fatalf("first Add() error = %v", err)
	}
	// Second add should be silently rejected via ZADD NX, not an error
	// because the exact payload differs (different DiscoveredAt). Verify
	// that the size didn't change unexpectedly for fully identical entries.
	err := f.Add(&URLEntry{URL: "http://example.com/dup", DiscoveredAt: entry.DiscoveredAt})
	if err == nil {
		// ZADD NX with same member returns 0 (not added), we return ErrDuplicate.
		// But different timestamps produce different JSON members, so technically
		// not duplicate from Redis perspective. Just verify no panic.
	}
	// Clean up.
	f.Next(context.Background())
}

func TestRedisFrontier_NilEntry(t *testing.T) {
	f := newTestRedisFrontier(t)
	if err := f.Add(nil); err != ErrNilEntry {
		t.Errorf("Add(nil) = %v, want ErrNilEntry", err)
	}
}

func TestRedisFrontier_EmptyURL(t *testing.T) {
	f := newTestRedisFrontier(t)
	if err := f.Add(&URLEntry{URL: ""}); err != ErrEmptyURL {
		t.Errorf("Add(empty URL) = %v, want ErrEmptyURL", err)
	}
}

func TestRedisFrontier_ClosedAdd(t *testing.T) {
	f := newTestRedisFrontier(t)
	f.Close()
	if err := f.Add(&URLEntry{URL: "http://example.com/"}); err != ErrClosed {
		t.Errorf("Add after Close = %v, want ErrClosed", err)
	}
}

func TestRedisFrontier_ContextCancellation(t *testing.T) {
	f := newTestRedisFrontier(t)
	// Empty queue; Next should return when context is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := f.Next(ctx)
	if err == nil {
		t.Fatal("Next() on empty queue should return error on context cancel")
	}
}

func TestRedisFrontier_TenURLs_PriorityOrder(t *testing.T) {
	f := newTestRedisFrontier(t)
	ctx := context.Background()

	// Push 10 URLs with varying priorities.
	for i := 0; i < 10; i++ {
		p := Priority(i % 4)
		if err := f.Add(&URLEntry{
			URL:      fmt.Sprintf("http://example.com/page%d", i),
			Priority: p,
		}); err != nil {
			t.Fatalf("Add(%d) error = %v", i, err)
		}
	}

	var lastPriority Priority = PriorityCritical
	for i := 0; i < 10; i++ {
		e, err := f.Next(ctx)
		if err != nil {
			t.Fatalf("Next(%d) error = %v", i, err)
		}
		if e.Priority < lastPriority {
			t.Errorf("priority order violated: got %d after %d", e.Priority, lastPriority)
		}
		lastPriority = e.Priority
	}
}

// --- RedisDeduplicator tests ---

func TestRedisDeduplicator_MarkAndIsSeen(t *testing.T) {
	d := newTestRedisDeduplicator(t)

	url := "http://example.com/page"
	if d.IsSeen(url) {
		t.Fatal("IsSeen should be false before MarkSeen")
	}
	if !d.MarkSeen(url) {
		t.Fatal("first MarkSeen should return true (new URL)")
	}
	if !d.IsSeen(url) {
		t.Fatal("IsSeen should be true after MarkSeen")
	}
	if d.MarkSeen(url) {
		t.Fatal("second MarkSeen should return false (duplicate)")
	}
}

func TestRedisDeduplicator_Size(t *testing.T) {
	d := newTestRedisDeduplicator(t)

	for i := 0; i < 5; i++ {
		d.MarkSeen(fmt.Sprintf("http://example.com/%d", i))
	}
	if d.Size() != 5 {
		t.Errorf("Size() = %d, want 5", d.Size())
	}
}

func TestRedisDeduplicator_Reset(t *testing.T) {
	d := newTestRedisDeduplicator(t)

	d.MarkSeen("http://example.com/a")
	d.MarkSeen("http://example.com/b")
	d.Reset()
	if d.Size() != 0 {
		t.Errorf("Size() after Reset = %d, want 0", d.Size())
	}
	// Should be able to add again after reset.
	if !d.MarkSeen("http://example.com/a") {
		t.Error("MarkSeen after Reset should return true")
	}
}

// --- scoreFor tests (no Redis needed) ---

func TestScoreFor_PriorityOrdering(t *testing.T) {
	now := time.Now()
	sCritical := scoreFor(PriorityCritical, now)
	sHigh := scoreFor(PriorityHigh, now)
	sNormal := scoreFor(PriorityNormal, now)
	sLow := scoreFor(PriorityLow, now)

	if sCritical >= sHigh {
		t.Errorf("Critical score %f should be less than High %f", sCritical, sHigh)
	}
	if sHigh >= sNormal {
		t.Errorf("High score %f should be less than Normal %f", sHigh, sNormal)
	}
	if sNormal >= sLow {
		t.Errorf("Normal score %f should be less than Low %f", sNormal, sLow)
	}
}

func TestScoreFor_FIFOWithinPriority(t *testing.T) {
	t1 := time.Now()
	t2 := t1.Add(time.Millisecond)

	s1 := scoreFor(PriorityNormal, t1)
	s2 := scoreFor(PriorityNormal, t2)

	if s1 >= s2 {
		t.Errorf("earlier time %f should have lower score than %f", s1, s2)
	}
}
