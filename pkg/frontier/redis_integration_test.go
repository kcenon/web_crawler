//go:build integration

package frontier

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startRedisContainer starts a Redis 7 container for integration tests
// and returns a cleanup function that terminates it.
func startRedisContainer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start Redis container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate Redis container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("get mapped port: %v", err)
	}

	return fmt.Sprintf("%s:%s", host, port.Port())
}

// newIntegrationRedisFrontier creates a RedisFrontier backed by a testcontainer.
func newIntegrationRedisFrontier(t *testing.T) *RedisFrontier {
	t.Helper()
	addr := startRedisContainer(t)
	cfg := RedisConfig{
		Addr:      addr,
		KeyPrefix: fmt.Sprintf("integration:%d", time.Now().UnixNano()),
	}
	f, err := NewRedisFrontier(cfg)
	if err != nil {
		t.Fatalf("NewRedisFrontier: %v", err)
	}
	t.Cleanup(func() {
		f.client.Del(context.Background(), f.key)
		f.Close()
	})
	return f
}

func TestRedisFrontier_Integration_PushPop(t *testing.T) {
	f := newIntegrationRedisFrontier(t)
	ctx := context.Background()

	entry := &URLEntry{URL: "https://integration.example.com/page", Priority: PriorityNormal}
	if err := f.Add(entry); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if f.Size() != 1 {
		t.Errorf("Size() = %d, want 1", f.Size())
	}

	got, err := f.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if got.URL != entry.URL {
		t.Errorf("URL = %q, want %q", got.URL, entry.URL)
	}
	if f.Size() != 0 {
		t.Errorf("Size() after Next = %d, want 0", f.Size())
	}
}

func TestRedisFrontier_Integration_PriorityOrdering(t *testing.T) {
	f := newIntegrationRedisFrontier(t)
	ctx := context.Background()

	now := time.Now()
	entries := []*URLEntry{
		{URL: "https://example.com/low", Priority: PriorityLow, DiscoveredAt: now},
		{URL: "https://example.com/critical", Priority: PriorityCritical, DiscoveredAt: now},
		{URL: "https://example.com/high", Priority: PriorityHigh, DiscoveredAt: now},
	}
	for _, e := range entries {
		if err := f.Add(e); err != nil {
			t.Fatalf("Add(%q): %v", e.URL, err)
		}
	}

	e1, _ := f.Next(ctx)
	if e1.Priority != PriorityCritical {
		t.Errorf("first = %d, want PriorityCritical(%d)", e1.Priority, PriorityCritical)
	}
	e2, _ := f.Next(ctx)
	if e2.Priority != PriorityHigh {
		t.Errorf("second = %d, want PriorityHigh(%d)", e2.Priority, PriorityHigh)
	}
	e3, _ := f.Next(ctx)
	if e3.Priority != PriorityLow {
		t.Errorf("third = %d, want PriorityLow(%d)", e3.Priority, PriorityLow)
	}
}

func TestRedisFrontier_Integration_FIFOWithinPriority(t *testing.T) {
	f := newIntegrationRedisFrontier(t)
	ctx := context.Background()

	t1 := time.Now()
	t2 := t1.Add(time.Millisecond)
	t3 := t1.Add(2 * time.Millisecond)

	urls := []string{
		"https://a.integration.example.com/",
		"https://b.integration.example.com/",
		"https://c.integration.example.com/",
	}

	// Add in reverse order; FIFO should return them in original order.
	for i := len(urls) - 1; i >= 0; i-- {
		times := []time.Time{t1, t2, t3}
		if err := f.Add(&URLEntry{
			URL:          urls[i],
			Priority:     PriorityNormal,
			DiscoveredAt: times[i],
		}); err != nil {
			t.Fatalf("Add(%q): %v", urls[i], err)
		}
	}

	for _, want := range urls {
		got, err := f.Next(ctx)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if got.URL != want {
			t.Errorf("URL = %q, want %q", got.URL, want)
		}
	}
}

func TestRedisDeduplicator_Integration_MarkAndIsSeen(t *testing.T) {
	addr := startRedisContainer(t)
	d, err := NewRedisDeduplicator(RedisConfig{
		Addr:      addr,
		KeyPrefix: fmt.Sprintf("integration-dedup:%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("NewRedisDeduplicator: %v", err)
	}
	t.Cleanup(func() {
		d.client.Del(context.Background(), d.key)
		d.Close()
	})

	url := "https://integration.example.com/seen"
	if d.IsSeen(url) {
		t.Fatal("IsSeen should be false before MarkSeen")
	}
	if !d.MarkSeen(url) {
		t.Fatal("first MarkSeen should return true")
	}
	if !d.IsSeen(url) {
		t.Fatal("IsSeen should be true after MarkSeen")
	}
	if d.MarkSeen(url) {
		t.Fatal("second MarkSeen should return false (duplicate)")
	}
}
