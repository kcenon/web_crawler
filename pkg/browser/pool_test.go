package browser_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kcenon/web_crawler/pkg/browser"
)

// newTestPool creates a small pool suitable for unit tests.
// Tests that actually launch Chrome are skipped in short mode.
func newTestPool(t *testing.T, max int) *browser.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping browser pool test in short mode (requires Chrome)")
	}
	p, err := browser.NewPool(browser.Config{
		MaxBrowsers:         max,
		MaxTabsPerBrowser:   5,
		Headless:            true,
		HealthCheckInterval: 24 * time.Hour, // disable background health checks
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	t.Cleanup(p.Close)
	return p
}

func TestPool_AcquireRelease(t *testing.T) {
	p := newTestPool(t, 2)

	b, release, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil browser")
	}
	release()
}

func TestPool_ConcurrentAcquire(t *testing.T) {
	const max = 3
	p := newTestPool(t, max)

	var wg sync.WaitGroup
	for i := 0; i < max; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, release, err := p.Acquire(context.Background())
			if err != nil {
				t.Errorf("Acquire: %v", err)
				return
			}
			if b == nil {
				t.Errorf("got nil browser")
			}
			time.Sleep(50 * time.Millisecond)
			release()
		}()
	}
	wg.Wait()
}

func TestPool_AcquireRespectsContext(t *testing.T) {
	// Pool with 1 browser; hold it, then try to acquire with a cancelled context.
	p := newTestPool(t, 1)

	_, release, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, _, err = p.Acquire(ctx)
	if err == nil {
		t.Fatal("expected error when context expires, got nil")
	}
}

func TestPool_Close(t *testing.T) {
	p := newTestPool(t, 1)
	p.Close()
	// Second close must not panic.
	p.Close()
}

func TestPool_BrowserExhaustion(t *testing.T) {
	// MaxTabsPerBrowser=2: after 2 tabs the browser should be replaced.
	if testing.Short() {
		t.Skip("skipping browser exhaustion test in short mode")
	}
	p, err := browser.NewPool(browser.Config{
		MaxBrowsers:         1,
		MaxTabsPerBrowser:   2,
		Headless:            true,
		HealthCheckInterval: 24 * time.Hour,
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	defer p.Close()

	for i := 0; i < 3; i++ {
		b, release, err := p.Acquire(context.Background())
		if err != nil {
			t.Fatalf("Acquire iteration %d: %v", i, err)
		}
		// Open a tab to increment usage.
		_, cancelTab, err := b.NewTab(context.Background())
		if err != nil {
			release()
			t.Fatalf("NewTab iteration %d: %v", i, err)
		}
		cancelTab()
		release()
	}
}

func TestBuilder(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping builder test in short mode")
	}
	p, err := browser.NewBuilder().
		WithMaxBrowsers(1).
		WithMaxTabsPerBrowser(10).
		WithHeadless(true).
		WithHealthCheckInterval(24 * time.Hour).
		Build()
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	defer p.Close()

	b, release, err := p.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer release()

	if err := b.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}
