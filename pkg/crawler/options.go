package crawler

// RequestOption configures individual URL requests added to the crawler.
type RequestOption func(*CrawlRequest)

// WithMethod sets the HTTP method for a request.
func WithMethod(method string) RequestOption {
	return func(r *CrawlRequest) { r.Method = method }
}

// WithHeaders sets the HTTP headers for a request.
func WithHeaders(headers map[string]string) RequestOption {
	return func(r *CrawlRequest) { r.Headers = headers }
}

// WithMeta adds a metadata key-value pair to a request.
func WithMeta(key, value string) RequestOption {
	return func(r *CrawlRequest) {
		if r.Meta == nil {
			r.Meta = make(map[string]string)
		}
		r.Meta[key] = value
	}
}

// WithDepth sets the crawl depth for a request.
func WithDepth(depth int) RequestOption {
	return func(r *CrawlRequest) { r.Depth = depth }
}

// WithBrowserRender marks a request for JavaScript rendering via the browser
// pool. The engine must be configured with a Renderer for this to take effect.
func WithBrowserRender() RequestOption {
	return func(r *CrawlRequest) { r.BrowserRender = true }
}
