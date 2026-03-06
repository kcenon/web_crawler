package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func mustAddURL(t *testing.T, c Crawler, url string, opts ...RequestOption) {
	t.Helper()
	if err := c.AddURL(url, opts...); err != nil {
		t.Fatalf("AddURL(%q) error = %v", url, err)
	}
}

func mustStart(t *testing.T, c Crawler, ctx context.Context) {
	t.Helper()
	if err := c.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func mustWait(t *testing.T, c Crawler) {
	t.Helper()
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
}

func newTestServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func newTestEngine(t *testing.T) (Crawler, func()) {
	t.Helper()
	c, err := NewBuilder().WithWorkerCount(2).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	return c, func() {}
}

func TestBuilder_Defaults(t *testing.T) {
	c, err := NewBuilder().Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	eng := c.(*Engine)
	if eng.cfg.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3", eng.cfg.MaxDepth)
	}
	if eng.cfg.WorkerCount != 10 {
		t.Errorf("WorkerCount = %d, want 10", eng.cfg.WorkerCount)
	}
	if eng.cfg.UserAgent != "web_crawler/0.1" {
		t.Errorf("UserAgent = %q, want %q", eng.cfg.UserAgent, "web_crawler/0.1")
	}
}

func TestBuilder_CustomConfig(t *testing.T) {
	c, err := NewBuilder().
		WithMaxDepth(5).
		WithMaxPages(100).
		WithWorkerCount(4).
		WithUserAgent("test/1.0").
		Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	eng := c.(*Engine)
	if eng.cfg.MaxDepth != 5 {
		t.Errorf("MaxDepth = %d, want 5", eng.cfg.MaxDepth)
	}
	if eng.cfg.MaxPages != 100 {
		t.Errorf("MaxPages = %d, want 100", eng.cfg.MaxPages)
	}
	if eng.cfg.WorkerCount != 4 {
		t.Errorf("WorkerCount = %d, want 4", eng.cfg.WorkerCount)
	}
}

func TestBuilder_InvalidConfig(t *testing.T) {
	_, err := NewBuilder().WithWorkerCount(-1).Build()
	if err == nil {
		t.Fatal("Build() with negative workers should return error")
	}
}

func TestEngine_BasicCrawl(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html>hello</html>")
	})
	defer ts.Close()

	c, _ := newTestEngine(t)
	var gotResponse atomic.Bool

	c.OnResponse(func(resp *CrawlResponse) {
		gotResponse.Store(true)
		if resp.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
		}
		if string(resp.Body) != "<html>hello</html>" {
			t.Errorf("Body = %q, want %q", string(resp.Body), "<html>hello</html>")
		}
	})

	if err := c.AddURL(ts.URL); err != nil {
		t.Fatalf("AddURL() error = %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if !gotResponse.Load() {
		t.Error("response callback was not called")
	}

	stats := c.Stats()
	if stats.RequestCount != 1 {
		t.Errorf("RequestCount = %d, want 1", stats.RequestCount)
	}
	if stats.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", stats.SuccessCount)
	}
}

func TestEngine_MultipleURLs(t *testing.T) {
	var requestCount atomic.Int32
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "ok")
	})
	defer ts.Close()

	c, _ := newTestEngine(t)

	urls := []string{ts.URL + "/a", ts.URL + "/b", ts.URL + "/c"}
	if err := c.AddURLs(urls); err != nil {
		t.Fatalf("AddURLs() error = %v", err)
	}
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if got := requestCount.Load(); got != 3 {
		t.Errorf("server received %d requests, want 3", got)
	}

	stats := c.Stats()
	if stats.RequestCount != 3 {
		t.Errorf("RequestCount = %d, want 3", stats.RequestCount)
	}
	if stats.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3", stats.SuccessCount)
	}
}

func TestEngine_OnRequestCallback(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	c, _ := newTestEngine(t)
	var requestURL atomic.Value

	c.OnRequest(func(req *CrawlRequest) {
		requestURL.Store(req.URL)
	})

	mustAddURL(t, c, ts.URL)
	mustStart(t, c, context.Background())
	mustWait(t, c)

	if got, _ := requestURL.Load().(string); got != ts.URL {
		t.Errorf("OnRequest URL = %q, want %q", got, ts.URL)
	}
}

func TestEngine_OnErrorCallback(t *testing.T) {
	c, _ := newTestEngine(t)
	var gotError atomic.Bool

	c.OnError(func(req *CrawlRequest, err error) {
		gotError.Store(true)
	})

	// Use an unreachable URL to trigger an error.
	mustAddURL(t, c, "http://127.0.0.1:1/unreachable")
	mustStart(t, c, context.Background())
	mustWait(t, c)

	if !gotError.Load() {
		t.Error("error callback was not called")
	}

	stats := c.Stats()
	if stats.ErrorCount == 0 {
		t.Error("ErrorCount should be > 0")
	}
}

