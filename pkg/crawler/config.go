package crawler

import (
	"net/http"

	"github.com/kcenon/web_crawler/pkg/client"
)

// Config holds the configuration for the crawler engine.
type Config struct {
	// MaxDepth limits how deep the crawler follows links. Default: 3.
	MaxDepth int

	// MaxPages limits the total number of pages to crawl. Zero means no limit.
	MaxPages int

	// Concurrency controls global and per-domain concurrency limits.
	Concurrency ConcurrencyConfig

	// Client configures the underlying HTTP client.
	Client client.Config

	// UserAgent sets the default User-Agent header. Default: "web_crawler/0.1".
	UserAgent string

	// WorkerCount is the number of concurrent worker goroutines. Default: 10.
	WorkerCount int

	// CookieJar, if non-nil, enables cookie management for all requests.
	CookieJar http.CookieJar
}

func (c Config) withDefaults() Config {
	if c.MaxDepth == 0 {
		c.MaxDepth = 3
	}
	if c.WorkerCount == 0 {
		c.WorkerCount = 10
	}
	if c.UserAgent == "" {
		c.UserAgent = "web_crawler/0.1"
	}
	c.Concurrency = c.Concurrency.withDefaults()
	return c
}
