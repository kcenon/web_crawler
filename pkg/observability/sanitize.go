package observability

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

// redacted is the placeholder string that replaces sensitive data.
const redacted = "[REDACTED]"

// sensitiveKeys lists attribute key substrings that trigger value redaction.
// If any key contains one of these substrings (case-insensitive), its value
// is replaced with [REDACTED].
var sensitiveKeys = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"api_key",
	"apikey",
	"api-key",
	"authorization",
	"credential",
	"private_key",
	"private-key",
	"access_key",
	"access-key",
}

// sensitivePatterns are compiled regexes that redact inline secrets found in
// arbitrary string values (log messages, URLs, headers, etc.).
var sensitivePatterns = []*regexp.Regexp{
	// Bearer tokens: "Bearer <token>"
	regexp.MustCompile(`(?i)(Bearer\s+)\S+`),
	// Basic auth headers: "Basic <base64>"
	regexp.MustCompile(`(?i)(Basic\s+)\S+`),
	// URL userinfo: "scheme://user:pass@host" → "scheme://[REDACTED]@host"
	regexp.MustCompile(`://[^@/\s]+@`),
}

// sensitiveReplacements maps each pattern to its replacement string.
var sensitiveReplacements = []string{
	"${1}" + redacted,
	"${1}" + redacted,
	"://" + redacted + "@",
}

// SanitizingHandler wraps an slog.Handler to redact sensitive data from log
// attributes and messages before they reach the underlying handler.
type SanitizingHandler struct {
	inner slog.Handler
}

// NewSanitizingHandler wraps the given handler with credential redaction.
func NewSanitizingHandler(inner slog.Handler) *SanitizingHandler {
	return &SanitizingHandler{inner: inner}
}

// Enabled delegates to the inner handler.
func (h *SanitizingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle sanitizes the record's message and attributes, then delegates to
// the inner handler.
func (h *SanitizingHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Message = sanitizeString(r.Message)

	sanitized := make([]slog.Attr, 0, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		sanitized = append(sanitized, sanitizeAttr(a))
		return true
	})

	// Build a new record with sanitized attributes.
	clean := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	clean.AddAttrs(sanitized...)

	return h.inner.Handle(ctx, clean)
}

// WithAttrs sanitizes attrs before passing them to the inner handler.
func (h *SanitizingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	sanitized := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		sanitized[i] = sanitizeAttr(a)
	}
	return &SanitizingHandler{inner: h.inner.WithAttrs(sanitized)}
}

// WithGroup delegates to the inner handler.
func (h *SanitizingHandler) WithGroup(name string) slog.Handler {
	return &SanitizingHandler{inner: h.inner.WithGroup(name)}
}

// sanitizeAttr redacts the value of an attribute if its key matches a
// sensitive key pattern, or if its string value contains inline secrets.
func sanitizeAttr(a slog.Attr) slog.Attr {
	// Check if the key itself indicates a secret.
	keyLower := strings.ToLower(a.Key)
	for _, s := range sensitiveKeys {
		if strings.Contains(keyLower, s) {
			return slog.String(a.Key, redacted)
		}
	}

	// For group attributes, recurse into sub-attributes.
	if a.Value.Kind() == slog.KindGroup {
		attrs := a.Value.Group()
		sanitized := make([]slog.Attr, len(attrs))
		for i, sub := range attrs {
			sanitized[i] = sanitizeAttr(sub)
		}
		return slog.Group(a.Key, groupToAny(sanitized)...)
	}

	// For string values, apply pattern-based redaction.
	if a.Value.Kind() == slog.KindString {
		a.Value = slog.StringValue(sanitizeString(a.Value.String()))
	}

	return a
}

// sanitizeString applies all sensitive patterns to a string value.
func sanitizeString(s string) string {
	for i, re := range sensitivePatterns {
		s = re.ReplaceAllString(s, sensitiveReplacements[i])
	}
	return s
}

// groupToAny converts a slice of slog.Attr to []any for slog.Group.
func groupToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, a := range attrs {
		result[i] = a
	}
	return result
}
