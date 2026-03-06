package middleware

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func staticFetcher(statusCode int, body string) RobotsFetcher {
	return func(_ context.Context, _ string) (int, []byte, error) {
		return statusCode, []byte(body), nil
	}
}

func TestRobots_AllowsPermittedPaths(t *testing.T) {
	fetcher := staticFetcher(200, "User-agent: *\nDisallow: /private/\n")
	r := NewRobots(DefaultRobotsConfig(), fetcher)

	c := NewChain(okHandler)
	c.Use(r)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com/public/page"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestRobots_BlocksDisallowedPaths(t *testing.T) {
	fetcher := staticFetcher(200, "User-agent: *\nDisallow: /private/\n")
	r := NewRobots(DefaultRobotsConfig(), fetcher)

	c := NewChain(okHandler)
	c.Use(r)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/private/secret"})
	if !errors.Is(err, ErrDisallowed) {
		t.Errorf("error = %v, want ErrDisallowed", err)
	}
}

func TestRobots_UserAgentMatching(t *testing.T) {
	robotsTxt := "User-agent: WebCrawlerSDK/1.0\nDisallow: /sdk-blocked/\n\nUser-agent: *\nDisallow: /general-blocked/\n"
	fetcher := staticFetcher(200, robotsTxt)
	r := NewRobots(DefaultRobotsConfig(), fetcher)

	c := NewChain(okHandler)
	c.Use(r)

	// SDK-specific path should be blocked.
	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/sdk-blocked/page"})
	if !errors.Is(err, ErrDisallowed) {
		t.Errorf("sdk-blocked: error = %v, want ErrDisallowed", err)
	}

	// General-blocked should be allowed for our specific agent
	// (specific agent rules take precedence over wildcard).
	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com/general-blocked/page"})
	if err != nil {
		t.Fatalf("general-blocked: unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("general-blocked: StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestRobots_CachesResults(t *testing.T) {
	var fetchCount atomic.Int32
	fetcher := func(_ context.Context, _ string) (int, []byte, error) {
		fetchCount.Add(1)
		return 200, []byte("User-agent: *\nDisallow:\n"), nil
	}

	r := NewRobots(DefaultRobotsConfig(), fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	for i := 0; i < 5; i++ {
		_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
		if err != nil {
			t.Fatal(err)
		}
	}

	if fetchCount.Load() != 1 {
		t.Errorf("fetched %d times, want 1 (cached)", fetchCount.Load())
	}
}

func TestRobots_CacheExpiry(t *testing.T) {
	var fetchCount atomic.Int32
	fetcher := func(_ context.Context, _ string) (int, []byte, error) {
		fetchCount.Add(1)
		return 200, []byte("User-agent: *\nDisallow:\n"), nil
	}

	cfg := DefaultRobotsConfig()
	cfg.CacheExpiry = 10 * time.Millisecond
	r := NewRobots(cfg, fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	_, err = c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if err != nil {
		t.Fatal(err)
	}

	if fetchCount.Load() != 2 {
		t.Errorf("fetched %d times, want 2 (cache expired)", fetchCount.Load())
	}
}

func TestRobots_Disabled(t *testing.T) {
	fetcher := staticFetcher(200, "User-agent: *\nDisallow: /\n")
	cfg := DefaultRobotsConfig()
	cfg.Enabled = false
	r := NewRobots(cfg, fetcher)

	c := NewChain(okHandler)
	c.Use(r)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com/blocked"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("disabled: StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestRobots_404AllowsAll(t *testing.T) {
	fetcher := staticFetcher(404, "")
	r := NewRobots(DefaultRobotsConfig(), fetcher)

	c := NewChain(okHandler)
	c.Use(r)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com/anything"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("404 robots.txt: StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestRobots_5xxNotCached(t *testing.T) {
	var fetchCount atomic.Int32
	fetcher := func(_ context.Context, _ string) (int, []byte, error) {
		n := fetchCount.Add(1)
		if n == 1 {
			return 500, nil, nil
		}
		return 200, []byte("User-agent: *\nDisallow:\n"), nil
	}

	r := NewRobots(DefaultRobotsConfig(), fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	// First request: 5xx → disallow all per RFC 9309.
	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if !errors.Is(err, ErrDisallowed) {
		t.Errorf("first request: error = %v, want ErrDisallowed", err)
	}

	// Second request: retry (not cached), server returns 200 → allow.
	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("second request: StatusCode = %d, want 200", resp.StatusCode)
	}
	if fetchCount.Load() != 2 {
		t.Errorf("fetched %d times, want 2 (5xx not cached)", fetchCount.Load())
	}
}

func TestRobots_FetchErrorFailOpen(t *testing.T) {
	fetcher := func(_ context.Context, _ string) (int, []byte, error) {
		return 0, nil, errors.New("network error")
	}

	r := NewRobots(DefaultRobotsConfig(), fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if err != nil {
		t.Fatalf("fetch error should fail open: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

func TestRobots_CrawlDelayCallback(t *testing.T) {
	fetcher := staticFetcher(200, "User-agent: *\nCrawl-delay: 5\n")

	var gotDomain string
	var gotDelay time.Duration
	cfg := DefaultRobotsConfig()
	cfg.OnCrawlDelay = func(domain string, delay time.Duration) {
		gotDomain = domain
		gotDelay = delay
	}

	r := NewRobots(cfg, fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if err != nil {
		t.Fatal(err)
	}

	if gotDomain != "example.com" {
		t.Errorf("crawl-delay domain = %q, want %q", gotDomain, "example.com")
	}
	if gotDelay != 5*time.Second {
		t.Errorf("crawl-delay = %v, want 5s", gotDelay)
	}
}

func TestRobots_SitemapExtraction(t *testing.T) {
	robotsTxt := "User-agent: *\nDisallow:\nSitemap: http://example.com/sitemap1.xml\nSitemap: http://example.com/sitemap2.xml\n"
	fetcher := staticFetcher(200, robotsTxt)

	r := NewRobots(DefaultRobotsConfig(), fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if err != nil {
		t.Fatal(err)
	}

	sitemaps := r.Sitemaps("http://example.com")
	if len(sitemaps) != 2 {
		t.Fatalf("sitemaps count = %d, want 2", len(sitemaps))
	}
	if sitemaps[0] != "http://example.com/sitemap1.xml" {
		t.Errorf("sitemaps[0] = %q", sitemaps[0])
	}
}

func TestRobots_DifferentDomains(t *testing.T) {
	fetcher := func(_ context.Context, robotsURL string) (int, []byte, error) {
		if strings.Contains(robotsURL, "blocked.com") {
			return 200, []byte("User-agent: *\nDisallow: /\n"), nil
		}
		return 200, []byte("User-agent: *\nDisallow:\n"), nil
	}

	r := NewRobots(DefaultRobotsConfig(), fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://allowed.com/page"})
	if err != nil {
		t.Fatalf("allowed.com: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("allowed.com: StatusCode = %d", resp.StatusCode)
	}

	_, err = c.Execute(context.Background(), &Request{URL: "http://blocked.com/page"})
	if !errors.Is(err, ErrDisallowed) {
		t.Errorf("blocked.com: error = %v, want ErrDisallowed", err)
	}
}

func TestRobots_RespectNoIndex(t *testing.T) {
	fetcher := staticFetcher(200, "User-agent: *\nDisallow:\n")
	r := NewRobots(DefaultRobotsConfig(), fetcher)

	handler := func(_ context.Context, _ *Request) (*Response, error) {
		return &Response{
			StatusCode: 200,
			Headers:    map[string]string{"X-Robots-Tag": "noindex, nofollow"},
		}, nil
	}

	c := NewChain(handler)
	c.Use(r)

	resp, err := c.Execute(context.Background(), &Request{URL: "http://example.com/page"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Meta == nil || resp.Meta["noindex"] != true {
		t.Error("expected noindex=true in response meta")
	}
}

func TestRobots_DefaultConfig(t *testing.T) {
	cfg := DefaultRobotsConfig()
	if !cfg.Enabled {
		t.Error("Enabled should be true")
	}
	if cfg.UserAgent != "WebCrawlerSDK/1.0" {
		t.Errorf("UserAgent = %q, want WebCrawlerSDK/1.0", cfg.UserAgent)
	}
	if cfg.CacheExpiry != 24*time.Hour {
		t.Errorf("CacheExpiry = %v, want 24h", cfg.CacheExpiry)
	}
	if !cfg.RespectNoIndex {
		t.Error("RespectNoIndex should be true")
	}
}

func TestRobots_AllowDisallowPrecedence(t *testing.T) {
	robotsTxt := "User-agent: *\nDisallow: /search\nAllow: /search/about\nSitemap: http://www.google.com/sitemap.xml\n"
	fetcher := staticFetcher(200, robotsTxt)

	r := NewRobots(DefaultRobotsConfig(), fetcher)
	c := NewChain(okHandler)
	c.Use(r)

	// /search should be disallowed.
	_, err := c.Execute(context.Background(), &Request{URL: "http://www.google.com/search?q=test"})
	if !errors.Is(err, ErrDisallowed) {
		t.Errorf("/search: error = %v, want ErrDisallowed", err)
	}

	// /search/about should be allowed (Allow overrides Disallow for more specific path).
	resp, err := c.Execute(context.Background(), &Request{URL: "http://www.google.com/search/about"})
	if err != nil {
		t.Fatalf("/search/about: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("/search/about: StatusCode = %d", resp.StatusCode)
	}

	// Root should be allowed.
	resp, err = c.Execute(context.Background(), &Request{URL: "http://www.google.com/"})
	if err != nil {
		t.Fatalf("/: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("/: StatusCode = %d", resp.StatusCode)
	}
}
