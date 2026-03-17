// Package plugin defines the extensible plugin system for the web crawler SDK.
//
// It provides a base Plugin interface and typed interfaces for different plugin
// categories (storage, parser, notifier, exporter). Plugins are managed through
// a thread-safe Registry that supports registration, retrieval, and lifecycle
// management.
package plugin

import (
	"context"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/storage"
)

// Plugin is the base interface that all plugins must implement.
// It provides identity, initialization, and cleanup capabilities.
type Plugin interface {
	// Name returns a unique identifier for the plugin.
	Name() string

	// Init initializes the plugin with the given configuration.
	// Called once before the plugin is used.
	Init(config map[string]any) error

	// Close releases any resources held by the plugin.
	// Called when the plugin is no longer needed.
	Close() error
}

// StoragePlugin stores crawled data to a backend system.
type StoragePlugin interface {
	Plugin

	// Store writes items to the storage backend.
	Store(ctx context.Context, items []storage.Item) error
}

// ParserPlugin extracts structured data from crawl responses.
type ParserPlugin interface {
	Plugin

	// CanParse reports whether this parser can handle the given content type.
	CanParse(contentType string) bool

	// Parse extracts data from the crawl response.
	Parse(ctx context.Context, resp *crawler.CrawlResponse) (*ParseResult, error)
}

// ParseResult holds the extracted data from a parser plugin.
type ParseResult struct {
	// Data contains the extracted key-value pairs.
	Data map[string]any

	// Links contains discovered URLs for further crawling.
	Links []string
}

// NotifierPlugin sends notifications about crawl events.
type NotifierPlugin interface {
	Plugin

	// Notify sends a notification for the given event.
	Notify(ctx context.Context, event *CrawlEvent) error
}

// EventType identifies the kind of crawl event.
type EventType string

const (
	EventStarted   EventType = "started"
	EventCompleted EventType = "completed"
	EventError     EventType = "error"
	EventThreshold EventType = "threshold"
)

// CrawlEvent represents a notable occurrence during a crawl.
type CrawlEvent struct {
	Type    EventType
	Message string
	Data    map[string]any
}

// ExporterPlugin exports crawled data to external systems.
type ExporterPlugin interface {
	Plugin

	// Export sends items to an external system in the specified format.
	Export(ctx context.Context, items []storage.Item, format string) error
}
