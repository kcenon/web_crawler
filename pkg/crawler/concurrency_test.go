package crawler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewConcurrencyManager_Defaults(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{})

	if cm.cfg.GlobalMaxConcurrency != 100 {
		t.Errorf("GlobalMaxConcurrency = %d, want 100", cm.cfg.GlobalMaxConcurrency)
	}
	if cm.cfg.PerDomainMaxConcurrency != 5 {
		t.Errorf("PerDomainMaxConcurrency = %d, want 5", cm.cfg.PerDomainMaxConcurrency)
	}
}

func TestAcquireRelease_Basic(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    2,
		PerDomainMaxConcurrency: 1,
	})

	ctx := context.Background()

	if err := cm.Acquire(ctx, "example.com"); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	cm.Release("example.com")
}

func TestGlobalLimitEnforcement(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    2,
		PerDomainMaxConcurrency: 10,
	})

	ctx := context.Background()

	// Acquire all global slots using different domains.
	if err := cm.Acquire(ctx, "a.com"); err != nil {
		t.Fatalf("Acquire(a.com) error = %v", err)
	}
	if err := cm.Acquire(ctx, "b.com"); err != nil {
		t.Fatalf("Acquire(b.com) error = %v", err)
	}

	// Third acquire should block; use a short timeout to verify.
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err := cm.Acquire(timeoutCtx, "c.com")
	if err == nil {
		t.Fatal("Acquire should block when global limit is reached")
	}

	// Release one slot, then acquire should succeed.
	cm.Release("a.com")

	if err := cm.Acquire(ctx, "c.com"); err != nil {
		t.Fatalf("Acquire(c.com) after release error = %v", err)
	}

	cm.Release("b.com")
	cm.Release("c.com")
}

func TestPerDomainLimitEnforcement(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    100,
		PerDomainMaxConcurrency: 2,
	})

	ctx := context.Background()

	// Acquire per-domain limit.
	if err := cm.Acquire(ctx, "example.com"); err != nil {
		t.Fatalf("Acquire #1 error = %v", err)
	}
	if err := cm.Acquire(ctx, "example.com"); err != nil {
		t.Fatalf("Acquire #2 error = %v", err)
	}

	// Third acquire for same domain should block.
	timeoutCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	err := cm.Acquire(timeoutCtx, "example.com")
	if err == nil {
		t.Fatal("Acquire should block when per-domain limit is reached")
	}

	// Different domain should still work.
	if err := cm.Acquire(ctx, "other.com"); err != nil {
		t.Fatalf("Acquire(other.com) error = %v", err)
	}

	cm.Release("example.com")
	cm.Release("example.com")
	cm.Release("other.com")
}

func TestDisablePerDomainLimits(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    100,
		PerDomainMaxConcurrency: 1,
		DisablePerDomainLimits:  true,
	})

	ctx := context.Background()

	// Should be able to acquire multiple times for same domain since
	// per-domain limits are disabled.
	for i := 0; i < 5; i++ {
		if err := cm.Acquire(ctx, "example.com"); err != nil {
			t.Fatalf("Acquire #%d error = %v", i, err)
		}
	}

	for i := 0; i < 5; i++ {
		cm.Release("example.com")
	}
}

func TestContextCancellation(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    1,
		PerDomainMaxConcurrency: 1,
	})

	ctx := context.Background()

	// Exhaust global limit.
	if err := cm.Acquire(ctx, "example.com"); err != nil {
		t.Fatalf("Acquire error = %v", err)
	}

	// Cancel context immediately.
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	err := cm.Acquire(cancelCtx, "other.com")
	if err == nil {
		t.Fatal("Acquire with cancelled context should return error")
	}

	cm.Release("example.com")
}

func TestContextCancellation_DomainSem(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    100,
		PerDomainMaxConcurrency: 1,
	})

	ctx := context.Background()

	// Exhaust per-domain limit.
	if err := cm.Acquire(ctx, "example.com"); err != nil {
		t.Fatalf("Acquire error = %v", err)
	}

	// Cancel context while waiting for domain semaphore.
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	err := cm.Acquire(cancelCtx, "example.com")
	if err == nil {
		t.Fatal("Acquire with cancelled context should return error")
	}

	cm.Release("example.com")
}

func TestConcurrentAccess(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    10,
		PerDomainMaxConcurrency: 3,
	})

	ctx := context.Background()
	domains := []string{"a.com", "b.com", "c.com"}
	var active atomic.Int32
	var maxActive atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(domain string) {
			defer wg.Done()

			if err := cm.Acquire(ctx, domain); err != nil {
				t.Errorf("Acquire(%s) error = %v", domain, err)
				return
			}

			cur := active.Add(1)
			// Track max concurrent.
			for {
				old := maxActive.Load()
				if cur <= old || maxActive.CompareAndSwap(old, cur) {
					break
				}
			}

			// Simulate work.
			time.Sleep(time.Millisecond)

			active.Add(-1)
			cm.Release(domain)
		}(domains[i%len(domains)])
	}

	wg.Wait()

	if got := maxActive.Load(); got > 10 {
		t.Errorf("max concurrent = %d, should not exceed global limit 10", got)
	}
}

func TestLazyDomainSemaphoreCreation(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    100,
		PerDomainMaxConcurrency: 5,
	})

	// No domain semaphores should exist initially.
	cm.mu.RLock()
	initialCount := len(cm.domainSems)
	cm.mu.RUnlock()
	if initialCount != 0 {
		t.Errorf("initial domainSems count = %d, want 0", initialCount)
	}

	ctx := context.Background()
	if err := cm.Acquire(ctx, "new.com"); err != nil {
		t.Fatalf("Acquire error = %v", err)
	}

	// Domain semaphore should now exist.
	cm.mu.RLock()
	afterCount := len(cm.domainSems)
	cm.mu.RUnlock()
	if afterCount != 1 {
		t.Errorf("domainSems count after Acquire = %d, want 1", afterCount)
	}

	cm.Release("new.com")
}

func TestReleaseUnknownDomain(t *testing.T) {
	cm := NewConcurrencyManager(ConcurrencyConfig{
		GlobalMaxConcurrency:    10,
		PerDomainMaxConcurrency: 5,
	})

	ctx := context.Background()
	if err := cm.Acquire(ctx, "known.com"); err != nil {
		t.Fatalf("Acquire error = %v", err)
	}

	// Release with the correct domain should not panic.
	cm.Release("known.com")
}
