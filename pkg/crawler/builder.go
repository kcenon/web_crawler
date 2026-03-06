package crawler

import (
	"fmt"

	"github.com/kcenon/web_crawler/pkg/client"
)

// Builder provides a fluent API for constructing a Crawler instance.
type Builder struct {
	cfg Config
}

// NewBuilder creates a new Builder with default configuration.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithConfig sets the full configuration.
func (b *Builder) WithConfig(cfg Config) *Builder {
	b.cfg = cfg
	return b
}

// WithMaxDepth sets the maximum crawl depth.
func (b *Builder) WithMaxDepth(depth int) *Builder {
	b.cfg.MaxDepth = depth
	return b
}

// WithMaxPages sets the maximum number of pages to crawl.
func (b *Builder) WithMaxPages(pages int) *Builder {
	b.cfg.MaxPages = pages
	return b
}

// WithWorkerCount sets the number of worker goroutines.
func (b *Builder) WithWorkerCount(count int) *Builder {
	b.cfg.WorkerCount = count
	return b
}

// WithUserAgent sets the default User-Agent header.
func (b *Builder) WithUserAgent(ua string) *Builder {
	b.cfg.UserAgent = ua
	return b
}

// WithProxy configures an HTTP or SOCKS5 proxy for all requests.
func (b *Builder) WithProxy(proxyURL string) *Builder {
	b.cfg.Client.Transport.Proxy = client.ProxyConfig{URL: proxyURL}
	return b
}

// WithProxyAuth configures an authenticated proxy.
func (b *Builder) WithProxyAuth(proxyURL, username, password string) *Builder {
	b.cfg.Client.Transport.Proxy = client.ProxyConfig{
		URL:      proxyURL,
		Username: username,
		Password: password,
	}
	return b
}

// Build creates a new Crawler from the builder configuration.
func (b *Builder) Build() (Crawler, error) {
	if b.cfg.WorkerCount < 0 {
		return nil, fmt.Errorf("worker count must be non-negative")
	}
	return newEngine(b.cfg)
}

// NewEngine creates a Crawler with the given configuration.
// For more control, use NewBuilder instead.
func NewEngine(cfg Config) Crawler {
	e, _ := newEngine(cfg)
	return e
}
