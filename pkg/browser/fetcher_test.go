package browser_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kcenon/web_crawler/pkg/browser"
	"github.com/kcenon/web_crawler/pkg/client"
	"github.com/kcenon/web_crawler/pkg/crawler"
)

// newFetcherPool creates a minimal browser pool for fetcher tests.
// Tests that call this will launch real Chrome processes and are skipped if Chrome
// is unavailable or the test is run in short mode.
func newFetcherPool(t *testing.T) *browser.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping browser fetcher test in short mode (requires Chrome)")
	}
	pool, err := browser.NewBuilder().
		WithMaxBrowsers(1).
		WithMaxTabsPerBrowser(10).
		Build()
	if err != nil {
		t.Skipf("skipping browser test: cannot create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// spaServer returns a test server that serves a minimal SPA page. The initial
// HTML has no content; a <script> tag immediately renders the text "JS rendered"
// into #app so tests can verify that JS execution actually ran.
func spaServer() *httptest.Server {
	html := `<!DOCTYPE html>
<html><head><title>SPA</title></head>
<body>
  <div id="app"></div>
  <script>
    document.getElementById('app').textContent = 'JS rendered';
  </script>
</body></html>`
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html)
	}))
}

// TestRenderFetcher_ImplementsHTTPClient asserts the interface is satisfied at
// compile time.
func TestRenderFetcher_ImplementsHTTPClient(t *testing.T) {
	pool := newFetcherPool(t)
	var _ client.HTTPClient = browser.NewRenderFetcher(pool)
}

// TestRenderFetcher_NilRequest verifies that a nil request returns an error.
func TestRenderFetcher_NilRequest(t *testing.T) {
	pool := newFetcherPool(t)
	f := browser.NewRenderFetcher(pool)
	_, err := f.Do(context.Background(), nil)
	if err == nil {
		t.Fatal("Do(nil) should return error")
	}
}

// TestRenderFetcher_NonGETMethod verifies that non-GET methods are rejected.
func TestRenderFetcher_NonGETMethod(t *testing.T) {
	pool := newFetcherPool(t)
	f := browser.NewRenderFetcher(pool)
	_, err := f.Do(context.Background(), &client.Request{
		URL:    "http://example.com",
		Method: "POST",
	})
	if err == nil {
		t.Fatal("Do with POST should return error")
	}
}

// TestRenderFetcher_FetchesHTML verifies that the fetcher returns rendered HTML.
func TestRenderFetcher_FetchesHTML(t *testing.T) {
	pool := newFetcherPool(t)
	ts := spaServer()
	defer ts.Close()

	f := browser.NewRenderFetcher(pool)
	resp, err := f.Do(context.Background(), &client.Request{URL: ts.URL})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(resp.ContentType, "text/html") {
		t.Errorf("ContentType = %q, want text/html", resp.ContentType)
	}
	if len(resp.Body) == 0 {
		t.Error("Body should not be empty")
	}
}

// TestRenderFetcher_JSRenderedContent verifies that JavaScript is executed and
// the resulting DOM content is returned (the SPA test server renders "JS rendered").
func TestRenderFetcher_JSRenderedContent(t *testing.T) {
	pool := newFetcherPool(t)
	ts := spaServer()
	defer ts.Close()

	f := browser.NewRenderFetcher(pool)
	resp, err := f.Do(context.Background(), &client.Request{URL: ts.URL})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	html := string(resp.Body)
	if !strings.Contains(html, "JS rendered") {
		t.Errorf("expected JS-rendered content in HTML, got:\n%s", html)
	}
}

// TestRenderFetcher_ContextTimeout verifies that a context deadline is respected.
func TestRenderFetcher_ContextTimeout(t *testing.T) {
	pool := newFetcherPool(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	f := browser.NewRenderFetcher(pool)
	_, err := f.Do(ctx, &client.Request{URL: "http://192.0.2.1/"})
	if err == nil {
		t.Fatal("Do() with expired context should return error")
	}
}

// TestRenderFetcher_Close verifies that Close is a no-op and returns nil.
func TestRenderFetcher_Close(t *testing.T) {
	pool := newFetcherPool(t)
	f := browser.NewRenderFetcher(pool)
	if err := f.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestEngine_WithBrowserRender is an integration test that crawls a local SPA
// using the full crawler engine with WithBrowserRender() and verifies that JS
// content is extracted via the HTML callback.
func TestEngine_WithBrowserRender(t *testing.T) {
	pool := newFetcherPool(t)
	ts := spaServer()
	defer ts.Close()

	c, err := crawler.NewBuilder().
		WithWorkerCount(2).
		WithBrowserPool(pool).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var gotHTML atomic.Value

	c.OnHTML("body", func(resp *crawler.CrawlResponse) {
		gotHTML.Store(string(resp.Body))
	})

	if err := c.AddURL(ts.URL, crawler.WithBrowserRender()); err != nil {
		t.Fatalf("AddURL() error = %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	html, ok := gotHTML.Load().(string)
	if !ok || html == "" {
		t.Fatal("HTML callback was not called or returned empty body")
	}
	if !strings.Contains(html, "JS rendered") {
		t.Errorf("expected JS-rendered content, got:\n%s", html)
	}
}

// TestEngine_WithBrowserRender_FallbackToHTTP verifies that URLs without
// WithBrowserRender() still use the plain HTTP client even when a pool is
// configured.
func TestEngine_WithBrowserRender_FallbackToHTTP(t *testing.T) {
	pool := newFetcherPool(t)

	// The plain HTTP server returns different content than the SPA.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>http only</body></html>")
	}))
	defer ts.Close()

	c, err := crawler.NewBuilder().
		WithWorkerCount(1).
		WithBrowserPool(pool).
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var body atomic.Value
	c.OnResponse(func(resp *crawler.CrawlResponse) {
		body.Store(string(resp.Body))
	})

	// No WithBrowserRender() — should use plain HTTP client.
	if err := c.AddURL(ts.URL); err != nil {
		t.Fatalf("AddURL() error = %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	got, ok := body.Load().(string)
	if !ok || !strings.Contains(got, "http only") {
		t.Errorf("expected plain HTTP response, got: %s", got)
	}
}

// BenchmarkRenderFetcher_Pool measures the throughput of the browser pool when
// 10 goroutines issue concurrent render requests against a local SPA server.
func BenchmarkRenderFetcher_Pool(b *testing.B) {
	pool, err := browser.NewBuilder().
		WithMaxBrowsers(3).
		WithMaxTabsPerBrowser(20).
		Build()
	if err != nil {
		b.Skipf("skipping browser benchmark: %v", err)
	}
	defer pool.Close()

	ts := spaServer()
	defer ts.Close()

	f := browser.NewRenderFetcher(pool)
	req := &client.Request{URL: ts.URL}

	b.SetParallelism(10)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := f.Do(context.Background(), req)
			if err != nil {
				b.Errorf("Do() error = %v", err)
				return
			}
			_ = resp
		}
	})
}