func TestEngine_OnHTMLCallback(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<html><body><h1>Title</h1></body></html>")
	})
	defer ts.Close()

	c, _ := newTestEngine(t)
	var gotHTML atomic.Bool

	c.OnHTML("h1", func(resp *CrawlResponse) {
		gotHTML.Store(true)
	})

	mustAddURL(t, c, ts.URL)
	mustStart(t, c, context.Background())
	mustWait(t, c)

	if !gotHTML.Load() {
		t.Error("HTML callback was not called for HTML response")
	}
}

func TestEngine_Stop(t *testing.T) {
	// Server blocks until unblocked via channel.
	unblock := make(chan struct{})
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
		w.WriteHeader(http.StatusOK)
	})
	defer func() {
		close(unblock)
		ts.Close()
	}()

	c, _ := newTestEngine(t)
	mustAddURL(t, c, ts.URL)
	mustStart(t, c, context.Background())

	time.Sleep(50 * time.Millisecond)
	if err := c.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if err := c.Wait(); err == nil {
		t.Log("Wait returned nil after Stop (ok if all URLs processed)")
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	unblock := make(chan struct{})
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
		w.WriteHeader(http.StatusOK)
	})
	defer func() {
		close(unblock)
		ts.Close()
	}()

	c, _ := newTestEngine(t)
	ctx, cancel := context.WithCancel(context.Background())

	mustAddURL(t, c, ts.URL)
	mustStart(t, c, ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	err := c.Wait()
	if err == nil {
		t.Log("Wait returned nil after context cancel (ok if all URLs processed)")
	}
}

func TestEngine_DoubleStart(t *testing.T) {
	c, _ := newTestEngine(t)
	if err := c.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := c.Start(context.Background()); err == nil {
		t.Fatal("second Start() should return error")
	}
	_ = c.Stop(context.Background())
	_ = c.Wait()
}

func TestEngine_WaitBeforeStart(t *testing.T) {
	c, _ := newTestEngine(t)
	if err := c.Wait(); err == nil {
		t.Fatal("Wait() before Start() should return error")
	}
}

func TestEngine_AddURLAfterStop(t *testing.T) {
	c, _ := newTestEngine(t)
	mustStart(t, c, context.Background())
	_ = c.Stop(context.Background())

	// Allow stop to take effect.
	time.Sleep(10 * time.Millisecond)

	if err := c.AddURL("http://example.com"); err == nil {
		t.Fatal("AddURL after Stop should return error")
	}
}

func TestEngine_StatsAccuracy(t *testing.T) {
	var bodySize int
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		body := "hello world"
		bodySize = len(body)
		fmt.Fprint(w, body)
	})
	defer ts.Close()

	c, _ := newTestEngine(t)
	mustAddURL(t, c, ts.URL)
	mustStart(t, c, context.Background())
	mustWait(t, c)

	stats := c.Stats()
	if stats.BytesDownloaded != int64(bodySize) {
		t.Errorf("BytesDownloaded = %d, want %d", stats.BytesDownloaded, bodySize)
	}
	if stats.Duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestEngine_ConcurrentCallbacks(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html>ok</html>")
	})
	defer ts.Close()

	c, err := NewBuilder().WithWorkerCount(4).Build()
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var mu sync.Mutex
	var responses []string

	c.OnResponse(func(resp *CrawlResponse) {
		mu.Lock()
		responses = append(responses, resp.Request.URL)
		mu.Unlock()
	})

	for i := 0; i < 10; i++ {
		mustAddURL(t, c, fmt.Sprintf("%s/%d", ts.URL, i))
	}
	mustStart(t, c, context.Background())
	mustWait(t, c)

	if len(responses) != 10 {
		t.Errorf("got %d responses, want 10", len(responses))
	}
}

func TestEngine_RequestOptions(t *testing.T) {
	ts := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("X-Custom = %q, want %q", r.Header.Get("X-Custom"), "value")
		}
		w.WriteHeader(http.StatusOK)
	})
	defer ts.Close()

	c, _ := newTestEngine(t)
	mustAddURL(t, c, ts.URL,
		WithMethod(http.MethodPost),
		WithHeaders(map[string]string{"X-Custom": "value"}),
		WithMeta("key", "val"),
		WithDepth(2),
	)
	mustStart(t, c, context.Background())
	mustWait(t, c)
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/path", "example.com"},
		{"http://sub.example.com:8080/path", "sub.example.com"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		if got := extractDomain(tt.url); got != tt.want {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
