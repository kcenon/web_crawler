package frontier

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func newTestFrontier() *MemoryFrontier {
	return NewMemoryFrontier(Config{CrawlDelay: 0})
}

func mustAdd(t *testing.T, f Frontier, entry *URLEntry) {
	t.Helper()
	if err := f.Add(entry); err != nil {
		t.Fatalf("Add(%q) error = %v", entry.URL, err)
	}
}

func mustNext(t *testing.T, f Frontier, ctx context.Context) *URLEntry {
	t.Helper()
	entry, err := f.Next(ctx)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	return entry
}

// --- Priority Queue Tests ---

func TestPriorityQueue_Ordering(t *testing.T) {
	f := newTestFrontier()
	ctx := context.Background()
	now := time.Now()

	mustAdd(t, f, &URLEntry{URL: "http://example.com/low", Priority: PriorityLow, DiscoveredAt: now})
	mustAdd(t, f, &URLEntry{URL: "http://example.com/critical", Priority: PriorityCritical, DiscoveredAt: now})
	mustAdd(t, f, &URLEntry{URL: "http://example.com/high", Priority: PriorityHigh, DiscoveredAt: now})

	// Should come out in priority order: critical, high, low.
	e1 := mustNext(t, f, ctx)
	if e1.Priority != PriorityCritical {
		t.Errorf("first = %d, want PriorityCritical(%d)", e1.Priority, PriorityCritical)
	}
	e2 := mustNext(t, f, ctx)
	if e2.Priority != PriorityHigh {
		t.Errorf("second = %d, want PriorityHigh(%d)", e2.Priority, PriorityHigh)
	}
	e3 := mustNext(t, f, ctx)
	if e3.Priority != PriorityLow {
		t.Errorf("third = %d, want PriorityLow(%d)", e3.Priority, PriorityLow)
	}
}

func TestPriorityQueue_FIFOWithinSamePriority(t *testing.T) {
	f := newTestFrontier()
	ctx := context.Background()

	t1 := time.Now()
	t2 := t1.Add(time.Second)
	t3 := t2.Add(time.Second)

	mustAdd(t, f, &URLEntry{URL: "http://a.com", Priority: PriorityNormal, DiscoveredAt: t1})
	mustAdd(t, f, &URLEntry{URL: "http://b.com", Priority: PriorityNormal, DiscoveredAt: t2})
	mustAdd(t, f, &URLEntry{URL: "http://c.com", Priority: PriorityNormal, DiscoveredAt: t3})

	e1 := mustNext(t, f, ctx)
	e2 := mustNext(t, f, ctx)
	e3 := mustNext(t, f, ctx)

	if e1.URL != "http://a.com" || e2.URL != "http://b.com" || e3.URL != "http://c.com" {
		t.Errorf("FIFO order broken: got %s, %s, %s", e1.URL, e2.URL, e3.URL)
	}
}

// --- Deduplication Tests ---

func TestDedup_RejectsDuplicate(t *testing.T) {
	f := newTestFrontier()

	mustAdd(t, f, &URLEntry{URL: "http://example.com/page"})
	err := f.Add(&URLEntry{URL: "http://example.com/page"})
	if err != ErrDuplicate {
		t.Errorf("duplicate Add error = %v, want ErrDuplicate", err)
	}

	if f.Size() != 1 {
		t.Errorf("Size = %d, want 1", f.Size())
	}
}

func TestDedup_CanonicalizationDedup(t *testing.T) {
	f := newTestFrontier()

	mustAdd(t, f, &URLEntry{URL: "http://example.com/page/"})
	err := f.Add(&URLEntry{URL: "http://EXAMPLE.COM/page"})
	if err != ErrDuplicate {
		t.Errorf("canonical duplicate error = %v, want ErrDuplicate", err)
	}
}

