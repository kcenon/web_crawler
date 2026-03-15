package middleware

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// --- Constructor ---

func TestProxyRotation_Panic_EmptyPool(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty proxy pool, got none")
		}
	}()
	NewProxyRotation(ProxyRotationConfig{}) // should panic
}

func TestProxyRotation_DefaultsMaxFailures(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies: []ProxyEntry{{URL: "http://proxy1:8080"}},
	})
	if pr.cfg.MaxFailures != 3 {
		t.Errorf("MaxFailures = %d, want 3", pr.cfg.MaxFailures)
	}
}

func TestProxyRotation_InvalidModeFallsBackToRoundRobin(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      []ProxyEntry{{URL: "http://proxy1:8080"}},
		RotationMode: "bogus-mode",
	})
	if pr.cfg.RotationMode != ProxyRotationRoundRobin {
		t.Errorf("RotationMode = %q, want %q", pr.cfg.RotationMode, ProxyRotationRoundRobin)
	}
}

func TestProxyRotation_NormalisesZeroWeight(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies: []ProxyEntry{{URL: "http://proxy1:8080", Weight: 0}},
	})
	if pr.cfg.Proxies[0].Weight != 1 {
		t.Errorf("Weight = %d, want 1 after normalisation", pr.cfg.Proxies[0].Weight)
	}
}

// --- Meta injection ---

func TestProxyRotation_SetsMetaProxy(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies: []ProxyEntry{{URL: "http://proxy1:8080"}},
	})
	req := &Request{URL: "http://example.com"}
	if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}
	if req.Meta[MetaKeyProxy] != "http://proxy1:8080" {
		t.Errorf("Meta[proxy] = %q, want %q", req.Meta[MetaKeyProxy], "http://proxy1:8080")
	}
}

func TestProxyRotation_SetsMetaProxy_WithCredentials(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies: []ProxyEntry{
			{URL: "socks5://proxy1:1080", Username: "usr", Password: "fixture-val"},
		},
	})
	req := &Request{URL: "http://example.com"}
	if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}
	got := req.Meta[MetaKeyProxy].(string)
	want := "socks5://usr:fixture-val@proxy1:1080"
	if got != want {
		t.Errorf("Meta[proxy] = %q, want %q", got, want)
	}
}

func TestProxyRotation_InitialisesNilMetaMap(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies: []ProxyEntry{{URL: "http://proxy1:8080"}},
	})
	req := &Request{URL: "http://example.com", Meta: nil}
	if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}
	if req.Meta == nil {
		t.Error("Meta map must be initialised by middleware")
	}
	if req.Meta[MetaKeyProxy] == nil {
		t.Error("Meta[proxy] must be set")
	}
}

// --- Round-robin ---

func TestProxyRotation_RoundRobin_Cycles(t *testing.T) {
	proxies := []ProxyEntry{
		{URL: "http://p1:8080"},
		{URL: "http://p2:8080"},
		{URL: "http://p3:8080"},
	}
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      proxies,
		RotationMode: ProxyRotationRoundRobin,
	})

	for i := 0; i < len(proxies)*3; i++ {
		req := &Request{URL: "http://example.com"}
		if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
			t.Fatal(err)
		}
		want := proxies[i%len(proxies)].URL
		if req.Meta[MetaKeyProxy] != want {
			t.Errorf("request %d: proxy = %q, want %q", i, req.Meta[MetaKeyProxy], want)
		}
	}
}

func TestProxyRotation_RoundRobin_SingleProxy(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies: []ProxyEntry{{URL: "http://only:8080"}},
	})
	for i := 0; i < 5; i++ {
		req := &Request{URL: "http://example.com"}
		if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
			t.Fatal(err)
		}
		if req.Meta[MetaKeyProxy] != "http://only:8080" {
			t.Errorf("proxy = %q, want %q", req.Meta[MetaKeyProxy], "http://only:8080")
		}
	}
}

// --- Random ---

func TestProxyRotation_Random_PicksFromPool(t *testing.T) {
	proxies := []ProxyEntry{
		{URL: "http://p1:8080"},
		{URL: "http://p2:8080"},
		{URL: "http://p3:8080"},
	}
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      proxies,
		RotationMode: ProxyRotationRandom,
	})
	pool := map[string]bool{
		"http://p1:8080": true,
		"http://p2:8080": true,
		"http://p3:8080": true,
	}
	for range 30 {
		req := &Request{URL: "http://example.com"}
		if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
			t.Fatal(err)
		}
		if !pool[req.Meta[MetaKeyProxy].(string)] {
			t.Errorf("random proxy %q not in pool", req.Meta[MetaKeyProxy])
		}
	}
}

// --- Weighted ---

