package frontier

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrClosed    = errors.New("frontier: closed")
	ErrFiltered  = errors.New("frontier: URL filtered")
	ErrDuplicate = errors.New("frontier: duplicate URL")
	ErrEmptyURL  = errors.New("frontier: empty URL")
	ErrNilEntry  = errors.New("frontier: nil entry")
)

// Config holds configuration for the in-memory frontier.
type Config struct {
	// CrawlDelay is the minimum delay between requests to the same domain.
	// A zero value means no delay. Use DefaultConfig() for sensible defaults.
	CrawlDelay time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		CrawlDelay: time.Second,
	}
}

// MemoryFrontier is an in-memory implementation of the Frontier interface.
// It uses a heap-based priority queue with deduplication and URL filtering.
type MemoryFrontier struct {
	cfg    Config
	pq     *priorityQueue
	dedup  *Deduplicator
	filter *Filter
	mu     sync.Mutex
	notify chan struct{}
	size   atomic.Int64
	closed atomic.Bool

	// Per-domain tracking for politeness.
	domainLast map[string]time.Time
	domainMu   sync.Mutex
}

// NewMemoryFrontier creates an in-memory frontier with the given config.
func NewMemoryFrontier(cfg Config) *MemoryFrontier {
	return &MemoryFrontier{
		cfg:        cfg,
		pq:         newPriorityQueue(),
		dedup:      NewDeduplicator(),
		filter:     NewFilter(),
		notify:     make(chan struct{}, 1),
		domainLast: make(map[string]time.Time),
	}
}

// Filter returns the URL filter for adding allow/deny rules.
func (f *MemoryFrontier) Filter() *Filter {
	return f.filter
}

// Add enqueues a URL entry. The URL is canonicalized, checked against
// filters and dedup before being added to the priority queue.
func (f *MemoryFrontier) Add(entry *URLEntry) error {
	if entry == nil {
		return ErrNilEntry
	}
	if entry.URL == "" {
		return ErrEmptyURL
	}
	if f.closed.Load() {
		return ErrClosed
	}

	// Canonicalize the URL.
	entry.URL = Canonicalize(entry.URL)

	// Apply filter rules.
	if !f.filter.IsAllowed(entry.URL) {
		return ErrFiltered
	}

	// Deduplicate.
	if !f.dedup.MarkSeen(entry.URL) {
		return ErrDuplicate
	}

	// Set discovery time if not set.
	if entry.DiscoveredAt.IsZero() {
		entry.DiscoveredAt = time.Now()
	}

	f.mu.Lock()
	heap.Push(f.pq, entry)
	f.mu.Unlock()
	f.size.Add(1)

	// Non-blocking notification for waiting consumers.
	select {
	case f.notify <- struct{}{}:
	default:
	}

	return nil
}

// Next returns the next URL to crawl. It blocks until a URL is available
// or the context is cancelled. Per-domain politeness delays are applied.
func (f *MemoryFrontier) Next(ctx context.Context) (*URLEntry, error) {
	for {
		if f.closed.Load() {
			return nil, ErrClosed
		}

		f.mu.Lock()
		if f.pq.Len() > 0 {
			entry := heap.Pop(f.pq).(*URLEntry)
			f.mu.Unlock()
			f.size.Add(-1)

			// Enforce per-domain crawl delay.
			f.enforceCrawlDelay(ctx, entry.URL)

			return entry, nil
		}
		f.mu.Unlock()

		// Wait for notification or context cancellation.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-f.notify:
			// New item available, loop back to check.
		}
	}
}

// Size returns the number of URLs currently in the queue.
func (f *MemoryFrontier) Size() int64 {
	return f.size.Load()
}

// Close marks the frontier as closed and wakes any waiting consumers.
func (f *MemoryFrontier) Close() error {
	f.closed.Store(true)
	// Wake up any blocked Next() calls.
	select {
	case f.notify <- struct{}{}:
	default:
	}
	return nil
}

// enforceCrawlDelay sleeps if needed to maintain the minimum delay
// between requests to the same domain.
func (f *MemoryFrontier) enforceCrawlDelay(ctx context.Context, rawURL string) {
	domain := extractDomain(rawURL)
	if domain == "" {
		return
	}

	f.domainMu.Lock()
	last, ok := f.domainLast[domain]
	now := time.Now()
	f.domainLast[domain] = now
	f.domainMu.Unlock()

	if !ok {
		return
	}

	elapsed := now.Sub(last)
	if elapsed >= f.cfg.CrawlDelay {
		return
	}

	wait := f.cfg.CrawlDelay - elapsed
	select {
	case <-time.After(wait):
	case <-ctx.Done():
	}
}

// extractDomain returns the host portion of a URL.
func extractDomain(rawURL string) string {
	// Simple extraction: find "://" then take until next "/" or end.
	idx := 0
	if i := len("://"); len(rawURL) > i {
		if pos := indexOf(rawURL, "://"); pos >= 0 {
			idx = pos + 3
		}
	}
	end := len(rawURL)
	for i := idx; i < len(rawURL); i++ {
		if rawURL[i] == '/' || rawURL[i] == '?' {
			end = i
			break
		}
	}
	host := rawURL[idx:end]
	// Strip port.
	if colonPos := lastIndexOf(host, ":"); colonPos >= 0 {
		host = host[:colonPos]
	}
	return host
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func lastIndexOf(s, sub string) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
