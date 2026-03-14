package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/sync/semaphore"
)

// Pool manages a bounded set of Chrome browser instances for concurrent
// JavaScript rendering. It prevents resource exhaustion by limiting the
// number of live browsers and restarting them after heavy use.
//
// Usage:
//
//	p, _ := NewPool(Config{MaxBrowsers: 3})
//	defer p.Close()
//
//	b, release, _ := p.Acquire(ctx)
//	defer release()
//	tabCtx, cancelTab, _ := b.NewTab(ctx)
//	defer cancelTab()
//	// use tabCtx with chromedp actions
type Pool struct {
	cfg      Config
	allocCtx context.Context
	allocCxl context.CancelFunc

	sem      *semaphore.Weighted
	mu       sync.Mutex
	browsers []*Browser

	healthTicker *time.Ticker
	stopHealth   chan struct{}
	closed       bool
}

// NewPool creates a new browser pool with the given configuration.
// Callers must call Close when done to free all browser resources.
func NewPool(cfg Config) (*Pool, error) {
	cfg = cfg.withDefaults()

	opts := chromedp.DefaultExecAllocatorOptions[:]
	if !cfg.Headless {
		// Override the headless flag explicitly to run in headful mode.
		opts = append(opts, chromedp.Flag("headless", false))
	}
	if cfg.UserAgent != "" {
		opts = append(opts, chromedp.UserAgent(cfg.UserAgent))
	}
	for _, flag := range cfg.ChromeFlags {
		opts = append(opts, chromedp.Flag(flag, true))
	}

	allocCtx, allocCxl := chromedp.NewExecAllocator(context.Background(), opts...)

	p := &Pool{
		cfg:        cfg,
		allocCtx:   allocCtx,
		allocCxl:   allocCxl,
		sem:        semaphore.NewWeighted(int64(cfg.MaxBrowsers)),
		browsers:   make([]*Browser, 0, cfg.MaxBrowsers),
		stopHealth: make(chan struct{}),
	}

	// Pre-warm browsers up to MaxBrowsers.
	for i := 0; i < cfg.MaxBrowsers; i++ {
		b, err := newBrowser(allocCtx, cfg.MaxTabsPerBrowser)
		if err != nil {
			p.closeAll()
			allocCxl()
			return nil, fmt.Errorf("pre-warm browser %d: %w", i, err)
		}
		p.browsers = append(p.browsers, b)
	}

	p.healthTicker = time.NewTicker(cfg.HealthCheckInterval)
	go p.healthLoop()

	return p, nil
}

// Acquire blocks until a healthy browser is available, then returns it along
// with a release function. The caller MUST call release when done; otherwise
// the pool semaphore will deadlock.
func (p *Pool) Acquire(ctx context.Context) (*Browser, func(), error) {
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return nil, nil, fmt.Errorf("acquire browser slot: %w", err)
	}

	b, err := p.pick()
	if err != nil {
		p.sem.Release(1)
		return nil, nil, err
	}

	release := func() {
		p.mu.Lock()
		if b.Exhausted() || b.closed.Load() {
			b.Close()
			replacement, replErr := newBrowser(p.allocCtx, p.cfg.MaxTabsPerBrowser)
			if replErr == nil {
				p.browsers = append(p.browsers, replacement)
			}
		} else {
			p.browsers = append(p.browsers, b)
		}
		p.mu.Unlock()
		p.sem.Release(1)
	}

	return b, release, nil
}

// pick removes and returns a browser from the idle list.
// Must be called with the semaphore already acquired.
func (p *Pool) pick() (*Browser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.browsers) == 0 {
		// No idle browser: launch a fresh one.
		b, err := newBrowser(p.allocCtx, p.cfg.MaxTabsPerBrowser)
		if err != nil {
			return nil, fmt.Errorf("launch browser: %w", err)
		}
		return b, nil
	}

	b := p.browsers[len(p.browsers)-1]
	p.browsers = p.browsers[:len(p.browsers)-1]
	return b, nil
}

// healthLoop periodically pings idle browsers and replaces any that are dead.
func (p *Pool) healthLoop() {
	for {
		select {
		case <-p.stopHealth:
			return
		case <-p.healthTicker.C:
			p.checkHealth()
		}
	}
}

func (p *Pool) checkHealth() {
	p.mu.Lock()
	live := make([]*Browser, 0, len(p.browsers))
	dead := make([]*Browser, 0)
	for _, b := range p.browsers {
		if b.closed.Load() || b.Exhausted() {
			dead = append(dead, b)
		} else {
			live = append(live, b)
		}
	}
	p.mu.Unlock()

	// Ping live browsers outside the lock.
	var healthy []*Browser
	for _, b := range live {
		ctx, cancel := context.WithTimeout(context.Background(), p.cfg.HealthCheckTimeout)
		err := b.Ping(ctx)
		cancel()
		if err != nil {
			dead = append(dead, b)
		} else {
			healthy = append(healthy, b)
		}
	}

	// Close dead browsers and spawn replacements.
	for _, b := range dead {
		b.Close()
	}

	p.mu.Lock()
	p.browsers = append(healthy, p.browsers...)
	needed := len(dead)
	p.mu.Unlock()

	for i := 0; i < needed; i++ {
		b, err := newBrowser(p.allocCtx, p.cfg.MaxTabsPerBrowser)
		if err != nil {
			continue
		}
		p.mu.Lock()
		p.browsers = append(p.browsers, b)
		p.mu.Unlock()
	}
}

// Close shuts down the pool and terminates all browser processes.
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	p.healthTicker.Stop()
	close(p.stopHealth)

	p.closeAll()
	p.allocCxl()
}

func (p *Pool) closeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, b := range p.browsers {
		b.Close()
	}
	p.browsers = nil
}
