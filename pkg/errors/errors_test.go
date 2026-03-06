package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestNetworkError(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := NewNetworkError("https://example.com", cause)

	if err.Code != CodeNetwork {
		t.Errorf("expected code %d, got %d", CodeNetwork, err.Code)
	}
	if err.URL != "https://example.com" {
		t.Errorf("expected URL, got %q", err.URL)
	}

	// Test Error() includes cause
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}

	// Test Unwrap
	if !errors.Is(err, cause) {
		t.Error("expected errors.Is to find cause")
	}
}

func TestHTTPError(t *testing.T) {
	err := NewHTTPError("https://example.com/page", 404)

	if err.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", err.StatusCode)
	}
	if err.Code != CodeHTTP {
		t.Errorf("expected code %d, got %d", CodeHTTP, err.Code)
	}

	// Test errors.As
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Error("expected errors.As to match HTTPError")
	}
}

func TestExtractionError(t *testing.T) {
	cause := fmt.Errorf("parse error")
	err := NewExtractionError("https://example.com", "h1.title", cause)

	if err.Selector != "h1.title" {
		t.Errorf("expected selector, got %q", err.Selector)
	}
	if !errors.Is(err, cause) {
		t.Error("expected errors.Is to find cause")
	}
}

func TestConfigError(t *testing.T) {
	err := NewConfigError("max_depth", "must be positive")

	if err.Field != "max_depth" {
		t.Errorf("expected field 'max_depth', got %q", err.Field)
	}
	if err.Code != CodeConfig {
		t.Errorf("expected code %d, got %d", CodeConfig, err.Code)
	}
}

func TestRobotsError(t *testing.T) {
	err := NewRobotsError("https://example.com/secret")

	var robotsErr *RobotsError
	if !errors.As(err, &robotsErr) {
		t.Error("expected errors.As to match RobotsError")
	}
	if robotsErr.URL != "https://example.com/secret" {
		t.Errorf("expected URL, got %q", robotsErr.URL)
	}
}

func TestRateLimitError(t *testing.T) {
	err := NewRateLimitError("example.com")

	if err.Domain != "example.com" {
		t.Errorf("expected domain, got %q", err.Domain)
	}
}

func TestTimeoutError(t *testing.T) {
	cause := fmt.Errorf("context deadline exceeded")
	err := NewTimeoutError("https://example.com", cause)

	if err.URL != "https://example.com" {
		t.Errorf("expected URL, got %q", err.URL)
	}
	if !errors.Is(err, cause) {
		t.Error("expected errors.Is to find cause")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"network error", NewNetworkError("url", nil), true},
		{"timeout error", NewTimeoutError("url", nil), true},
		{"rate limit error", NewRateLimitError("domain"), true},
		{"HTTP 500", NewHTTPError("url", 500), true},
		{"HTTP 502", NewHTTPError("url", 502), true},
		{"HTTP 429", NewHTTPError("url", 429), true},
		{"HTTP 404", NewHTTPError("url", 404), false},
		{"HTTP 200", NewHTTPError("url", 200), false},
		{"config error", NewConfigError("field", "msg"), false},
		{"robots error", NewRobotsError("url"), false},
		{"extraction error", NewExtractionError("url", "sel", nil), false},
		{"plain error", fmt.Errorf("some error"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsRetryable(tc.err)
			if got != tc.expected {
				t.Errorf("IsRetryable(%v) = %v, want %v", tc.err, got, tc.expected)
			}
		})
	}
}

func TestIsTemporary(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"network error", NewNetworkError("url", nil), true},
		{"timeout error", NewTimeoutError("url", nil), true},
		{"rate limit error", NewRateLimitError("domain"), true},
		{"HTTP error", NewHTTPError("url", 500), false},
		{"config error", NewConfigError("field", "msg"), false},
		{"robots error", NewRobotsError("url"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsTemporary(tc.err)
			if got != tc.expected {
				t.Errorf("IsTemporary(%v) = %v, want %v", tc.err, got, tc.expected)
			}
		})
	}
}

func TestWrappedErrorChain(t *testing.T) {
	rootCause := fmt.Errorf("DNS resolution failed")
	netErr := NewNetworkError("https://example.com", rootCause)
	wrapped := fmt.Errorf("crawl operation: %w", netErr)

	// Should find NetworkError through wrapping
	var found *NetworkError
	if !errors.As(wrapped, &found) {
		t.Error("expected errors.As to find NetworkError in wrapped chain")
	}

	// Should find root cause
	if !errors.Is(wrapped, rootCause) {
		t.Error("expected errors.Is to find root cause in wrapped chain")
	}

	// IsRetryable should work through wrapping
	if !IsRetryable(wrapped) {
		t.Error("expected wrapped NetworkError to be retryable")
	}
}

func TestCrawlerError_NoCause(t *testing.T) {
	err := &CrawlerError{
		Code:    CodeUnknown,
		Message: "something went wrong",
	}

	if err.Error() != "something went wrong" {
		t.Errorf("unexpected message: %q", err.Error())
	}
	if err.Unwrap() != nil {
		t.Error("expected nil cause")
	}
}
