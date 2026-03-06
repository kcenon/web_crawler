package crawler

import "context"

// Crawler defines the high-level interface for web crawling operations.
// Implementations handle the actual HTTP fetching, link extraction, and
// crawl scheduling. The gRPC server delegates all work to this interface.
type Crawler interface {
	// Crawl fetches the given URLs using the provided configuration and
	// returns results for each URL.
	Crawl(ctx context.Context, urls []string, cfg *CrawlConfig) ([]*Result, error)

	// Start launches a long-running crawler instance with the given ID
	// and configuration.
	Start(ctx context.Context, id string, cfg *CrawlConfig) error

	// Stop gracefully shuts down a running crawler and returns its final
	// statistics.
	Stop(ctx context.Context, id string) (*Stats, error)

	// Stats returns the current runtime statistics for a running crawler.
	Stats(ctx context.Context, id string) (*Stats, error)

	// AddURLs injects additional URLs into a running crawler's frontier
	// and returns the number of URLs actually added (excluding duplicates).
	AddURLs(ctx context.Context, id string, urls []string) (int, error)
}

// CrawlConfig holds the parameters for a crawl operation.
type CrawlConfig struct {
	MaxDepth         int32
	MaxPages         int32
	RespectRobotsTxt bool
	URLs             []string
}

// Result represents the outcome of crawling a single URL.
type Result struct {
	URL        string
	Content    string
	StatusCode int
	Error      error
}

// Stats holds runtime statistics for a crawler instance.
type Stats struct {
	PagesCrawled int64
	PagesFailed  int64
	PagesQueued  int64
}
