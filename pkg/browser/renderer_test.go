package browser_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kcenon/web_crawler/pkg/browser"
)

// testSPAServer returns a test HTTP server that serves an HTML page with
// JavaScript that updates the DOM after a short delay. It mimics a SPA.
func testSPAServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>SPA Test</title></head>
<body>
  <div id="app">loading...</div>
  <script>
    setTimeout(function() {
      document.getElementById('app').textContent = 'content-loaded';
    }, 100);
  </script>
</body>
</html>`))
	})
	mux.HandleFunc("/static", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html><body><p id="text">Hello, world!</p></body></html>`))
	})
	return httptest.NewServer(mux)
}

// newRendererForTest creates a Renderer backed by a single-browser pool.
func newRendererForTest(t *testing.T) *browser.Renderer {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping renderer test in short mode (requires Chrome)")
	}
	p, err := browser.NewPool(browser.Config{
		MaxBrowsers:         1,
		MaxTabsPerBrowser:   20,
		Headless:            true,
		HealthCheckInterval: 24 * time.Hour,
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	t.Cleanup(p.Close)
	return browser.NewRenderer(p)
}

func TestRenderer_BasicRender(t *testing.T) {
	srv := testSPAServer(t)
	defer srv.Close()

	r := newRendererForTest(t)
	res, err := r.Render(context.Background(), browser.RenderRequest{
		URL:  srv.URL + "/static",
		Wait: browser.WaitDOMContentLoaded{},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(res.HTML, "Hello, world!") {
		t.Errorf("expected HTML to contain 'Hello, world!', got: %s", res.HTML[:min(200, len(res.HTML))])
	}
	if res.URL == "" {
		t.Error("expected non-empty URL in result")
	}
}

func TestRenderer_WaitSelector(t *testing.T) {
	srv := testSPAServer(t)
	defer srv.Close()

	r := newRendererForTest(t)
	res, err := r.Render(context.Background(), browser.RenderRequest{
		URL:  srv.URL + "/",
		Wait: browser.WaitSelector{Selector: "#app"},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(res.HTML, "app") {
		t.Errorf("expected #app element in HTML, got: %s", res.HTML[:min(300, len(res.HTML))])
	}
}

func TestRenderer_WaitDelay(t *testing.T) {
	srv := testSPAServer(t)
	defer srv.Close()

	r := newRendererForTest(t)
	start := time.Now()
	_, err := r.Render(context.Background(), browser.RenderRequest{
		URL:  srv.URL + "/static",
		Wait: browser.WaitDelay{Duration: 200 * time.Millisecond},
	})
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("expected at least 200ms wait, got %v", elapsed)
	}
}

func TestRenderer_WaitNetworkIdle(t *testing.T) {
	srv := testSPAServer(t)
	defer srv.Close()

	r := newRendererForTest(t)
	res, err := r.Render(context.Background(), browser.RenderRequest{
		URL: srv.URL + "/static",
		Wait: browser.WaitNetworkIdle{
			QuietPeriod: 200 * time.Millisecond,
			Timeout:     5 * time.Second,
		},
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if res.HTML == "" {
		t.Error("expected non-empty HTML")
	}
}

func TestRenderer_Screenshot(t *testing.T) {
	srv := testSPAServer(t)
	defer srv.Close()

	r := newRendererForTest(t)
	res, err := r.Render(context.Background(), browser.RenderRequest{
		URL:        srv.URL + "/static",
		Screenshot: true,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(res.Screenshot) == 0 {
		t.Error("expected non-empty screenshot bytes")
	}
	// PNG magic bytes: 0x89 0x50 0x4E 0x47
	if len(res.Screenshot) < 4 || res.Screenshot[0] != 0x89 || res.Screenshot[1] != 0x50 {
		t.Error("screenshot does not look like a PNG")
	}
}

func TestRenderer_ResourceFilter(t *testing.T) {
	srv := testSPAServer(t)
	defer srv.Close()

	r := newRendererForTest(t)
	// Blocking all resources should still succeed for the HTML document itself.
	res, err := r.Render(context.Background(), browser.RenderRequest{
		URL:       srv.URL + "/static",
		Resources: browser.BlockAll(),
	})
	if err != nil {
		t.Fatalf("Render with resource filter: %v", err)
	}
	if !strings.Contains(res.HTML, "Hello, world!") {
		t.Errorf("expected HTML content, got: %s", res.HTML[:min(200, len(res.HTML))])
	}
}

func TestRenderer_JavaScriptEval(t *testing.T) {
	srv := testSPAServer(t)
	defer srv.Close()

	r := newRendererForTest(t)
	res, err := r.Render(context.Background(), browser.RenderRequest{
		URL:        srv.URL + "/static",
		JavaScript: `document.getElementById('text').textContent`,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if res.ScriptResult == "" {
		t.Error("expected non-empty script result")
	}
}

func TestRenderer_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}
	// Create a local server that accepts the connection but never sends HTTP headers,
	// forcing Chrome to hang in page.Navigate indefinitely.
	hung := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not write headers; block until the connection is closed.
		<-r.Context().Done()
	}))
	// Force-close any connections Chrome holds before calling Close, so that
	// hung.Close() doesn't block waiting for Chrome's open connection.
	defer func() {
		hung.CloseClientConnections()
		hung.Close()
	}()

	p, err := browser.NewPool(browser.Config{
		MaxBrowsers:         1,
		Headless:            true,
		HealthCheckInterval: 24 * time.Hour,
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	defer p.Close()

	r := browser.NewRenderer(p)
	_, err = r.Render(context.Background(), browser.RenderRequest{
		URL:     hung.URL,
		Timeout: 2 * time.Second,
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestResourceFilter_BlockAll(t *testing.T) {
	f := browser.BlockAll()
	if !f.BlockImages || !f.BlockStylesheets || !f.BlockFonts || !f.BlockMedia {
		t.Error("BlockAll should set all block flags to true")
	}
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
