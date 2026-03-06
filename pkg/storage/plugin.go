package storage

import (
	"context"
	"time"
)

// Plugin defines the interface for storing crawled data.
// Implementations handle the specifics of each storage backend.
type Plugin interface {
	// Init initializes the storage backend with the given configuration.
	Init(config map[string]any) error

	// Store writes items to the storage backend.
	Store(ctx context.Context, items []Item) error

	// Close flushes pending data and releases resources.
	Close() error
}

// Item represents a single crawled page and its extracted data.
type Item struct {
	URL       string         `json:"url"`
	Data      map[string]any `json:"data,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CrawledAt time.Time      `json:"crawled_at"`
}
