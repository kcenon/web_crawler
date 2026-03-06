package crawler

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// ConcurrencyConfig holds configuration for the ConcurrencyManager.
type ConcurrencyConfig struct {
	// GlobalMaxConcurrency is the maximum number of concurrent requests
	// across all domains. Default: 100.
	GlobalMaxConcurrency int64

	// PerDomainMaxConcurrency is the maximum number of concurrent requests
	// per individual domain. Default: 5.
	PerDomainMaxConcurrency int64

	// DisablePerDomainLimits, if true, disables per-domain concurrency
	// limits. Only the global limit will be enforced. Per-domain limits
	// are enabled by default.
	DisablePerDomainLimits bool
}

func (c ConcurrencyConfig) withDefaults() ConcurrencyConfig {
	if c.GlobalMaxConcurrency == 0 {
		c.GlobalMaxConcurrency = 100
	}
	if c.PerDomainMaxConcurrency == 0 {
		c.PerDomainMaxConcurrency = 5
	}
	return c
}

// ConcurrencyManager controls global and per-domain concurrent request
// limits using weighted semaphores. It ensures that the crawler does not
// overwhelm target servers or exhaust local resources.
type ConcurrencyManager struct {
	globalSem  *semaphore.Weighted
	domainSems map[string]*semaphore.Weighted
	mu         sync.RWMutex
	cfg        ConcurrencyConfig
}

// NewConcurrencyManager creates a ConcurrencyManager with the given config.
func NewConcurrencyManager(cfg ConcurrencyConfig) *ConcurrencyManager {
	cfg = cfg.withDefaults()
	return &ConcurrencyManager{
		globalSem:  semaphore.NewWeighted(cfg.GlobalMaxConcurrency),
		domainSems: make(map[string]*semaphore.Weighted),
		cfg:        cfg,
	}
}

// Acquire obtains permission to make a request to the given domain. It
// blocks until both the global and per-domain semaphores are acquired, or
// the context is cancelled.
func (cm *ConcurrencyManager) Acquire(ctx context.Context, domain string) error {
	if err := cm.globalSem.Acquire(ctx, 1); err != nil {
		return err
	}

	if cm.cfg.DisablePerDomainLimits {
		return nil
	}

	domainSem := cm.getOrCreateDomainSem(domain)
	if err := domainSem.Acquire(ctx, 1); err != nil {
		cm.globalSem.Release(1)
		return err
	}

	return nil
}

// Release returns the semaphore permits for the given domain. It must be
// called exactly once for each successful Acquire call.
func (cm *ConcurrencyManager) Release(domain string) {
	if !cm.cfg.DisablePerDomainLimits {
		cm.mu.RLock()
		domainSem, ok := cm.domainSems[domain]
		cm.mu.RUnlock()
		if ok {
			domainSem.Release(1)
		}
	}
	cm.globalSem.Release(1)
}

// getOrCreateDomainSem returns the semaphore for the given domain, creating
// one lazily if it does not yet exist. Thread-safe via double-checked locking.
func (cm *ConcurrencyManager) getOrCreateDomainSem(domain string) *semaphore.Weighted {
	cm.mu.RLock()
	sem, ok := cm.domainSems[domain]
	cm.mu.RUnlock()
	if ok {
		return sem
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Double-check after acquiring write lock.
	if sem, ok = cm.domainSems[domain]; ok {
		return sem
	}

	sem = semaphore.NewWeighted(cm.cfg.PerDomainMaxConcurrency)
	cm.domainSems[domain] = sem
	return sem
}
