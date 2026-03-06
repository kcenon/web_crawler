package crawler

import "context"

// Service defines the interface used by the gRPC server to manage
// multiple crawler instances. It provides a multi-tenant management layer
// on top of individual Crawler instances.
type Service interface {
	Crawl(ctx context.Context, urls []string, cfg *CrawlConfig) ([]*Result, error)
	Start(ctx context.Context, id string, cfg *CrawlConfig) error
	Stop(ctx context.Context, id string) (*Stats, error)
	Stats(ctx context.Context, id string) (*Stats, error)
	AddURLs(ctx context.Context, id string, urls []string) (int, error)
}

// CrawlConfig holds the parameters for a server-managed crawl operation.
type CrawlConfig struct {
	MaxDepth         int32
	MaxPages         int32
	RespectRobotsTxt bool
	URLs             []string
}

// Result represents the outcome of crawling a single URL via the server.
type Result struct {
	URL        string
	Content    string
	StatusCode int
	Error      error
}

// Stats holds runtime statistics for a server-managed crawler instance.
type Stats struct {
	PagesCrawled int64
	PagesFailed  int64
	PagesQueued  int64
}