func TestDedup_Standalone(t *testing.T) {
	d := NewDeduplicator(0)

	if d.IsSeen("http://example.com") {
		t.Error("should not be seen initially")
	}

	if !d.MarkSeen("http://example.com") {
		t.Error("first MarkSeen should return true")
	}

	if d.MarkSeen("http://example.com") {
		t.Error("second MarkSeen should return false")
	}

	if d.Size() != 1 {
		t.Errorf("Size = %d, want 1", d.Size())
	}

	d.Reset()
	if d.Size() != 0 {
		t.Errorf("Size after Reset = %d, want 0", d.Size())
	}
}

func TestMemoryFrontier_InitialCapacity(t *testing.T) {
	const n = 1000
	f := NewMemoryFrontier(Config{InitialCapacity: n})

	for i := 0; i < n; i++ {
		err := f.Add(&URLEntry{
			URL:      fmt.Sprintf("http://example.com/page/%d", i),
			Priority: PriorityNormal,
		})
		if err != nil {
			t.Fatalf("Add[%d] unexpected error: %v", i, err)
		}
	}

	if got := f.Size(); got != n {
		t.Errorf("Size = %d, want %d", got, n)
	}
}

// --- Canonicalization Tests ---

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HTTP://EXAMPLE.COM/path", "http://example.com/path"},
		{"http://example.com:80/path", "http://example.com/path"},
		{"https://example.com:443/path", "https://example.com/path"},
		{"http://example.com:8080/path", "http://example.com:8080/path"},
		{"http://example.com/path/", "http://example.com/path"},
		{"http://example.com/", "http://example.com/"},
		{"http://example.com/path#fragment", "http://example.com/path"},
		{"http://example.com/path?b=2&a=1", "http://example.com/path?a=1&b=2"},
		{"http://example.com/path?a=2&a=1", "http://example.com/path?a=1&a=2"},
	}

	for _, tt := range tests {
		got := Canonicalize(tt.input)
		if got != tt.want {
			t.Errorf("Canonicalize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- Filter Tests ---

func TestFilter_AllowDenyRules(t *testing.T) {
	fl := NewFilter()
	if err := fl.AddDenyRule(`\.pdf$`); err != nil {
		t.Fatal(err)
	}
	if err := fl.AddAllowRule(`^https://allowed\.com`); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		url  string
		want bool
	}{
		{"http://example.com/page", true},      // no rule matches, default allow
		{"http://example.com/file.pdf", false}, // denied by .pdf rule
		{"https://allowed.com/file.pdf", true}, // allow rule matches first (if added first)
	}

	// Re-create with order: allow first, deny second.
	fl = NewFilter()
	if err := fl.AddAllowRule(`^https://allowed\.com`); err != nil {
		t.Fatal(err)
	}
	if err := fl.AddDenyRule(`\.pdf$`); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		got := fl.IsAllowed(tt.url)
		if got != tt.want {
			t.Errorf("IsAllowed(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestFilter_FrontierIntegration(t *testing.T) {
	f := newTestFrontier()
	if err := f.Filter().AddDenyRule(`/blocked`); err != nil {
		t.Fatal(err)
	}

	err := f.Add(&URLEntry{URL: "http://example.com/blocked/page"})
	if err != ErrFiltered {
		t.Errorf("filtered Add error = %v, want ErrFiltered", err)
	}

	mustAdd(t, f, &URLEntry{URL: "http://example.com/allowed"})
	if f.Size() != 1 {
		t.Errorf("Size = %d, want 1", f.Size())
	}
}

// --- Frontier Lifecycle Tests ---

func TestFrontier_Size(t *testing.T) {
	f := newTestFrontier()
	ctx := context.Background()

	if f.Size() != 0 {
		t.Errorf("initial Size = %d, want 0", f.Size())
	}

	mustAdd(t, f, &URLEntry{URL: "http://a.com"})
	mustAdd(t, f, &URLEntry{URL: "http://b.com"})

	if f.Size() != 2 {
		t.Errorf("Size = %d, want 2", f.Size())
	}

	mustNext(t, f, ctx)
	if f.Size() != 1 {
		t.Errorf("Size after Next = %d, want 1", f.Size())
	}
}

func TestFrontier_Close(t *testing.T) {
	f := newTestFrontier()

	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	err := f.Add(&URLEntry{URL: "http://example.com"})
	if err != ErrClosed {
		t.Errorf("Add after Close = %v, want ErrClosed", err)
	}

	_, err = f.Next(context.Background())
	if err != ErrClosed {
		t.Errorf("Next after Close = %v, want ErrClosed", err)
	}
}

func TestFrontier_NextBlocksUntilAdd(t *testing.T) {
	f := newTestFrontier()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var got *URLEntry
	done := make(chan struct{})
	go func() {
		var err error
		got, err = f.Next(ctx)
		if err != nil {
			t.Errorf("Next() error = %v", err)
		}
		close(done)
	}()

	// Small delay to ensure Next is blocking.
	time.Sleep(50 * time.Millisecond)

	mustAdd(t, f, &URLEntry{URL: "http://example.com"})

	select {
	case <-done:
		if got == nil || got.URL != "http://example.com" {
			t.Errorf("Next() returned %v, want http://example.com", got)
		}
	case <-ctx.Done():
		t.Fatal("Next() did not unblock after Add")
	}
}

func TestFrontier_NextCancelledContext(t *testing.T) {
	f := newTestFrontier()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.Next(ctx)
	if err != context.Canceled {
		t.Errorf("Next() error = %v, want context.Canceled", err)
	}
}

func TestFrontier_NilEntry(t *testing.T) {
	f := newTestFrontier()
	if err := f.Add(nil); err != ErrNilEntry {
		t.Errorf("Add(nil) error = %v, want ErrNilEntry", err)
	}
}

func TestFrontier_EmptyURL(t *testing.T) {
	f := newTestFrontier()
	if err := f.Add(&URLEntry{URL: ""}); err != ErrEmptyURL {
		t.Errorf("Add(empty) error = %v, want ErrEmptyURL", err)
	}
}

// --- Concurrency Test ---

func TestFrontier_ConcurrentAddAndNext(t *testing.T) {
	f := newTestFrontier()
	ctx := context.Background()

	const n = 100

	// Add all items first.
	for i := 0; i < n; i++ {
		mustAdd(t, f, &URLEntry{
			URL:      "http://example.com/" + itoa(i),
			Priority: PriorityNormal,
		})
	}

	// Consume with multiple goroutines.
	var consumed sync.WaitGroup
	consumed.Add(n)

	for j := 0; j < 4; j++ {
		go func() {
			for {
				_, err := f.Next(ctx)
				if err != nil {
					return
				}
				consumed.Done()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		consumed.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All items consumed.
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for all items to be consumed")
	}

	if f.Size() != 0 {
		t.Errorf("Size after consuming all = %d, want 0", f.Size())
	}
}

// --- Crawl Delay Test ---

func TestFrontier_CrawlDelay(t *testing.T) {
	delay := 100 * time.Millisecond
	f := NewMemoryFrontier(Config{CrawlDelay: delay})
	ctx := context.Background()

	// Add two URLs from the same domain.
	mustAdd(t, f, &URLEntry{URL: "http://example.com/a", Priority: PriorityCritical})
	mustAdd(t, f, &URLEntry{URL: "http://example.com/b", Priority: PriorityCritical})

	start := time.Now()
	mustNext(t, f, ctx) // First should return immediately.
	mustNext(t, f, ctx) // Second should wait for crawl delay.
	elapsed := time.Since(start)

	// Should take at least the crawl delay.
	if elapsed < delay {
		t.Errorf("elapsed %v < crawl delay %v", elapsed, delay)
	}
}

func TestFrontier_DiscoveredAtAutoSet(t *testing.T) {
	f := newTestFrontier()
	ctx := context.Background()

	before := time.Now()
	mustAdd(t, f, &URLEntry{URL: "http://example.com"})
	entry := mustNext(t, f, ctx)

	if entry.DiscoveredAt.Before(before) {
		t.Error("DiscoveredAt should be set automatically")
	}
}

// itoa is a simple int-to-string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
