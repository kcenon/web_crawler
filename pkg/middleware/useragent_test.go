package middleware

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// nopHandler is a terminal handler that always returns 200.
func nopHandler(_ context.Context, _ *Request) (*Response, error) {
	return &Response{StatusCode: 200}, nil
}

func TestUserAgent_DefaultConfig(t *testing.T) {
	cfg := DefaultUserAgentConfig()
	if len(cfg.UserAgents) == 0 {
		t.Error("DefaultUserAgentConfig: UserAgents must not be empty")
	}
	if cfg.RotationMode != RotationRoundRobin {
		t.Errorf("DefaultUserAgentConfig: RotationMode = %q, want %q", cfg.RotationMode, RotationRoundRobin)
	}
}

func TestUserAgent_EmptyPoolFallsBackToDefaults(t *testing.T) {
	ua := NewUserAgent(UserAgentConfig{UserAgents: nil})
	if len(ua.cfg.UserAgents) == 0 {
		t.Error("NewUserAgent with nil pool: expected fallback to DefaultUserAgents")
	}
}

func TestUserAgent_InvalidModeFallsBackToRoundRobin(t *testing.T) {
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   DefaultUserAgents,
		RotationMode: "unknown-mode",
	})
	if ua.cfg.RotationMode != RotationRoundRobin {
		t.Errorf("RotationMode = %q, want %q", ua.cfg.RotationMode, RotationRoundRobin)
	}
}

func TestUserAgent_SetsHeader(t *testing.T) {
	req := &Request{URL: "http://example.com"}
	c := NewChain(nopHandler)
	c.Use(NewUserAgent(DefaultUserAgentConfig()))

	if _, err := c.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}

	if req.Headers["User-Agent"] == "" {
		t.Error("User-Agent header must not be empty after middleware")
	}
}

func TestUserAgent_InitialisesNilHeaderMap(t *testing.T) {
	req := &Request{URL: "http://example.com", Headers: nil}
	ua := NewUserAgent(DefaultUserAgentConfig())
	if _, err := ua.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}
	if req.Headers == nil {
		t.Error("Headers map must be initialised by middleware")
	}
	if req.Headers["User-Agent"] == "" {
		t.Error("User-Agent header must be set")
	}
}

func TestUserAgent_RoundRobin_Cycles(t *testing.T) {
	pool := []string{"UA-A", "UA-B", "UA-C"}
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   pool,
		RotationMode: RotationRoundRobin,
	})

	req := func() *Request { return &Request{URL: "http://example.com"} }

	for i := 0; i < len(pool)*3; i++ {
		r := req()
		if _, err := ua.ProcessRequest(context.Background(), r, nopHandler); err != nil {
			t.Fatal(err)
		}
		want := pool[i%len(pool)]
		if r.Headers["User-Agent"] != want {
			t.Errorf("request %d: User-Agent = %q, want %q", i, r.Headers["User-Agent"], want)
		}
	}
}

func TestUserAgent_RoundRobin_SingleAgent(t *testing.T) {
	pool := []string{"only-agent"}
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   pool,
		RotationMode: RotationRoundRobin,
	})

	for i := 0; i < 5; i++ {
		r := &Request{URL: "http://example.com"}
		if _, err := ua.ProcessRequest(context.Background(), r, nopHandler); err != nil {
			t.Fatal(err)
		}
		if r.Headers["User-Agent"] != "only-agent" {
			t.Errorf("User-Agent = %q, want %q", r.Headers["User-Agent"], "only-agent")
		}
	}
}

func TestUserAgent_RoundRobin_Concurrent(t *testing.T) {
	pool := []string{"UA-A", "UA-B", "UA-C"}
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   pool,
		RotationMode: RotationRoundRobin,
	})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			r := &Request{URL: "http://example.com"}
			if _, err := ua.ProcessRequest(context.Background(), r, nopHandler); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			agent := r.Headers["User-Agent"]
			valid := false
			for _, p := range pool {
				if agent == p {
					valid = true
					break
				}
			}
			if !valid {
				t.Errorf("User-Agent %q not in pool", agent)
			}
		}()
	}
	wg.Wait()
}

func TestUserAgent_Random_PicksFromPool(t *testing.T) {
	pool := []string{"UA-A", "UA-B", "UA-C"}
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   pool,
		RotationMode: RotationRandom,
	})

	for range 30 {
		r := &Request{URL: "http://example.com"}
		if _, err := ua.ProcessRequest(context.Background(), r, nopHandler); err != nil {
			t.Fatal(err)
		}
		agent := r.Headers["User-Agent"]
		found := false
		for _, p := range pool {
			if agent == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("random User-Agent %q is not in pool", agent)
		}
	}
}

func TestUserAgent_PerDomain_Stable(t *testing.T) {
	pool := []string{"UA-A", "UA-B", "UA-C"}
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   pool,
		RotationMode: RotationPerDomain,
	})

	// Same domain must always produce the same agent.
	r1 := &Request{URL: "http://example.com/page1"}
	r2 := &Request{URL: "http://example.com/page2"}

	if _, err := ua.ProcessRequest(context.Background(), r1, nopHandler); err != nil {
		t.Fatal(err)
	}
	if _, err := ua.ProcessRequest(context.Background(), r2, nopHandler); err != nil {
		t.Fatal(err)
	}
	if r1.Headers["User-Agent"] != r2.Headers["User-Agent"] {
		t.Errorf("same domain: got %q and %q, want identical agents",
			r1.Headers["User-Agent"], r2.Headers["User-Agent"])
	}
}

func TestUserAgent_PerDomain_DifferentDomains(t *testing.T) {
	// Use a large pool to reduce the probability of a collision.
	pool := make([]string, 16)
	for i := range pool {
		pool[i] = fmt.Sprintf("UA-%d", i)
	}
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   pool,
		RotationMode: RotationPerDomain,
	})

	domains := []string{
		"http://alpha.com",
		"http://beta.com",
		"http://gamma.com",
	}

	agents := make(map[string]string, len(domains))
	for _, url := range domains {
		r := &Request{URL: url}
		if _, err := ua.ProcessRequest(context.Background(), r, nopHandler); err != nil {
			t.Fatal(err)
		}
		agents[url] = r.Headers["User-Agent"]
	}

	// Verify each domain returns the same agent on repeated calls.
	for _, url := range domains {
		r := &Request{URL: url}
		if _, err := ua.ProcessRequest(context.Background(), r, nopHandler); err != nil {
			t.Fatal(err)
		}
		if r.Headers["User-Agent"] != agents[url] {
			t.Errorf("domain %s: second call returned %q, want %q",
				url, r.Headers["User-Agent"], agents[url])
		}
	}
}

func TestUserAgent_PerDomain_UnparsableURLFallsBackToRoundRobin(t *testing.T) {
	pool := []string{"UA-A", "UA-B", "UA-C"}
	ua := NewUserAgent(UserAgentConfig{
		UserAgents:   pool,
		RotationMode: RotationPerDomain,
	})

	// An unparseable or empty URL should fall through to round-robin.
	for i := 0; i < len(pool); i++ {
		r := &Request{URL: "://invalid-url"}
		if _, err := ua.ProcessRequest(context.Background(), r, nopHandler); err != nil {
			t.Fatal(err)
		}
		want := pool[i%len(pool)]
		if r.Headers["User-Agent"] != want {
			t.Errorf("request %d: User-Agent = %q, want %q", i, r.Headers["User-Agent"], want)
		}
	}
}