func TestProxyRotation_Weighted_RespectsWeights(t *testing.T) {
	proxies := []ProxyEntry{
		{URL: "http://p1:8080", Weight: 9},
		{URL: "http://p2:8080", Weight: 1},
	}
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      proxies,
		RotationMode: ProxyRotationWeighted,
	})

	counts := map[string]int{}
	const iters = 1000
	for range iters {
		req := &Request{URL: "http://example.com"}
		if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
			t.Fatal(err)
		}
		counts[req.Meta[MetaKeyProxy].(string)]++
	}
	// p1 should be selected ~90% of the time; allow ±15% tolerance.
	p1Ratio := float64(counts["http://p1:8080"]) / iters
	if p1Ratio < 0.75 || p1Ratio > 1.0 {
		t.Errorf("p1 ratio = %.2f, want ~0.90 (±0.15 tolerance)", p1Ratio)
	}
}

// --- Health / failover ---

func TestProxyRotation_SkipsUnhealthyProxy(t *testing.T) {
	// MaxFailures=1: a single error marks the proxy unhealthy.
	proxies := []ProxyEntry{
		{URL: "http://bad:8080"},
		{URL: "http://good:8080"},
	}
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      proxies,
		RotationMode: ProxyRotationRoundRobin,
		MaxFailures:  1,
	})

	alwaysFail := func(_ context.Context, _ *Request) (*Response, error) {
		return nil, errors.New("connection refused")
	}

	// First request hits idx=0 (bad) and fails → unhealthy.
	req1 := &Request{URL: "http://example.com"}
	_, _ = pr.ProcessRequest(context.Background(), req1, alwaysFail)

	// Second request: round-robin base=1, skips idx=0 (unhealthy), picks idx=1.
	req2 := &Request{URL: "http://example.com"}
	if _, err := pr.ProcessRequest(context.Background(), req2, nopHandler); err != nil {
		t.Fatal(err)
	}
	if req2.Meta[MetaKeyProxy] != "http://good:8080" {
		t.Errorf("Meta[proxy] = %q, want %q", req2.Meta[MetaKeyProxy], "http://good:8080")
	}
}

func TestProxyRotation_AllUnhealthy_ReturnsError(t *testing.T) {
	proxies := []ProxyEntry{
		{URL: "http://bad1:8080"},
		{URL: "http://bad2:8080"},
	}
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      proxies,
		RotationMode: ProxyRotationRoundRobin,
		MaxFailures:  1,
	})

	alwaysFail := func(_ context.Context, _ *Request) (*Response, error) {
		return nil, errors.New("refused")
	}

	// Two requests: idx=0 and idx=1 both fail → both unhealthy.
	for range 2 {
		req := &Request{URL: "http://example.com"}
		_, _ = pr.ProcessRequest(context.Background(), req, alwaysFail)
	}

	req := &Request{URL: "http://example.com"}
	_, err := pr.ProcessRequest(context.Background(), req, nopHandler)
	if err == nil {
		t.Error("expected error when all proxies are unhealthy, got nil")
	}
}

func TestProxyRotation_ResetsCounterOnSuccess(t *testing.T) {
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:     []ProxyEntry{{URL: "http://p1:8080"}},
		MaxFailures: 3,
	})

	alwaysFail := func(_ context.Context, _ *Request) (*Response, error) {
		return nil, errors.New("err")
	}

	// Fail twice (still healthy because MaxFailures=3).
	for range 2 {
		req := &Request{URL: "http://example.com"}
		_, _ = pr.ProcessRequest(context.Background(), req, alwaysFail)
	}

	// Succeed once → counter resets.
	req := &Request{URL: "http://example.com"}
	if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}
	if s := pr.HealthStatus(); s[0] != 0 {
		t.Errorf("failure count = %d after success, want 0", s[0])
	}
}

func TestProxyRotation_HealthStatus(t *testing.T) {
	proxies := []ProxyEntry{
		{URL: "http://p1:8080"},
		{URL: "http://p2:8080"},
	}
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      proxies,
		MaxFailures:  5,
		RotationMode: ProxyRotationRoundRobin,
	})

	alwaysFail := func(_ context.Context, _ *Request) (*Response, error) {
		return nil, errors.New("err")
	}

	// Each request hits a different proxy (round-robin).
	for range 2 {
		req := &Request{URL: "http://example.com"}
		_, _ = pr.ProcessRequest(context.Background(), req, alwaysFail)
	}

	status := pr.HealthStatus()
	if status[0] != 1 {
		t.Errorf("p1 failures = %d, want 1", status[0])
	}
	if status[1] != 1 {
		t.Errorf("p2 failures = %d, want 1", status[1])
	}
}

// --- Concurrency ---

func TestProxyRotation_Concurrent(t *testing.T) {
	proxies := []ProxyEntry{
		{URL: "http://p1:8080"},
		{URL: "http://p2:8080"},
		{URL: "http://p3:8080"},
	}
	pr := NewProxyRotation(ProxyRotationConfig{
		Proxies:      proxies,
		RotationMode: ProxyRotationRoundRobin,
	})
	pool := map[string]bool{
		"http://p1:8080": true,
		"http://p2:8080": true,
		"http://p3:8080": true,
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			req := &Request{URL: "http://example.com"}
			if _, err := pr.ProcessRequest(context.Background(), req, nopHandler); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !pool[req.Meta[MetaKeyProxy].(string)] {
				t.Errorf("proxy %q not in pool", req.Meta[MetaKeyProxy])
			}
		}()
	}
	wg.Wait()
}
