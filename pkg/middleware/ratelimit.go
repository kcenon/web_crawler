package middleware

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimitConfig configures the rate limiting middleware.
type RateLimitConfig struct {
	GlobalRPS        float64            // Global requests per second (default: 10).
	DefaultDomainRPS float64            // Default per-domain RPS (default: 2).
	DomainOverrides  map[string]float64 // Per-domain RPS overrides.
	BurstSize        int                // Token bucket burst size (default: 5).
}

// DefaultRateLimitConfig returns a RateLimitConfig with sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRPS:        10,
		DefaultDomainRPS: 2,
		BurstSize:        5,
	}
}

// RateLimit is a middleware that enforces request rate limits
// both globally and per-domain using token bucket algorithm.
type RateLimit struct {
	cfg            RateLimitConfig
	globalLimiter  *rate.Limiter
	domainMu       sync.RWMutex
	domainLimiters map[string]*rate.Limiter
}

// NewRateLimit creates a rate limiting middleware with the given config.
func NewRateLimit(cfg RateLimitConfig) *RateLimit {
	if cfg.GlobalRPS <= 0 {
		cfg.GlobalRPS = 10
	}
	if cfg.DefaultDomainRPS <= 0 {
		cfg.DefaultDomainRPS = 2
	}
	if cfg.BurstSize <= 0 {
		cfg.BurstSize = 5
	}

	return &RateLimit{
		cfg:            cfg,
		globalLimiter:  rate.NewLimiter(rate.Limit(cfg.GlobalRPS), cfg.BurstSize),
		domainLimiters: make(map[string]*rate.Limiter),
	}
}

// ProcessRequest implements Middleware. It waits for both the global
// and per-domain rate limiters before forwarding the request.
func (rl *RateLimit) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	// Wait for global rate limit.
	if err := rl.globalLimiter.Wait(ctx); err != nil {
		return nil, err
	}

	// Wait for per-domain rate limit.
	domain := extractHost(req.URL)
	if domain != "" {
		limiter := rl.getDomainLimiter(domain)
		if err := limiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	return next(ctx, req)
}

// getDomainLimiter returns the rate limiter for a specific domain,
// creating one lazily if it doesn't exist.
func (rl *RateLimit) getDomainLimiter(domain string) *rate.Limiter {
	// Fast path: read lock.
	rl.domainMu.RLock()
	limiter, ok := rl.domainLimiters[domain]
	rl.domainMu.RUnlock()
	if ok {
		return limiter
	}

	// Slow path: write lock.
	rl.domainMu.Lock()
	defer rl.domainMu.Unlock()

	// Double-check.
	if limiter, ok = rl.domainLimiters[domain]; ok {
		return limiter
	}

	rps := rl.cfg.DefaultDomainRPS
	if override, exists := rl.cfg.DomainOverrides[domain]; exists {
		rps = override
	}

	limiter = rate.NewLimiter(rate.Limit(rps), rl.cfg.BurstSize)
	rl.domainLimiters[domain] = limiter
	return limiter
}

// SetDomainRate dynamically updates the rate limit for a specific domain.
// This can be used to apply crawl-delay from robots.txt.
func (rl *RateLimit) SetDomainRate(domain string, rps float64) {
	rl.domainMu.Lock()
	defer rl.domainMu.Unlock()

	if limiter, ok := rl.domainLimiters[domain]; ok {
		limiter.SetLimit(rate.Limit(rps))
	} else {
		rl.domainLimiters[domain] = rate.NewLimiter(rate.Limit(rps), rl.cfg.BurstSize)
	}
}

// extractHost extracts the host portion of a URL.
func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	return strings.ToLower(host)
}
