package crawler

import (
	"context"
	"net/http"
	"time"
)

// Crawler defines the SDK-facing interface for web crawling operations.
// It provides a callback-driven API for processing crawled pages.
type Crawler interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Wait() error
	AddURL(url string, opts ...RequestOption) error
	AddURLs(urls []string, opts ...RequestOption) error
	OnRequest(callback RequestCallback)
	OnResponse(callback ResponseCallback)
	OnError(callback ErrorCallback)
	OnHTML(selector string, callback HTMLCallback)
	Stats() *CrawlStats
}

// CrawlStats holds runtime statistics for a crawler.
type CrawlStats struct {
	RequestCount    int64
	SuccessCount    int64
	ErrorCount      int64
	BytesDownloaded int64
	Duration        time.Duration
}

// CrawlRequest represents a request within the crawl pipeline.
type CrawlRequest struct {
	URL     string
	Method  string
	Headers map[string]string
	Depth   int
	Meta    map[string]string
}

// CrawlResponse represents a response within the crawl pipeline.
type CrawlResponse struct {
	Request     *CrawlRequest
	StatusCode  int
	Headers     http.Header
	Body        []byte
	ContentType string
}

// RequestCallback is called before a request is executed.
type RequestCallback func(req *CrawlRequest)

// ResponseCallback is called after a successful response is received.
type ResponseCallback func(resp *CrawlResponse)

// ErrorCallback is called when a request fails.
type ErrorCallback func(req *CrawlRequest, err error)

// HTMLCallback is called for HTML responses matching a CSS selector.
// Selector matching will be implemented with the data extraction engine.
type HTMLCallback func(resp *CrawlResponse)
