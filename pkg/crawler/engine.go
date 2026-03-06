package crawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kcenon/web_crawler/pkg/client"
)

// engineStats holds atomic counters for crawler statistics.
type engineStats struct {
	requestCount    atomic.Int64
	successCount    atomic.Int64
	errorCount      atomic.Int64
	bytesDownloaded atomic.Int64
}

// Engine implements the Crawler interface with a worker pool architecture.
type Engine struct {
	cfg         Config
	httpClient  *client.Client
	concurrency *ConcurrencyManager

	urlQueue chan *CrawlRequest

	cbMu        sync.RWMutex
	requestCBs  []RequestCallback
	responseCBs []ResponseCallback
	errorCBs    []ErrorCallback
	htmlCBs     map[string][]HTMLCallback

	stats     engineStats
	started   atomic.Bool
	stopped   atomic.Bool
	startTime time.Time

	ctx      context.Context
	cancel   context.CancelFunc
	workerWg sync.WaitGroup // tracks worker goroutines
	urlWg    sync.WaitGroup // tracks pending URLs
}

// newEngine creates a new Engine from the given configuration.
func newEngine(cfg Config) (*Engine, error) {
	cfg = cfg.withDefaults()

	httpClient, err := client.New(client.Config{
		UserAgent: cfg.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	return &Engine{
		cfg:         cfg,
		httpClient:  httpClient,
		concurrency: NewConcurrencyManager(cfg.Concurrency),
		urlQueue:    make(chan *CrawlRequest, 10000),
		htmlCBs:     make(map[string][]HTMLCallback),
	}, nil
}

// Start launches the worker pool and begins processing queued URLs.
func (e *Engine) Start(ctx context.Context) error {
	if e.started.Swap(true) {
		return fmt.Errorf("crawler already started")
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.startTime = time.Now()

	for i := 0; i < e.cfg.WorkerCount; i++ {
		e.workerWg.Add(1)
		go e.worker()
	}

	return nil
}

// Stop cancels the crawling context and signals workers to exit.
func (e *Engine) Stop(_ context.Context) error {
	if e.stopped.Swap(true) {
		return nil
	}
	if e.cancel != nil {
		e.cancel()
	}
	return nil
}

// Wait blocks until all queued URLs have been processed or the crawler
// is stopped. It shuts down workers and releases resources before returning.
func (e *Engine) Wait() error {
	if !e.started.Load() {
		return fmt.Errorf("crawler not started")
	}

	waitDone := make(chan struct{})
	go func() {
		e.urlWg.Wait()
		close(waitDone)
	}()

	var result error
	select {
	case <-waitDone:
		// All URLs processed; stop workers.
		e.cancel()
	case <-e.ctx.Done():
		// Stop was called; drain remaining queue items.
		for {
			select {
			case <-e.urlQueue:
				e.urlWg.Done()
			default:
				result = e.ctx.Err()
				goto cleanup
			}
		}
	}

cleanup:
	e.workerWg.Wait()
	e.httpClient.Close()
	return result
}

// AddURL adds a single URL to the crawl queue.
func (e *Engine) AddURL(rawURL string, opts ...RequestOption) error {
	if e.stopped.Load() {
		return fmt.Errorf("crawler is stopped")
	}

	req := &CrawlRequest{URL: rawURL}
	for _, opt := range opts {
		opt(req)
	}

	e.urlWg.Add(1)
	select {
	case e.urlQueue <- req:
		return nil
	default:
		e.urlWg.Done()
		return fmt.Errorf("url queue is full")
	}
}

// AddURLs adds multiple URLs to the crawl queue.
func (e *Engine) AddURLs(urls []string, opts ...RequestOption) error {
	for _, u := range urls {
		if err := e.AddURL(u, opts...); err != nil {
			return err
		}
	}
	return nil
}

// OnRequest registers a callback invoked before each request.
func (e *Engine) OnRequest(cb RequestCallback) {
	e.cbMu.Lock()
	e.requestCBs = append(e.requestCBs, cb)
	e.cbMu.Unlock()
}

// OnResponse registers a callback invoked after each successful response.
func (e *Engine) OnResponse(cb ResponseCallback) {
	e.cbMu.Lock()
	e.responseCBs = append(e.responseCBs, cb)
	e.cbMu.Unlock()
}

// OnError registers a callback invoked when a request fails.
func (e *Engine) OnError(cb ErrorCallback) {
	e.cbMu.Lock()
	e.errorCBs = append(e.errorCBs, cb)
	e.cbMu.Unlock()
}

// OnHTML registers a callback for HTML responses matching the given CSS
// selector. Selector matching will be implemented with the extraction engine;
// currently all HTML callbacks are invoked for every HTML response.
func (e *Engine) OnHTML(selector string, cb HTMLCallback) {
	e.cbMu.Lock()
	e.htmlCBs[selector] = append(e.htmlCBs[selector], cb)
	e.cbMu.Unlock()
}

// Stats returns a snapshot of the current crawler statistics.
func (e *Engine) Stats() *CrawlStats {
	var dur time.Duration
	if !e.startTime.IsZero() {
		dur = time.Since(e.startTime)
	}
	return &CrawlStats{
		RequestCount:    e.stats.requestCount.Load(),
		SuccessCount:    e.stats.successCount.Load(),
		ErrorCount:      e.stats.errorCount.Load(),
		BytesDownloaded: e.stats.bytesDownloaded.Load(),
		Duration:        dur,
	}
}

// worker is the main loop for a single worker goroutine.
func (e *Engine) worker() {
	defer e.workerWg.Done()
	for {
		select {
		case <-e.ctx.Done():
			return
		case req, ok := <-e.urlQueue:
			if !ok {
				return
			}
			e.processRequest(req)
			e.urlWg.Done()
		}
	}
}

// processRequest handles a single crawl request: acquires concurrency,
// makes the HTTP call, and invokes registered callbacks.
func (e *Engine) processRequest(req *CrawlRequest) {
	domain := extractDomain(req.URL)

	if err := e.concurrency.Acquire(e.ctx, domain); err != nil {
		e.stats.errorCount.Add(1)
		e.fireErrorCallbacks(req, err)
		return
	}
	defer e.concurrency.Release(domain)

	e.stats.requestCount.Add(1)
	e.fireRequestCallbacks(req)

	method := req.Method
	if method == "" {
		method = "GET"
	}

	httpResp, err := e.httpClient.Do(e.ctx, &client.Request{
		URL:     req.URL,
		Method:  method,
		Headers: req.Headers,
	})
	if err != nil {
		e.stats.errorCount.Add(1)
		e.fireErrorCallbacks(req, err)
		return
	}

	e.stats.successCount.Add(1)
	e.stats.bytesDownloaded.Add(int64(len(httpResp.Body)))

	resp := &CrawlResponse{
		Request:     req,
		StatusCode:  httpResp.StatusCode,
		Headers:     httpResp.Headers,
		Body:        httpResp.Body,
		ContentType: httpResp.ContentType,
	}

	e.fireResponseCallbacks(resp)

	if isHTMLResponse(resp.ContentType) {
		e.fireHTMLCallbacks(resp)
	}
}

func (e *Engine) fireRequestCallbacks(req *CrawlRequest) {
	e.cbMu.RLock()
	defer e.cbMu.RUnlock()
	for _, cb := range e.requestCBs {
		cb(req)
	}
}

func (e *Engine) fireResponseCallbacks(resp *CrawlResponse) {
	e.cbMu.RLock()
	defer e.cbMu.RUnlock()
	for _, cb := range e.responseCBs {
		cb(resp)
	}
}

func (e *Engine) fireErrorCallbacks(req *CrawlRequest, err error) {
	e.cbMu.RLock()
	defer e.cbMu.RUnlock()
	for _, cb := range e.errorCBs {
		cb(req, err)
	}
}

func (e *Engine) fireHTMLCallbacks(resp *CrawlResponse) {
	e.cbMu.RLock()
	defer e.cbMu.RUnlock()
	for _, cbs := range e.htmlCBs {
		for _, cb := range cbs {
			cb(resp)
		}
	}
}

// extractDomain returns the hostname from a URL string.
func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}

// isHTMLResponse checks if the content type indicates an HTML response.
func isHTMLResponse(contentType string) bool {
	return strings.Contains(contentType, "text/html")
}
