package frontier

import (
	"context"
	"time"
)

// Priority represents the crawl priority of a URL.
// Lower values indicate higher priority.
type Priority int

const (
	PriorityCritical Priority = 0
	PriorityHigh     Priority = 1
	PriorityNormal   Priority = 2
	PriorityLow      Priority = 3
)

// URLEntry represents a single URL to be crawled, along with scheduling metadata.
type URLEntry struct {
	URL          string
	Priority     Priority
	Depth        int
	Metadata     map[string]string
	DiscoveredAt time.Time
}

// Frontier manages URL scheduling for the crawler.
// It determines which URLs are crawled and in what order.
type Frontier interface {
	// Add enqueues a URL entry for crawling.
	Add(entry *URLEntry) error

	// Next returns the next URL to crawl, blocking until one is available
	// or the context is cancelled.
	Next(ctx context.Context) (*URLEntry, error)

	// Size returns the number of URLs currently queued.
	Size() int64

	// Close releases resources held by the frontier.
	Close() error
}
