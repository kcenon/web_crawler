package browser

import "time"

// Config holds configuration for the browser pool.
type Config struct {
	// MaxBrowsers is the maximum number of concurrent browser instances. Default: 3.
	MaxBrowsers int

	// MaxTabsPerBrowser is the number of tab uses before a browser is restarted
	// to prevent memory leaks. Default: 50.
	MaxTabsPerBrowser int

	// Headless controls whether Chrome runs in headless mode. Default: true.
	Headless bool

	// UserAgent sets the default user agent string for all browser sessions.
	// If empty, Chrome's default is used.
	UserAgent string

	// ChromeFlags holds additional Chrome command-line flags.
	ChromeFlags []string

	// HealthCheckInterval is how often the pool health-checks idle browsers. Default: 30s.
	HealthCheckInterval time.Duration

	// HealthCheckTimeout is the deadline for a single health-check ping. Default: 5s.
	HealthCheckTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.MaxBrowsers <= 0 {
		c.MaxBrowsers = 3
	}
	if c.MaxTabsPerBrowser <= 0 {
		c.MaxTabsPerBrowser = 50
	}
	if c.HealthCheckInterval <= 0 {
		c.HealthCheckInterval = 30 * time.Second
	}
	if c.HealthCheckTimeout <= 0 {
		c.HealthCheckTimeout = 5 * time.Second
	}
	return c
}
