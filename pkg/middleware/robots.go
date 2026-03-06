package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// ErrDisallowed is returned when a URL is blocked by robots.txt.
var ErrDisallowed = errors.New("blocked by robots.txt")

// RobotsConfig configures the robots.txt compliance middleware.
type RobotsConfig struct {
	Enabled        bool                                     // Whether enforcement is active (default: true).
	UserAgent      string                                   // User-agent for matching (default: "WebCrawlerSDK/1.0").
	CacheExpiry    time.Duration                            // Cache TTL for parsed robots.txt (default: 24h).
	RespectNoIndex bool                                     // Check X-Robots-Tag noindex (default: true).
	OnCrawlDelay   func(domain string, delay time.Duration) // Called when Crawl-delay directive is found.
}

// DefaultRobotsConfig returns a RobotsConfig with sensible defaults.
func DefaultRobotsConfig() RobotsConfig {
	return RobotsConfig{
		Enabled:        true,
		UserAgent:      "WebCrawlerSDK/1.0",
		CacheExpiry:    24 * time.Hour,
		RespectNoIndex: true,
	}
}

// RobotsFetcher fetches robots.txt content for a given URL.
// Returns the HTTP status code and body bytes.
type RobotsFetcher func(ctx context.Context, robotsURL string) (statusCode int, body []byte, err error)

// Robots is a middleware that enforces robots.txt rules.
// It automatically fetches, parses, and caches robots.txt for each domain.
type Robots struct {
	cfg     RobotsConfig
	fetcher RobotsFetcher
	mu      sync.RWMutex
	cache   map[string]*robotsCacheEntry
}

type robotsCacheEntry struct {
	group     *robotstxt.Group
	sitemaps  []string
	fetchedAt time.Time
}

// NewRobots creates a robots.txt compliance middleware.
// The fetcher is used to retrieve robots.txt content and should bypass
// the middleware chain to avoid circular dependencies.
func NewRobots(cfg RobotsConfig, fetcher RobotsFetcher) *Robots {
	if cfg.UserAgent == "" {
		cfg.UserAgent = "WebCrawlerSDK/1.0"
	}
	if cfg.CacheExpiry <= 0 {
		cfg.CacheExpiry = 24 * time.Hour
	}
	return &Robots{
		cfg:     cfg,
		fetcher: fetcher,
		cache:   make(map[string]*robotsCacheEntry),
	}
}

// ProcessRequest implements Middleware. It checks robots.txt rules
// before forwarding the request and inspects X-Robots-Tag on the response.
func (r *Robots) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	if !r.cfg.Enabled {
		return next(ctx, req)
	}

	u, err := url.Parse(req.URL)
	if err != nil {
		return next(ctx, req)
	}

	if u.Host == "" {
		return next(ctx, req)
	}

	domain := strings.ToLower(u.Scheme + "://" + u.Host)

	entry, err := r.getOrFetch(ctx, domain)
	if err != nil {
		// Network error fetching robots.txt: fail open.
		return next(ctx, req)
	}

	path := u.RequestURI()
	if !entry.group.Test(path) {
		return nil, fmt.Errorf("%w: %s", ErrDisallowed, req.URL)
	}

	resp, err := next(ctx, req)
	if err != nil {
		return nil, err
	}

	// Check X-Robots-Tag for noindex directive.
	if r.cfg.RespectNoIndex && resp != nil && resp.Headers != nil {
		if tag := resp.Headers["X-Robots-Tag"]; strings.Contains(strings.ToLower(tag), "noindex") {
			if resp.Meta == nil {
				resp.Meta = make(map[string]any)
			}
			resp.Meta["noindex"] = true
		}
	}

	return resp, nil
}

// Sitemaps returns cached sitemap URLs for a domain (e.g., "http://example.com").
func (r *Robots) Sitemaps(domain string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if entry, ok := r.cache[domain]; ok {
		return entry.sitemaps
	}
	return nil
}

// getOrFetch returns a cached robots.txt entry or fetches a new one.
func (r *Robots) getOrFetch(ctx context.Context, domain string) (*robotsCacheEntry, error) {
	// Fast path: read lock.
	r.mu.RLock()
	entry, ok := r.cache[domain]
	r.mu.RUnlock()
	if ok && time.Since(entry.fetchedAt) < r.cfg.CacheExpiry {
		return entry, nil
	}

	// Slow path: write lock.
	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if entry, ok = r.cache[domain]; ok && time.Since(entry.fetchedAt) < r.cfg.CacheExpiry {
		return entry, nil
	}

	robotsURL := domain + "/robots.txt"
	statusCode, body, err := r.fetcher(ctx, robotsURL)
	if err != nil {
		return nil, err
	}

	// 5xx: temporary unavailability → disallow all per RFC 9309.
	// Don't cache so we retry on next request.
	if statusCode >= 500 {
		denyAll, _ := robotstxt.FromBytes([]byte("User-agent: *\nDisallow: /\n"))
		return &robotsCacheEntry{
			group:     denyAll.FindGroup(r.cfg.UserAgent),
			fetchedAt: time.Now(),
		}, nil
	}

	data, err := robotstxt.FromStatusAndBytes(statusCode, body)
	if err != nil {
		return nil, err
	}

	group := data.FindGroup(r.cfg.UserAgent)

	// Notify about crawl-delay if callback is set.
	if r.cfg.OnCrawlDelay != nil && group.CrawlDelay > 0 {
		host := strings.TrimPrefix(domain, "http://")
		host = strings.TrimPrefix(host, "https://")
		r.cfg.OnCrawlDelay(host, group.CrawlDelay)
	}

	entry = &robotsCacheEntry{
		group:     group,
		sitemaps:  data.Sitemaps,
		fetchedAt: time.Now(),
	}
	r.cache[domain] = entry

	return entry, nil
}
