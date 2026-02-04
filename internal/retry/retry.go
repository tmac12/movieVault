package retry

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

// Retry executes fn with exponential backoff until it succeeds or maxAttempts is reached.
// The backoff doubles after each failed attempt starting from initialBackoff.
// Non-retryable errors (like 401, 404) return immediately without retry.
func Retry(fn func() error, maxAttempts int, initialBackoff time.Duration) error {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Don't retry non-retryable errors
		if !IsRetryable(lastErr) && !IsRateLimited(lastErr) {
			return lastErr
		}

		// Don't sleep after the last attempt
		if attempt < maxAttempts {
			// Use longer backoff for rate limited errors
			sleepDuration := backoff
			if IsRateLimited(lastErr) {
				sleepDuration = backoff * 2
			}
			time.Sleep(sleepDuration)
			backoff *= 2
		}
	}

	return lastErr
}

// IsRetryable returns true if the error is a transient error that should be retried.
// This includes network timeouts and 5xx server errors.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	// Check for URL errors (connection refused, DNS errors, etc.)
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
	}

	// Check for 5xx errors in error message
	errStr := err.Error()
	if strings.Contains(errStr, "status 500") ||
		strings.Contains(errStr, "status 501") ||
		strings.Contains(errStr, "status 502") ||
		strings.Contains(errStr, "status 503") ||
		strings.Contains(errStr, "status 504") {
		return true
	}

	// Check for common transient error messages
	if strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}

	return false
}

// IsRateLimited returns true if the error indicates rate limiting (HTTP 429).
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "status 429")
}
