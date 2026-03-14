package browser

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/kcenon/web_crawler/pkg/client"
)

// RenderFetcher implements client.HTTPClient using a browser pool for
// JavaScript rendering. It satisfies the same Do/Close interface as the
// standard HTTP client so the crawler engine can swap them transparently.
type RenderFetcher struct {
	renderer *Renderer
}

// NewRenderFetcher creates a RenderFetcher backed by the given pool.
func NewRenderFetcher(pool *Pool) *RenderFetcher {
	return &RenderFetcher{renderer: NewRenderer(pool)}
}

// Do renders the URL in a pooled browser tab and returns the page HTML as
// a client.Response. Only GET semantics are supported; other methods return
// an error. The response body is the full outer HTML after JS execution.
func (f *RenderFetcher) Do(ctx context.Context, req *client.Request) (*client.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("render fetcher: request must not be nil")
	}
	if req.Method != "" && req.Method != "GET" {
		return nil, fmt.Errorf("render fetcher: only GET is supported, got %s", req.Method)
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	start := time.Now()
	result, err := f.renderer.Render(ctx, RenderRequest{
		URL:     req.URL,
		Timeout: timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("render fetcher: %w", err)
	}

	headers := make(http.Header)
	headers.Set("Content-Type", "text/html; charset=utf-8")

	return &client.Response{
		StatusCode:  200,
		Headers:     headers,
		Body:        []byte(result.HTML),
		ContentType: "text/html; charset=utf-8",
		FetchTime:   time.Since(start),
		FinalURL:    result.URL,
	}, nil
}

// Close is a no-op. The underlying pool is managed by the caller.
func (f *RenderFetcher) Close() error {
	return nil
}
