package middleware

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Default config ---

func TestCache_DefaultConfig(t *testing.T) {
	cfg := DefaultCacheConfig()
	if cfg.MaxEntries != 256 {
		t.Errorf("MaxEntries = %d, want 256", cfg.MaxEntries)
	}
	if cfg.DefaultTTL != 5*time.Minute {
		t.Errorf("DefaultTTL = %v, want 5m", cfg.DefaultTTL)
	}
	if len(cfg.CacheableMethods) == 0 {
		t.Error("CacheableMethods must not be empty")
	}
	if len(cfg.CacheableStatuses) == 0 {
		t.Error("CacheableStatuses must not be empty")
	}
}

func TestCache_DefaultsApplied(t *testing.T) {
	c := NewCache(CacheConfig{})
	if c.cfg.MaxEntries != 256 {
		t.Errorf("MaxEntries = %d, want 256", c.cfg.MaxEntries)
	}
	if c.cfg.DefaultTTL != 5*time.Minute {
		t.Errorf("DefaultTTL = %v, want 5m", c.cfg.DefaultTTL)
	}
}

// --- Cache miss / hit ---

func TestCache_Miss_CallsNext(t *testing.T) {
	var calls int
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls++
		return &Response{StatusCode: 200, Body: []byte("fresh")}, nil
	}

	c := NewCache(DefaultCacheConfig())
	req := &Request{URL: "http://example.com", Method: "GET"}

	if _, err := c.ProcessRequest(context.Background(), req, handler); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("handler called %d times, want 1", calls)
	}
}

func TestCache_Hit_DoesNotCallNext(t *testing.T) {
	var calls int
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls++
		return &Response{StatusCode: 200, Body: []byte("body")}, nil
	}

	c := NewCache(DefaultCacheConfig())
	req := func() *Request { return &Request{URL: "http://example.com", Method: "GET"} }

	// First request: miss.
	if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
		t.Fatal(err)
	}
	// Second request: hit.
	if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
		t.Fatal(err)
	}

	if calls != 1 {
		t.Errorf("handler called %d times after 2 requests, want 1 (second should be cached)", calls)
	}
}

func TestCache_Hit_ReturnsCachedBody(t *testing.T) {
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		return &Response{StatusCode: 200, Body: []byte("cached-body")}, nil
	}

	c := NewCache(DefaultCacheConfig())
	req := func() *Request { return &Request{URL: "http://example.com", Method: "GET"} }

	if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
		t.Fatal(err)
	}
	resp, err := c.ProcessRequest(context.Background(), req(), handler)
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Body) != "cached-body" {
		t.Errorf("body = %q, want %q", resp.Body, "cached-body")
	}
}

// --- Method filtering ---

func TestCache_NonCacheableMethod_BypassesCache(t *testing.T) {
	var calls int
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls++
		return &Response{StatusCode: 200}, nil
	}

	c := NewCache(DefaultCacheConfig()) // only GET is cacheable
	req := func() *Request { return &Request{URL: "http://example.com", Method: "POST"} }

	for range 3 {
		if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Errorf("POST handler called %d times, want 3 (no caching)", calls)
	}
}

func TestCache_DefaultMethod_TreatedAsGET(t *testing.T) {
	var calls int
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls++
		return &Response{StatusCode: 200}, nil
	}

	c := NewCache(DefaultCacheConfig())
	// Empty method defaults to GET.
	req := func() *Request { return &Request{URL: "http://example.com", Method: ""} }

	if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("handler called %d times, want 1 (empty method cached as GET)", calls)
	}
}

// --- Status filtering ---

func TestCache_NonCacheableStatus_NotStored(t *testing.T) {
	var calls int
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls++
		return &Response{StatusCode: 404}, nil
	}

	c := NewCache(DefaultCacheConfig()) // only 200 is cacheable
	req := func() *Request { return &Request{URL: "http://example.com", Method: "GET"} }

	for range 3 {
		if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Errorf("handler called %d times, want 3 (404 should not be cached)", calls)
	}
}

// --- Cache-Control ---

func TestCache_NoStore_NotStored(t *testing.T) {
	var calls int
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls++
		return &Response{
			StatusCode: 200,
			Headers:    map[string]string{"Cache-Control": "no-store"},
		}, nil
	}

	c := NewCache(DefaultCacheConfig())
	req := func() *Request { return &Request{URL: "http://example.com", Method: "GET"} }

	for range 3 {
		if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 3 {
		t.Errorf("handler called %d times, want 3 (no-store response must not be cached)", calls)
	}
	if c.Len() != 0 {
		t.Errorf("cache.Len() = %d, want 0 (no-store)", c.Len())
	}
}

