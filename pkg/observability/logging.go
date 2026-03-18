package observability

import (
	"io"
	"log/slog"
	"os"
)

// LogConfig configures the structured logger.
type LogConfig struct {
	// Level sets the minimum log level. Default: INFO.
	Level slog.Level

	// Format selects the log output format: "json" or "text". Default: "json".
	Format string

	// Output sets the log output destination. Default: os.Stdout.
	Output io.Writer

	// AddSource includes source file information in log entries.
	AddSource bool

	// Sanitize enables automatic redaction of sensitive data (credentials,
	// tokens, API keys) in log output. Recommended for production use.
	Sanitize bool
}

// NewLogger creates a structured slog.Logger from the given configuration.
func NewLogger(cfg LogConfig) *slog.Logger {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(cfg.Output, opts)
	} else {
		handler = slog.NewJSONHandler(cfg.Output, opts)
	}

	if cfg.Sanitize {
		handler = NewSanitizingHandler(handler)
	}

	return slog.New(handler)
}

// RequestLogFields returns common slog attributes for a crawl request.
func RequestLogFields(requestID, domain, url string) []slog.Attr {
	return []slog.Attr{
		slog.String("request_id", requestID),
		slog.String("domain", domain),
		slog.String("url", url),
	}
}
