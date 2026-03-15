package frontier

import "sync"

// Deduplicator tracks seen URLs to avoid processing duplicates.
type Deduplicator struct {
	mu   sync.RWMutex
	seen map[string]struct{}
}

// NewDeduplicator creates a new hash-set based deduplicator.
// capacity is a pre-allocation hint for the underlying map; 0 uses the Go default.
func NewDeduplicator(capacity int) *Deduplicator {
	return &Deduplicator{
		seen: make(map[string]struct{}, capacity),
	}
}

// IsSeen returns true if the URL has already been seen.
func (d *Deduplicator) IsSeen(url string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.seen[url]
	return ok
}

// MarkSeen marks a URL as seen. Returns true if the URL was new,
// false if it was already seen.
//
// Uses a read-before-write pattern: duplicate URLs (the common case in a
// running crawl) are detected under a shared read lock without blocking
// concurrent readers or other writers. Only genuinely new URLs escalate to
// an exclusive write lock, with a re-check to guard against a concurrent
// insert that may have occurred between the read unlock and write lock.
func (d *Deduplicator) MarkSeen(url string) bool {
	// Fast path: check under read lock (concurrent with other readers).
	d.mu.RLock()
	_, ok := d.seen[url]
	d.mu.RUnlock()
	if ok {
		return false
	}

	// Slow path: URL appears new — acquire write lock and re-check.
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.seen[url]; ok {
		return false
	}
	d.seen[url] = struct{}{}
	return true
}

// Size returns the number of unique URLs seen.
func (d *Deduplicator) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.seen)
}

// Reset clears all seen URLs.
func (d *Deduplicator) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen = make(map[string]struct{}, len(d.seen))
}
