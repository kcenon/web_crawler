package browser

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// WaitCondition describes how to wait for a page to become ready after
// navigation. Implementations return chromedp actions to append to the
// render task list.
type WaitCondition interface {
	actions(ctx context.Context) []chromedp.Action
}

// WaitNetworkIdle waits until there have been no in-flight network requests
// for at least QuietPeriod (default: 500ms). It is suitable for SPAs that
// issue multiple API calls during load.
type WaitNetworkIdle struct {
	// QuietPeriod is how long the network must be idle before we consider
	// the page done. Zero uses the default (500ms).
	QuietPeriod time.Duration
	// Timeout is the maximum time to wait. Zero uses the context deadline.
	Timeout time.Duration
}

func (w WaitNetworkIdle) actions(ctx context.Context) []chromedp.Action {
	quiet := w.QuietPeriod
	if quiet == 0 {
		quiet = 500 * time.Millisecond
	}
	return []chromedp.Action{chromedp.ActionFunc(func(ctx context.Context) error {
		return waitNetworkIdle(ctx, quiet, w.Timeout)
	})}
}

// waitNetworkIdle listens to CDP network events and returns once the number
// of in-flight requests has been zero for the full quiet period.
func waitNetworkIdle(ctx context.Context, quiet, maxWait time.Duration) error {
	var inFlight atomic.Int64
	lastActive := time.Now()

	// Listen to network request/response events.
	chromedp.ListenTarget(ctx, func(ev any) {
		switch ev.(type) {
		case *network.EventRequestWillBeSent:
			inFlight.Add(1)
			lastActive = time.Now()
		case *network.EventLoadingFinished,
			*network.EventLoadingFailed:
			if v := inFlight.Add(-1); v < 0 {
				inFlight.Store(0)
			}
		}
	})

	// Enable network events.
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		return err
	}

	deadline := time.Now().Add(maxWait)
	if maxWait == 0 {
		deadline = time.Now().Add(30 * time.Second)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if time.Now().After(deadline) {
			return nil // Timed out waiting — return what we have
		}
		if inFlight.Load() == 0 && time.Since(lastActive) >= quiet {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// WaitSelector waits until the given CSS selector is present in the DOM.
// This is the most reliable condition for pages that render a specific element
// when content is ready. The overall timeout is controlled by RenderRequest.Timeout.
type WaitSelector struct {
	// Selector is the CSS selector to wait for (required).
	Selector string
}

func (w WaitSelector) actions(_ context.Context) []chromedp.Action {
	// Timeout is respected via the context deadline set by Renderer.Render.
	// chromedp.WaitVisible inherits cancellation from the tab context.
	return []chromedp.Action{
		chromedp.WaitVisible(w.Selector, chromedp.ByQuery),
	}
}

// WaitDelay waits a fixed duration after navigation before reading the DOM.
// Use this only when the other strategies are not applicable.
type WaitDelay struct {
	Duration time.Duration
}

func (w WaitDelay) actions(_ context.Context) []chromedp.Action {
	d := w.Duration
	return []chromedp.Action{chromedp.ActionFunc(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
			return nil
		}
	})}
}

// WaitDOMContentLoaded waits for the DOMContentLoaded event, which fires when
// the initial HTML has been parsed (before stylesheets and images load).
// Equivalent to jQuery's $(document).ready().
type WaitDOMContentLoaded struct{}

func (w WaitDOMContentLoaded) actions(_ context.Context) []chromedp.Action {
	return []chromedp.Action{chromedp.WaitReady("body", chromedp.ByQuery)}
}
