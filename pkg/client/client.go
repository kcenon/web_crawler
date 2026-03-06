package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Request represents an HTTP request to be executed by the client.
type Request struct {
	URL      string
	Method   string
	Headers  map[string]string
	Body     io.Reader
	Timeout  time.Duration
	Metadata map[string]string
}

// Response represents the result of an HTTP request execution.
type Response struct {
	StatusCode  int
	Headers     http.Header
	Body        []byte
	ContentType string
	FetchTime   time.Duration
	FinalURL    string
}

// HTTPClient defines the interface for executing HTTP requests.
type HTTPClient interface {
	Do(ctx context.Context, req *Request) (*Response, error)
	Close() error
}

// Config holds the HTTP client configuration.
type Config struct {
	// Timeout specifies the overall request timeout. Default: 30s.
	Timeout time.Duration

	// MaxRedirects controls the maximum number of redirects to follow.
	// Default: 10.
	MaxRedirects int

	// UserAgent sets the default User-Agent header.
	// Default: "web_crawler/0.1".
	UserAgent string

	// Transport configures the underlying HTTP transport.
	Transport TransportConfig

	// CookieJar, if non-nil, enables cookie management. Cookies from
	// Set-Cookie response headers are stored and automatically sent
	// on subsequent requests matching the cookie's domain and path.
	CookieJar http.CookieJar
}

func (c Config) withDefaults() Config {
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRedirects == 0 {
		c.MaxRedirects = 10
	}
	if c.UserAgent == "" {
		c.UserAgent = "web_crawler/0.1"
	}
	c.Transport = c.Transport.withDefaults()
	return c
}

// Option configures optional client parameters.
type Option func(*Client)

// Client implements HTTPClient using Go's net/http package.
type Client struct {
	httpClient *http.Client
	cfg        Config
	pool       *Pool
}

// New creates a new Client with the given configuration and options.
func New(cfg Config, opts ...Option) (*Client, error) {
	cfg = cfg.withDefaults()

	pool, err := newPool(cfg.Transport)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	c := &Client{
		cfg:  cfg,
		pool: pool,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.httpClient = &http.Client{
		Transport: c.pool.transport,
		Timeout:   cfg.Timeout,
		Jar:       cfg.CookieJar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= cfg.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", cfg.MaxRedirects)
			}
			return nil
		},
	}

	return c, nil
}

// Do executes an HTTP request and returns the response.
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be nil")
	}

	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", c.cfg.UserAgent)
	}

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
		httpReq = httpReq.WithContext(ctx)
	}

	start := time.Now()
	c.pool.recordRequest()

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return &Response{
		StatusCode:  httpResp.StatusCode,
		Headers:     httpResp.Header,
		Body:        body,
		ContentType: httpResp.Header.Get("Content-Type"),
		FetchTime:   time.Since(start),
		FinalURL:    httpResp.Request.URL.String(),
	}, nil
}

// Stats returns current connection pool statistics.
func (c *Client) Stats() PoolStats {
	return c.pool.stats()
}

// Close releases all resources held by the client.
func (c *Client) Close() error {
	c.pool.close()
	return nil
}
