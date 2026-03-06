package middleware

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func okHandler(_ context.Context, _ *Request) (*Response, error) {
	return &Response{StatusCode: 200}, nil
}

func TestRateLimit_AllowsBurst(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        100,
		DefaultDomainRPS: 100,
		BurstSize:        5,
	})

	c := NewChain(okHandler)
	c.Use(rl)

	// Burst of 5 requests should complete almost instantly.
	start := time.Now()
	for i := 0; i < 5; i++ {
		_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
		if err != nil {
			t.Fatalf("request %d error: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("burst of 5 took %v, expected <100ms", elapsed)
	}
}

func TestRateLimit_EnforcesGlobalRate(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        5,
		DefaultDomainRPS: 100,
		BurstSize:        1,
	})

	c := NewChain(okHandler)
	c.Use(rl)

	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
		if err != nil {
			t.Fatalf("request %d error: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	// 3 requests at 5 RPS with burst=1: first instant, 2 more at ~200ms intervals.
	// Should take at least 300ms total.
	if elapsed < 300*time.Millisecond {
		t.Errorf("3 requests at 5 RPS took %v, expected ≥300ms", elapsed)
	}
}

func TestRateLimit_EnforcesPerDomainRate(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        100,
		DefaultDomainRPS: 5,
		BurstSize:        1,
	})

	c := NewChain(okHandler)
	c.Use(rl)

	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := c.Execute(context.Background(), &Request{URL: "http://slow.example.com"})
		if err != nil {
			t.Fatalf("request %d error: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	if elapsed < 300*time.Millisecond {
		t.Errorf("3 requests to same domain at 5 RPS took %v, expected ≥300ms", elapsed)
	}
}

func TestRateLimit_DifferentDomainsIndependent(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        100,
		DefaultDomainRPS: 100,
		BurstSize:        5,
	})

	c := NewChain(okHandler)
	c.Use(rl)

	domains := []string{
		"http://a.example.com",
		"http://b.example.com",
		"http://c.example.com",
	}

	start := time.Now()
	for _, domain := range domains {
		_, err := c.Execute(context.Background(), &Request{URL: domain})
		if err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)

	// Different domains should not block each other.
	if elapsed > 100*time.Millisecond {
		t.Errorf("requests to 3 different domains took %v, expected <100ms", elapsed)
	}
}

func TestRateLimit_DomainOverrides(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        100,
		DefaultDomainRPS: 100,
		DomainOverrides: map[string]float64{
			"slow.example.com": 5,
		},
		BurstSize: 1,
	})

	c := NewChain(okHandler)
	c.Use(rl)

	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := c.Execute(context.Background(), &Request{URL: "http://slow.example.com"})
		if err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)

	if elapsed < 300*time.Millisecond {
		t.Errorf("overridden domain took %v, expected ≥300ms", elapsed)
	}
}

func TestRateLimit_ContextCancellation(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        1,
		DefaultDomainRPS: 1,
		BurstSize:        1,
	})

	c := NewChain(okHandler)
	c.Use(rl)

	// First request consumes the burst.
	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}

	// Second request should wait but cancel immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = c.Execute(ctx, &Request{URL: "http://example.com"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestRateLimit_SetDomainRate(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        100,
		DefaultDomainRPS: 100,
		BurstSize:        1,
	})

	// Dynamically set a slow rate for a domain.
	rl.SetDomainRate("slow.example.com", 5)

	c := NewChain(okHandler)
	c.Use(rl)

	start := time.Now()
	for i := 0; i < 3; i++ {
		_, err := c.Execute(context.Background(), &Request{URL: "http://slow.example.com"})
		if err != nil {
			t.Fatal(err)
		}
	}
	elapsed := time.Since(start)

	if elapsed < 300*time.Millisecond {
		t.Errorf("dynamically limited domain took %v, expected ≥300ms", elapsed)
	}
}

func TestRateLimit_ConcurrentAccess(t *testing.T) {
	rl := NewRateLimit(RateLimitConfig{
		GlobalRPS:        1000,
		DefaultDomainRPS: 1000,
		BurstSize:        100,
	})

	c := NewChain(okHandler)
	c.Use(rl)

	var wg sync.WaitGroup
	var errCount atomic.Int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			domain := "http://example.com"
			if idx%2 == 0 {
				domain = "http://other.com"
			}
			_, err := c.Execute(context.Background(), &Request{URL: domain})
			if err != nil {
				errCount.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if errCount.Load() > 0 {
		t.Errorf("got %d errors in concurrent access", errCount.Load())
	}
}

func TestRateLimit_DefaultConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	if cfg.GlobalRPS != 10 {
		t.Errorf("GlobalRPS = %v, want 10", cfg.GlobalRPS)
	}
	if cfg.DefaultDomainRPS != 2 {
		t.Errorf("DefaultDomainRPS = %v, want 2", cfg.DefaultDomainRPS)
	}
	if cfg.BurstSize != 5 {
		t.Errorf("BurstSize = %d, want 5", cfg.BurstSize)
	}
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"http://example.com/path", "example.com"},
		{"https://SUB.Example.COM:8080/path", "sub.example.com"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		got := extractHost(tt.url)
		if got != tt.want {
			t.Errorf("extractHost(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
