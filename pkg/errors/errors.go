// Package errors defines the error type hierarchy for the web crawler SDK.
//
// All error types implement the standard error interface and support
// unwrapping via errors.Is() and errors.As() for programmatic handling.
package errors

import (
	"errors"
	"fmt"
)

// ErrorCode identifies error categories for programmatic handling.
// Codes are stable across versions for Python SDK mapping.
type ErrorCode int

// ErrorCode values.
const (
	CodeUnknown    ErrorCode = 0
	CodeNetwork    ErrorCode = 1
	CodeHTTP       ErrorCode = 2
	CodeExtraction ErrorCode = 3
	CodeConfig     ErrorCode = 4
	CodeRobots     ErrorCode = 5
	CodeRateLimit  ErrorCode = 6
	CodeTimeout    ErrorCode = 7
	CodeCancelled  ErrorCode = 8
)

// CrawlerError is the base error type for all crawler errors.
type CrawlerError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *CrawlerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *CrawlerError) Unwrap() error {
	return e.Cause
}

// NetworkError represents connection, DNS, or transport-level errors.
type NetworkError struct {
	CrawlerError
	URL string
}

// NewNetworkError creates a NetworkError.
func NewNetworkError(url string, cause error) *NetworkError {
	return &NetworkError{
		CrawlerError: CrawlerError{
			Code:    CodeNetwork,
			Message: fmt.Sprintf("network error for %s", url),
			Cause:   cause,
		},
		URL: url,
	}
}

// HTTPError represents an HTTP response with an error status code.
type HTTPError struct {
	CrawlerError
	StatusCode int
	URL        string
}

// NewHTTPError creates an HTTPError.
func NewHTTPError(url string, statusCode int) *HTTPError {
	return &HTTPError{
		CrawlerError: CrawlerError{
			Code:    CodeHTTP,
			Message: fmt.Sprintf("HTTP %d for %s", statusCode, url),
		},
		StatusCode: statusCode,
		URL:        url,
	}
}

// ExtractionError represents a failure during data extraction.
type ExtractionError struct {
	CrawlerError
	Selector string
	URL      string
}

// NewExtractionError creates an ExtractionError.
func NewExtractionError(url, selector string, cause error) *ExtractionError {
	return &ExtractionError{
		CrawlerError: CrawlerError{
			Code:    CodeExtraction,
			Message: fmt.Sprintf("extraction failed for selector %q on %s", selector, url),
			Cause:   cause,
		},
		Selector: selector,
		URL:      url,
	}
}

// ConfigError represents an invalid configuration.
type ConfigError struct {
	CrawlerError
	Field string
}

// NewConfigError creates a ConfigError.
func NewConfigError(field, message string) *ConfigError {
	return &ConfigError{
		CrawlerError: CrawlerError{
			Code:    CodeConfig,
			Message: fmt.Sprintf("config error: %s: %s", field, message),
		},
		Field: field,
	}
}

// RobotsError represents a URL blocked by robots.txt.
type RobotsError struct {
	CrawlerError
	URL string
}

// NewRobotsError creates a RobotsError.
func NewRobotsError(url string) *RobotsError {
	return &RobotsError{
		CrawlerError: CrawlerError{
			Code:    CodeRobots,
			Message: fmt.Sprintf("blocked by robots.txt: %s", url),
		},
		URL: url,
	}
}

// RateLimitError represents a rate limit being exceeded.
type RateLimitError struct {
	CrawlerError
	Domain string
}

// NewRateLimitError creates a RateLimitError.
func NewRateLimitError(domain string) *RateLimitError {
	return &RateLimitError{
		CrawlerError: CrawlerError{
			Code:    CodeRateLimit,
			Message: fmt.Sprintf("rate limit exceeded for %s", domain),
		},
		Domain: domain,
	}
}

// TimeoutError represents a request or operation timeout.
type TimeoutError struct {
	CrawlerError
	URL string
}

// NewTimeoutError creates a TimeoutError.
func NewTimeoutError(url string, cause error) *TimeoutError {
	return &TimeoutError{
		CrawlerError: CrawlerError{
			Code:    CodeTimeout,
			Message: fmt.Sprintf("timeout for %s", url),
			Cause:   cause,
		},
		URL: url,
	}
}

// IsRetryable returns true if the error represents a transient condition
// that may succeed on retry. Network errors, timeouts, rate limits,
// and 5xx HTTP errors are considered retryable.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return true
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode >= 500 || httpErr.StatusCode == 429
	}

	return false
}

// IsTemporary returns true if the error is likely temporary and the
// operation may succeed later without changes.
func IsTemporary(err error) bool {
	if err == nil {
		return false
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return true
	}

	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return true
	}

	var rateLimitErr *RateLimitError
	return errors.As(err, &rateLimitErr)
}
