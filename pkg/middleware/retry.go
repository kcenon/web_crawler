package middleware

import (
	"context"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig configures the retry middleware behavior.
type RetryConfig struct {
	MaxRetries           int
	InitialBackoff       time.Duration
	MaxBackoff           time.Duration
	Multiplier           float64
	JitterFactor         float64
	RetryableStatusCodes []int
}

// DefaultRetryConfig returns a RetryConfig with sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
		JitterFactor:   0.1,
		RetryableStatusCodes: []int{
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		},
	}
}

// Retry is a middleware that retries failed requests with exponential backoff.
type Retry struct {
	cfg RetryConfig
}

// NewRetry creates a retry middleware with the given config.
func NewRetry(cfg RetryConfig) *Retry {
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = time.Second
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if len(cfg.RetryableStatusCodes) == 0 {
		cfg.RetryableStatusCodes = DefaultRetryConfig().RetryableStatusCodes
	}
	return &Retry{cfg: cfg}
}

// ProcessRequest implements Middleware. It retries the request if the response
// has a retryable status code or an error occurred.
func (r *Retry) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	var lastResp *Response
	var lastErr error

	for attempt := 0; attempt <= r.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := r.calculateDelay(attempt, lastResp)
			if err := sleep(ctx, delay); err != nil {
				return lastResp, err
			}
		}

		resp, err := next(ctx, req)
		if err == nil && !r.isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		lastResp = resp
		lastErr = err
	}

	return lastResp, lastErr
}

// calculateDelay computes the backoff delay for the given attempt number.
// It respects the Retry-After header if present.
func (r *Retry) calculateDelay(attempt int, resp *Response) time.Duration {
	// Check Retry-After header.
	if resp != nil && resp.Headers != nil {
		if ra := resp.Headers["Retry-After"]; ra != "" {
			if d := parseRetryAfter(ra); d > 0 {
				return d
			}
		}
	}

	// Exponential backoff: initialBackoff * multiplier^(attempt-1)
	backoff := float64(r.cfg.InitialBackoff) * math.Pow(r.cfg.Multiplier, float64(attempt-1))

	// Cap at max backoff.
	if backoff > float64(r.cfg.MaxBackoff) {
		backoff = float64(r.cfg.MaxBackoff)
	}

	// Apply jitter: delay ± (delay * jitterFactor)
	if r.cfg.JitterFactor > 0 {
		jitter := backoff * r.cfg.JitterFactor
		backoff += (rand.Float64()*2 - 1) * jitter //nolint:gosec // jitter does not need cryptographic randomness
	}

	return time.Duration(backoff)
}

// isRetryableStatus returns true if the status code is in the retryable set.
func (r *Retry) isRetryableStatus(code int) bool {
	for _, c := range r.cfg.RetryableStatusCodes {
		if c == code {
			return true
		}
	}
	return false
}

// parseRetryAfter parses a Retry-After header value.
// Supports both seconds (e.g., "120") and HTTP-date formats.
func parseRetryAfter(value string) time.Duration {
	// Try parsing as seconds.
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date.
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}

	return 0
}

// sleep waits for the given duration or until the context is cancelled.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
