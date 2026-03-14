package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

// RenderRequest configures a single page render operation.
type RenderRequest struct {
	// URL is the page to navigate to (required).
	URL string

	// Wait defines how long to wait after navigation before reading the DOM.
	// Nil uses WaitDOMContentLoaded.
	Wait WaitCondition

	// Resources, if non-zero, installs a URL block list before navigation.
	Resources ResourceFilter

	// Screenshot, if true, captures a full-page PNG screenshot.
	Screenshot bool

	// ScreenshotQuality sets JPEG quality (1-100). Ignored when Screenshot
	// is true and the output format is PNG (quality 0 = lossless PNG).
	ScreenshotQuality int

	// Timeout is the maximum time for the entire render. Zero uses 30s.
	Timeout time.Duration

	// JavaScript is an optional JS snippet to evaluate after the page is ready.
	// The expression result is available in RenderResult.ScriptResult.
	JavaScript string
}

// RenderResult holds the output of a render operation.
type RenderResult struct {
	// HTML is the full outer HTML of the page after JS execution.
	HTML string

	// Screenshot holds raw PNG bytes when RenderRequest.Screenshot is true.
	Screenshot []byte

	// ScriptResult holds the JSON-serialised result of RenderRequest.JavaScript.
	ScriptResult string

	// URL is the final URL after any redirects.
	URL string
}

// Renderer renders pages using a browser pool.
// It is safe for concurrent use.
type Renderer struct {
	pool *Pool
}

// NewRenderer creates a Renderer backed by the given pool.
func NewRenderer(pool *Pool) *Renderer {
	return &Renderer{pool: pool}
}

// Render navigates to req.URL in a pooled browser tab, applies wait conditions
// and resource filters, then returns the page HTML and optional screenshot.
func (r *Renderer) Render(ctx context.Context, req RenderRequest) (*RenderResult, error) {
	if req.URL == "" {
		return nil, fmt.Errorf("render: URL is required")
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Acquire a browser from the pool.
	b, release, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("render: acquire browser: %w", err)
	}
	defer release()

	// Open a new tab.
	tabCtx, cancelTab, err := b.NewTab(ctx)
	if err != nil {
		return nil, fmt.Errorf("render: new tab: %w", err)
	}
	defer cancelTab()

	// Apply the render timeout to the tab context. context.WithTimeout derived
	// from a chromedp context preserves the CDP target info while adding a
	// deadline that chromedp.Run will respect.
	tabCtx, cancelTimeout := context.WithTimeout(tabCtx, timeout)
	defer cancelTimeout()

	result := &RenderResult{}

	// Build the action list.
	var actions []chromedp.Action

	// 1. Resource blocking (must be set before navigation).
	actions = append(actions, req.Resources.apply()...)

	// 2. Navigate.
	actions = append(actions, chromedp.Navigate(req.URL))

	// 3. Wait condition.
	wait := req.Wait
	if wait == nil {
		wait = WaitDOMContentLoaded{}
	}
	actions = append(actions, wait.actions(tabCtx)...)

	// 4. Capture the final URL after redirects.
	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Location(&result.URL).Do(ctx)
	}))

	// 5. Read HTML.
	actions = append(actions, chromedp.OuterHTML("html", &result.HTML, chromedp.ByQuery))

	// 6. Screenshot (full-page PNG).
	if req.Screenshot {
		actions = append(actions, chromedp.FullScreenshot(&result.Screenshot, 100))
	}

	// 7. Optional JavaScript evaluation.
	if req.JavaScript != "" {
		actions = append(actions, chromedp.Evaluate(req.JavaScript, &result.ScriptResult))
	}

	if err := chromedp.Run(tabCtx, actions...); err != nil {
		return nil, fmt.Errorf("render %s: %w", req.URL, err)
	}

	return result, nil
}