func TestCache_MaxAge_OverridesDefaultTTL(t *testing.T) {
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		return &Response{
			StatusCode: 200,
			Headers:    map[string]string{"Cache-Control": "max-age=1"},
			Body:       []byte("data"),
		}, nil
	}

	c := NewCache(CacheConfig{
		MaxEntries: 10,
		DefaultTTL: 10 * time.Minute, // long default — max-age=1 should win
	})
	req := func() *Request { return &Request{URL: "http://example.com", Method: "GET"} }

	// Prime the cache.
	if _, err := c.ProcessRequest(context.Background(), req(), handler); err != nil {
		t.Fatal(err)
	}

	// Wait for max-age to expire.
	time.Sleep(1100 * time.Millisecond)

	// Should be a miss and call next again.
	var calls int
	expired := func(_ context.Context, _ *Request) (*Response, error) {
		calls++
		return &Response{StatusCode: 200, Body: []byte("fresh")}, nil
	}
	if _, err := c.ProcessRequest(context.Background(), req(), expired); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("handler called %d times after expiry, want 1", calls)
	}
}

// --- LRU eviction ---

func TestCache_LRU_EvictsWhenFull(t *testing.T) {
	c := NewCache(CacheConfig{
		MaxEntries:        2,
		DefaultTTL:        time.Hour,
		CacheableMethods:  []string{"GET"},
		CacheableStatuses: []int{200},
	})

	makeHandler := func(body string) NextFunc {
		return func(_ context.Context, _ *Request) (*Response, error) {
			return &Response{StatusCode: 200, Body: []byte(body)}, nil
		}
	}

	url := func(u string) *Request { return &Request{URL: u, Method: "GET"} }

	// Fill cache: A, B.
	if _, err := c.ProcessRequest(context.Background(), url("http://a.com"), makeHandler("A")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ProcessRequest(context.Background(), url("http://b.com"), makeHandler("B")); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Fatalf("Len = %d, want 2", c.Len())
	}

	// Access A to make it more recently used.
	if _, err := c.ProcessRequest(context.Background(), url("http://a.com"), makeHandler("A")); err != nil {
		t.Fatal(err)
	}

	// Add C: should evict B (LRU) not A (recently accessed).
	if _, err := c.ProcessRequest(context.Background(), url("http://c.com"), makeHandler("C")); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Fatalf("Len = %d after eviction, want 2", c.Len())
	}

	// A must still be in cache (was recently accessed).
	var callsA int
	respA, err := c.ProcessRequest(context.Background(), url("http://a.com"), func(_ context.Context, _ *Request) (*Response, error) {
		callsA++
		return &Response{StatusCode: 200}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if callsA != 0 {
		t.Error("A was evicted but should have stayed (was recently used)")
	}
	if string(respA.Body) != "A" {
		t.Errorf("A body = %q, want %q", respA.Body, "A")
	}
}

// --- Len ---

func TestCache_Len_TracksEntries(t *testing.T) {
	c := NewCache(CacheConfig{
		MaxEntries:        10,
		DefaultTTL:        time.Hour,
		CacheableMethods:  []string{"GET"},
		CacheableStatuses: []int{200},
	})
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		return &Response{StatusCode: 200}, nil
	}

	if c.Len() != 0 {
		t.Fatalf("Len = %d before any request, want 0", c.Len())
	}

	if _, err := c.ProcessRequest(context.Background(), &Request{URL: "http://a.com", Method: "GET"}, handler); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 1 {
		t.Errorf("Len = %d after 1 unique URL, want 1", c.Len())
	}

	if _, err := c.ProcessRequest(context.Background(), &Request{URL: "http://b.com", Method: "GET"}, handler); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Errorf("Len = %d after 2 unique URLs, want 2", c.Len())
	}

	// Same URL again: no new entry.
	if _, err := c.ProcessRequest(context.Background(), &Request{URL: "http://a.com", Method: "GET"}, handler); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 2 {
		t.Errorf("Len = %d after duplicate URL, want 2", c.Len())
	}
}

// --- Concurrency ---

func TestCache_Concurrent(t *testing.T) {
	var calls atomic.Int64
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls.Add(1)
		return &Response{StatusCode: 200, Body: []byte("ok")}, nil
	}

	c := NewCache(DefaultCacheConfig())

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			req := &Request{URL: "http://example.com", Method: "GET"}
			if _, err := c.ProcessRequest(context.Background(), req, handler); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	// Handler should be called very few times due to caching (ideally 1,
	// but a small number due to race during initial fill is acceptable).
	if n := calls.Load(); n > 5 {
		t.Errorf("handler called %d times in concurrent scenario, want ≤5", n)
	}
}
