package middleware

import (
	"context"
	"hash/fnv"
	"math/rand/v2"
	"sync/atomic"
)

// Rotation mode constants for UserAgentConfig.
const (
	RotationRoundRobin = "round-robin" // Cycle through agents in order (default).
	RotationRandom     = "random"      // Pick an agent at random each request.
	RotationPerDomain  = "per-domain"  // Always use the same agent for a given host.
)

// DefaultUserAgents is a pool of realistic browser User-Agent strings covering
// major browsers and operating systems to blend in with normal web traffic.
var DefaultUserAgents = []string{
	// Chrome on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	// Chrome on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	// Chrome on Linux
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
	// Firefox on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:123.0) Gecko/20100101 Firefox/123.0",
	// Firefox on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.3; rv:123.0) Gecko/20100101 Firefox/123.0",
	// Firefox on Linux
	"Mozilla/5.0 (X11; Linux x86_64; rv:123.0) Gecko/20100101 Firefox/123.0",
	// Safari on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_3_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15",
	// Edge on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
}

// UserAgentConfig configures the User-Agent rotation middleware.
type UserAgentConfig struct {
	// UserAgents is the pool of User-Agent strings to rotate through.
	// If empty or nil, DefaultUserAgents is used.
	UserAgents []string

	// RotationMode controls how an agent is selected from the pool.
	// Accepted values: RotationRoundRobin (default), RotationRandom, RotationPerDomain.
	RotationMode string
}

// DefaultUserAgentConfig returns a UserAgentConfig with sensible defaults.
func DefaultUserAgentConfig() UserAgentConfig {
	return UserAgentConfig{
		UserAgents:   DefaultUserAgents,
		RotationMode: RotationRoundRobin,
	}
}

// UserAgent is a middleware that sets the User-Agent request header by rotating
// through a configurable pool of strings according to the selected rotation mode.
type UserAgent struct {
	cfg   UserAgentConfig
	index atomic.Uint64 // Used by round-robin; incremented atomically.
}

// NewUserAgent creates a User-Agent rotation middleware.
// If cfg.UserAgents is empty, DefaultUserAgents is used.
// If cfg.RotationMode is unrecognised, RotationRoundRobin is used.
func NewUserAgent(cfg UserAgentConfig) *UserAgent {
	if len(cfg.UserAgents) == 0 {
		cfg.UserAgents = DefaultUserAgents
	}
	switch cfg.RotationMode {
	case RotationRoundRobin, RotationRandom, RotationPerDomain:
		// valid
	default:
		cfg.RotationMode = RotationRoundRobin
	}
	return &UserAgent{cfg: cfg}
}

// ProcessRequest implements Middleware. It selects a User-Agent from the pool
// according to the configured rotation mode, sets the "User-Agent" request header,
// and forwards the request to the next handler.
func (ua *UserAgent) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	agent := ua.pick(req.URL)

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers["User-Agent"] = agent

	return next(ctx, req)
}

// pick selects a User-Agent string from the pool using the configured rotation mode.
func (ua *UserAgent) pick(rawURL string) string {
	pool := ua.cfg.UserAgents
	n := uint64(len(pool))

	switch ua.cfg.RotationMode {
	case RotationRandom:
		return pool[rand.Uint64N(n)] //nolint:gosec // non-cryptographic selection is intentional
	case RotationPerDomain:
		host := extractHost(rawURL)
		if host == "" {
			break
		}
		h := fnv.New64a()
		_, _ = h.Write([]byte(host))
		return pool[h.Sum64()%n]
	}

	// Default: round-robin (also fallback for per-domain with unparseable URL).
	idx := ua.index.Add(1) - 1
	return pool[idx%n]
}
