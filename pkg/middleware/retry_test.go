package middleware

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetry_NoRetryOnSuccess(t *testing.T) {
	var calls atomic.Int32
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls.Add(1)
		return &Response{StatusCode: 200}, nil
	}

	c := NewChain(handler)
	c.Use(NewRetry(RetryConfig{MaxRetries: 3, InitialBackoff: time.Millisecond}))

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if calls.Load() != 1 {
		t.Errorf("handler called %d times, want 1", calls.Load())
	}
}

func TestRetry_RetriesOnServerError(t *testing.T) {
	var calls atomic.Int32
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		n := calls.Add(1)
		if n < 3 {
			return &Response{StatusCode: 503}, nil
		}
		return &Response{StatusCode: 200}, nil
	}

	c := NewChain(handler)
	c.Use(NewRetry(RetryConfig{MaxRetries: 3, InitialBackoff: time.Millisecond}))

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if calls.Load() != 3 {
		t.Errorf("handler called %d times, want 3", calls.Load())
	}
}

func TestRetry_RetriesOnError(t *testing.T) {
	var calls atomic.Int32
	testErr := errors.New("connection refused")

	handler := func(_ context.Context, _ *Request) (*Response, error) {
		n := calls.Add(1)
		if n < 3 {
			return nil, testErr
		}
		return &Response{StatusCode: 200}, nil
	}

	c := NewChain(handler)
	c.Use(NewRetry(RetryConfig{MaxRetries: 3, InitialBackoff: time.Millisecond}))

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if calls.Load() != 3 {
		t.Errorf("handler called %d times, want 3", calls.Load())
	}
}

func TestRetry_MaxRetriesExhausted(t *testing.T) {
	var calls atomic.Int32
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls.Add(1)
		return &Response{StatusCode: 500}, nil
	}

	c := NewChain(handler)
	c.Use(NewRetry(RetryConfig{MaxRetries: 2, InitialBackoff: time.Millisecond}))

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("final StatusCode = %d, want 500", resp.StatusCode)
	}
	// 1 initial + 2 retries = 3 total calls.
	if calls.Load() != 3 {
		t.Errorf("handler called %d times, want 3", calls.Load())
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var calls atomic.Int32
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		n := calls.Add(1)
		if n == 1 {
			cancel() // Cancel after first attempt.
		}
		return &Response{StatusCode: 503}, nil
	}

	c := NewChain(handler)
	c.Use(NewRetry(RetryConfig{MaxRetries: 5, InitialBackoff: time.Second}))

	_, err := c.Execute(ctx, &Request{URL: "http://example.com"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestRetry_NonRetryableStatusCode(t *testing.T) {
	var calls atomic.Int32
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		calls.Add(1)
		return &Response{StatusCode: 404}, nil
	}

	c := NewChain(handler)
	c.Use(NewRetry(RetryConfig{MaxRetries: 3, InitialBackoff: time.Millisecond}))

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	if calls.Load() != 1 {
		t.Errorf("should not retry 404, called %d times", calls.Load())
	}
}

func TestRetry_429WithRetryAfterSeconds(t *testing.T) {
	var calls atomic.Int32
	handler := func(_ context.Context, _ *Request) (*Response, error) {
		n := calls.Add(1)
		if n == 1 {
			return &Response{
				StatusCode: 429,
				Headers:    map[string]string{"Retry-After": "1"},
			}, nil
		}
		return &Response{StatusCode: 200}, nil
	}

	c := NewChain(handler)
	cfg := RetryConfig{MaxRetries: 2, InitialBackoff: time.Millisecond}
	c.Use(NewRetry(cfg))

	start := time.Now()
	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	// Should wait approximately 1 second due to Retry-After header.
	if elapsed < 900*time.Millisecond {
		t.Errorf("elapsed %v, expected ≥1s due to Retry-After", elapsed)
	}
}

func TestRetry_BackoffCalculation(t *testing.T) {
	r := NewRetry(RetryConfig{
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     10 * time.Second,
		Multiplier:     2.0,
		JitterFactor:   0, // No jitter for deterministic test.
	})

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 100 * time.Millisecond},  // 100ms * 2^0
		{2, 200 * time.Millisecond},  // 100ms * 2^1
		{3, 400 * time.Millisecond},  // 100ms * 2^2
		{4, 800 * time.Millisecond},  // 100ms * 2^3
		{5, 1600 * time.Millisecond}, // 100ms * 2^4
	}

	for _, tt := range tests {
		got := r.calculateDelay(tt.attempt, nil)
		if got != tt.want {
			t.Errorf("delay(attempt=%d) = %v, want %v", tt.attempt, got, tt.want)
		}
	}
}

func TestRetry_MaxBackoffCap(t *testing.T) {
	r := NewRetry(RetryConfig{
		InitialBackoff: time.Second,
		MaxBackoff:     5 * time.Second,
		Multiplier:     10.0,
		JitterFactor:   0,
	})

	got := r.calculateDelay(3, nil) // 1s * 10^2 = 100s, capped at 5s.
	if got != 5*time.Second {
		t.Errorf("delay = %v, want 5s (capped)", got)
	}
}

func TestRetry_DefaultConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != time.Second {
		t.Errorf("InitialBackoff = %v, want 1s", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", cfg.MaxBackoff)
	}
	if len(cfg.RetryableStatusCodes) != 5 {
		t.Errorf("RetryableStatusCodes len = %d, want 5", len(cfg.RetryableStatusCodes))
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		value string
		want  bool // true if duration should be > 0
	}{
		{"120", true},
		{"0", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		got := parseRetryAfter(tt.value)
		if tt.want && got <= 0 {
			t.Errorf("parseRetryAfter(%q) = %v, want > 0", tt.value, got)
		}
		if !tt.want && got > 0 {
			t.Errorf("parseRetryAfter(%q) = %v, want 0", tt.value, got)
		}
	}
}
