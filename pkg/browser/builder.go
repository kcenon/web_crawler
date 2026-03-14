package browser

import "time"

// Builder provides a fluent API for constructing a Pool.
type Builder struct {
	cfg Config
}

// NewBuilder returns a Builder with default configuration.
func NewBuilder() *Builder {
	return &Builder{}
}

// WithMaxBrowsers sets the maximum number of concurrent browser processes.
func (b *Builder) WithMaxBrowsers(n int) *Builder {
	b.cfg.MaxBrowsers = n
	return b
}

// WithMaxTabsPerBrowser sets the tab-usage limit before a browser is restarted.
func (b *Builder) WithMaxTabsPerBrowser(n int) *Builder {
	b.cfg.MaxTabsPerBrowser = n
	return b
}

// WithHeadless controls headless mode (default: true).
func (b *Builder) WithHeadless(headless bool) *Builder {
	b.cfg.Headless = headless
	return b
}

// WithUserAgent sets the User-Agent string for browser sessions.
func (b *Builder) WithUserAgent(ua string) *Builder {
	b.cfg.UserAgent = ua
	return b
}

// WithChromeFlag appends an extra Chrome command-line flag.
func (b *Builder) WithChromeFlag(flag string) *Builder {
	b.cfg.ChromeFlags = append(b.cfg.ChromeFlags, flag)
	return b
}

// WithHealthCheckInterval sets the interval between health checks.
func (b *Builder) WithHealthCheckInterval(d time.Duration) *Builder {
	b.cfg.HealthCheckInterval = d
	return b
}

// Build creates the Pool from the current builder configuration.
func (b *Builder) Build() (*Pool, error) {
	return NewPool(b.cfg)
}
