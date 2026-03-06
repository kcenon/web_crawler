package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	if c.cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", c.cfg.Timeout)
	}
	if c.cfg.MaxRedirects != 10 {
		t.Errorf("MaxRedirects = %d, want 10", c.cfg.MaxRedirects)
	}
	if c.cfg.UserAgent != "web_crawler/0.1" {
		t.Errorf("UserAgent = %q, want %q", c.cfg.UserAgent, "web_crawler/0.1")
	}
}

func TestNew_CustomConfig(t *testing.T) {
	c, err := New(Config{
		Timeout:      5 * time.Second,
		MaxRedirects: 3,
		UserAgent:    "custom/1.0",
		Transport: TransportConfig{
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 5,
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	if c.cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", c.cfg.Timeout)
	}
	if c.cfg.MaxRedirects != 3 {
		t.Errorf("MaxRedirects = %d, want 3", c.cfg.MaxRedirects)
	}
	if c.cfg.Transport.MaxIdleConns != 50 {
		t.Errorf("MaxIdleConns = %d, want 50", c.cfg.Transport.MaxIdleConns)
	}
}

func TestDo_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<html>hello</html>")
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	resp, err := c.Do(context.Background(), &Request{URL: ts.URL})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if resp.ContentType != "text/html" {
		t.Errorf("ContentType = %q, want %q", resp.ContentType, "text/html")
	}
	if string(resp.Body) != "<html>hello</html>" {
		t.Errorf("Body = %q, want %q", string(resp.Body), "<html>hello</html>")
	}
	if resp.FinalURL != ts.URL {
		t.Errorf("FinalURL = %q, want %q", resp.FinalURL, ts.URL)
	}
	if resp.FetchTime <= 0 {
		t.Error("FetchTime should be positive")
	}
}

func TestDo_POST(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	resp, err := c.Do(context.Background(), &Request{
		URL:    ts.URL,
		Method: http.MethodPost,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: strings.NewReader(`{"key":"value"}`),
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
}

func TestDo_CustomHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Custom"); got != "test-value" {
			t.Errorf("X-Custom = %q, want %q", got, "test-value")
		}
		if got := r.Header.Get("User-Agent"); got != "custom-agent" {
			t.Errorf("User-Agent = %q, want %q", got, "custom-agent")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	_, err = c.Do(context.Background(), &Request{
		URL: ts.URL,
		Headers: map[string]string{
			"X-Custom":   "test-value",
			"User-Agent": "custom-agent",
		},
	})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestDo_DefaultUserAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "web_crawler/0.1" {
			t.Errorf("User-Agent = %q, want %q", got, "web_crawler/0.1")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	_, err = c.Do(context.Background(), &Request{URL: ts.URL})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestDo_DefaultMethod(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	_, err = c.Do(context.Background(), &Request{URL: ts.URL})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
}

func TestDo_NilRequest(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	_, err = c.Do(context.Background(), nil)
	if err == nil {
		t.Fatal("Do(nil) should return error")
	}
}

func TestDo_InvalidURL(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	_, err = c.Do(context.Background(), &Request{URL: "://invalid"})
	if err == nil {
		t.Fatal("Do() with invalid URL should return error")
	}
}

func TestDo_ContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err = c.Do(ctx, &Request{URL: ts.URL})
	if err == nil {
		t.Fatal("Do() with cancelled context should return error")
	}
}

func TestDo_PerRequestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := New(Config{Timeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	_, err = c.Do(context.Background(), &Request{
		URL:     ts.URL,
		Timeout: 50 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("Do() should timeout with short per-request timeout")
	}
}

func TestDo_Redirect(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/end", http.StatusFound)
			return
		}
		fmt.Fprint(w, "final")
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	resp, err := c.Do(context.Background(), &Request{URL: ts.URL + "/start"})
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if string(resp.Body) != "final" {
		t.Errorf("Body = %q, want %q", string(resp.Body), "final")
	}
	if resp.FinalURL != ts.URL+"/end" {
		t.Errorf("FinalURL = %q, want %q", resp.FinalURL, ts.URL+"/end")
	}
}

func TestDo_MaxRedirectsExceeded(t *testing.T) {
	redirectCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		http.Redirect(w, r, fmt.Sprintf("/r%d", redirectCount), http.StatusFound)
	}))
	defer ts.Close()

	c, err := New(Config{MaxRedirects: 2})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	_, err = c.Do(context.Background(), &Request{URL: ts.URL})
	if err == nil {
		t.Fatal("Do() should fail when max redirects exceeded")
	}
}

func TestPoolStats(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	if got := c.Stats().TotalRequests; got != 0 {
		t.Errorf("initial TotalRequests = %d, want 0", got)
	}

	for i := 0; i < 3; i++ {
		_, err := c.Do(context.Background(), &Request{URL: ts.URL})
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
	}

	if got := c.Stats().TotalRequests; got != 3 {
		t.Errorf("TotalRequests = %d, want 3", got)
	}
}

func TestTransportConfig_Defaults(t *testing.T) {
	cfg := TransportConfig{}.withDefaults()

	if cfg.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", cfg.MaxIdleConns)
	}
	if cfg.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 10", cfg.MaxIdleConnsPerHost)
	}
	if cfg.DialTimeout != 10*time.Second {
		t.Errorf("DialTimeout = %v, want 10s", cfg.DialTimeout)
	}
	if cfg.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("TLSHandshakeTimeout = %v, want 10s", cfg.TLSHandshakeTimeout)
	}
	if cfg.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", cfg.IdleConnTimeout)
	}
}

func TestTransportConfig_HTTP2Toggle(t *testing.T) {
	// HTTP/2 enabled by default (DisableHTTP2 = false)
	transport := buildTransport(TransportConfig{}.withDefaults())
	if !transport.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be true by default")
	}

	// HTTP/2 disabled explicitly
	transport = buildTransport(TransportConfig{DisableHTTP2: true}.withDefaults())
	if transport.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be false when DisableHTTP2 is set")
	}
}

func TestTransportConfig_TLS12Minimum(t *testing.T) {
	transport := buildTransport(TransportConfig{}.withDefaults())
	if transport.TLSClientConfig.MinVersion != 0x0303 { // tls.VersionTLS12
		t.Errorf("TLS MinVersion = %#x, want TLS 1.2 (%#x)", transport.TLSClientConfig.MinVersion, 0x0303)
	}
}

func TestClose(t *testing.T) {
	c, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
