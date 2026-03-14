package browser

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
	"sync/atomic"
)

// Browser wraps a single chromedp browser instance with lifecycle tracking.
//
// Each Browser tracks how many tabs it has served. Once tabsUsed reaches
// maxTabs, the browser is considered exhausted and will be replaced by the
// pool on the next release.
type Browser struct {
	ctx    context.Context
	cancel context.CancelFunc

	tabsUsed atomic.Int64
	maxTabs  int64

	// closed is set to true when the browser context has been cancelled.
	closed atomic.Bool
}

// newBrowser launches a new Chrome process using the given allocator context
// and returns a Browser wrapper.
func newBrowser(allocCtx context.Context, maxTabs int) (*Browser, error) {
	ctx, cancel := chromedp.NewContext(allocCtx)

	// Ensure the browser process is started.
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		return nil, fmt.Errorf("start browser: %w", err)
	}

	return &Browser{
		ctx:     ctx,
		cancel:  cancel,
		maxTabs: int64(maxTabs),
	}, nil
}

// NewTab creates a new tab context within this browser.
// The caller must cancel the returned context when the tab is no longer needed.
func (b *Browser) NewTab(ctx context.Context) (context.Context, context.CancelFunc, error) {
	if b.closed.Load() {
		return nil, nil, fmt.Errorf("browser is closed")
	}
	b.tabsUsed.Add(1)
	// Derive the tab from the browser context (not the caller ctx) so it lives
	// within this browser process. Use chromedp.WithNewBrowserContext to open a
	// proper tab, not a nested browser.
	tabCtx, cancel := chromedp.NewContext(b.ctx)
	return tabCtx, cancel, nil
}

// Exhausted reports whether this browser has reached its tab usage limit.
func (b *Browser) Exhausted() bool {
	return b.tabsUsed.Load() >= b.maxTabs
}

// Ping executes a trivial JavaScript expression to verify the browser is alive.
func (b *Browser) Ping(ctx context.Context) error {
	if b.closed.Load() {
		return fmt.Errorf("browser is closed")
	}
	tabCtx, cancel := chromedp.NewContext(b.ctx)
	defer cancel()
	var result string
	return chromedp.Run(tabCtx,
		chromedp.Evaluate(`"ok"`, &result),
	)
}

// Close terminates the browser process.
func (b *Browser) Close() {
	if b.closed.Swap(true) {
		return // already closed
	}
	b.cancel()
}
