package middleware

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/url"
	"sync"
	"sync/atomic"
)

// Proxy rotation mode constants.
const (
	ProxyRotationRoundRobin = "round-robin" // Cycle through proxies in order (default).
	ProxyRotationRandom     = "random"      // Pick a proxy at random each request.
	ProxyRotationWeighted   = "weighted"    // Pick proportional to ProxyEntry.Weight.
)

// MetaKeyProxy is the key used in Request.Meta to communicate the selected
// proxy URL (with embedded credentials) to the terminal handler.
const MetaKeyProxy = "proxy"

// ProxyEntry describes a single proxy server in the rotation pool.
type ProxyEntry struct {
	// URL is the proxy address. Supported schemes: http://, https://, socks5://.
	URL string

	// Username and Password are optional proxy authentication credentials.
	// When non-empty they are embedded in the URL written to Request.Meta.
	Username string
	Password string

	// Weight is the relative selection weight for ProxyRotationWeighted.
	// Values ≤ 0 are normalised to 1.
	Weight int
}

// ProxyRotationConfig configures the proxy rotation middleware.
type ProxyRotationConfig struct {
	// Proxies is the pool of proxy servers to rotate through. Required.
	Proxies []ProxyEntry

	// RotationMode controls how a proxy is selected from the pool.
	// Accepted values: ProxyRotationRoundRobin (default), ProxyRotationRandom,
	// ProxyRotationWeighted.
	RotationMode string

	// MaxFailures is the number of consecutive errors after which a proxy is
	// considered unhealthy and skipped. Default: 3. Must be > 0.
	MaxFailures int
}

// ProxyRotation is a middleware that selects a proxy from a pool and stores the
// choice in req.Meta[MetaKeyProxy] before forwarding the request to the next
// handler.
//
// Health tracking: after each call to next, if an error is returned the chosen
// proxy's consecutive-failure counter is incremented; on success it is reset to
// zero. Proxies whose counter reaches MaxFailures are skipped in future picks
// until they recover (i.e. until a successful request resets their counter).
type ProxyRotation struct {
	cfg      ProxyRotationConfig
	index    atomic.Uint64 // round-robin counter; incremented atomically
	failures sync.Map      // map[int]*atomic.Int64: consecutive failure count per pool index
}

// NewProxyRotation creates a proxy rotation middleware from cfg.
// Panics if cfg.Proxies is empty.
func NewProxyRotation(cfg ProxyRotationConfig) *ProxyRotation {
	if len(cfg.Proxies) == 0 {
		panic("middleware.NewProxyRotation: Proxies must not be empty")
	}
	switch cfg.RotationMode {
	case ProxyRotationRoundRobin, ProxyRotationRandom, ProxyRotationWeighted:
		// valid
	default:
		cfg.RotationMode = ProxyRotationRoundRobin
	}
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 3
	}
	// Normalise weights: any value ≤ 0 becomes 1.
	for i := range cfg.Proxies {
		if cfg.Proxies[i].Weight <= 0 {
			cfg.Proxies[i].Weight = 1
		}
	}
	return &ProxyRotation{cfg: cfg}
}

// ProcessRequest implements Middleware. It selects a healthy proxy, records it
// in req.Meta[MetaKeyProxy], calls next, then updates the proxy's health counter.
func (pr *ProxyRotation) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	idx, ok := pr.pick()
	if !ok {
		return nil, fmt.Errorf("middleware.ProxyRotation: no healthy proxy available")
	}

	if req.Meta == nil {
		req.Meta = make(map[string]any)
	}
	req.Meta[MetaKeyProxy] = pr.proxyURL(idx)

	resp, err := next(ctx, req)
	pr.recordResult(idx, err)
	return resp, err
}

// pick selects a proxy index according to the configured rotation mode.
// Returns (index, true) on success, or (-1, false) if all proxies are unhealthy.
func (pr *ProxyRotation) pick() (int, bool) {
	n := len(pr.cfg.Proxies)

	switch pr.cfg.RotationMode {
	case ProxyRotationRandom:
		start := int(rand.Uint64N(uint64(n))) //nolint:gosec // non-cryptographic selection
		for i := 0; i < n; i++ {
			idx := (start + i) % n
			if pr.isHealthy(idx) {
				return idx, true
			}
		}
	case ProxyRotationWeighted:
		return pr.pickWeighted()
	default: // ProxyRotationRoundRobin
		base := int(pr.index.Add(1) - 1)
		for i := 0; i < n; i++ {
			idx := (base + i) % n
			if pr.isHealthy(idx) {
				return idx, true
			}
		}
	}
	return -1, false
}

// pickWeighted selects a proxy proportional to its Weight, skipping unhealthy ones.
func (pr *ProxyRotation) pickWeighted() (int, bool) {
	type entry struct{ idx, w int }
	pool := make([]entry, 0, len(pr.cfg.Proxies))
	total := 0
	for i, p := range pr.cfg.Proxies {
		if pr.isHealthy(i) {
			pool = append(pool, entry{i, p.Weight})
			total += p.Weight
		}
	}
	if total == 0 {
		return -1, false
	}

	r := int(rand.Uint64N(uint64(total))) //nolint:gosec
	for _, e := range pool {
		r -= e.w
		if r < 0 {
			return e.idx, true
		}
	}
	return pool[len(pool)-1].idx, true
}

// isHealthy reports whether the proxy at idx has fewer consecutive errors than MaxFailures.
func (pr *ProxyRotation) isHealthy(idx int) bool {
	v, ok := pr.failures.Load(idx)
	if !ok {
		return true
	}
	return v.(*atomic.Int64).Load() < int64(pr.cfg.MaxFailures)
}

// recordResult updates the consecutive-failure counter for the proxy at idx.
// On error the counter is incremented; on success it is reset to zero.
func (pr *ProxyRotation) recordResult(idx int, err error) {
	ctr := pr.counter(idx)
	if err != nil {
		ctr.Add(1)
	} else {
		ctr.Store(0)
	}
}

// counter returns the failure counter for idx, creating it lazily if absent.
func (pr *ProxyRotation) counter(idx int) *atomic.Int64 {
	v, _ := pr.failures.LoadOrStore(idx, new(atomic.Int64))
	return v.(*atomic.Int64)
}

// proxyURL builds the proxy URL string with embedded credentials (if any).
func (pr *ProxyRotation) proxyURL(idx int) string {
	e := pr.cfg.Proxies[idx]
	if e.Username == "" {
		return e.URL
	}
	u, err := url.Parse(e.URL)
	if err != nil {
		return e.URL // return as-is if unparseable
	}
	u.User = url.UserPassword(e.Username, e.Password)
	return u.String()
}

// HealthStatus returns the current consecutive-failure count for each proxy in
// pool order. A value of zero means the proxy is healthy; a value ≥ MaxFailures
// means it is being skipped.
func (pr *ProxyRotation) HealthStatus() []int64 {
	out := make([]int64, len(pr.cfg.Proxies))
	for i := range out {
		if v, ok := pr.failures.Load(i); ok {
			out[i] = v.(*atomic.Int64).Load()
		}
	}
	return out
}
