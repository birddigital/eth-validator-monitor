package collector

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// RetryConfig configures the retry behavior
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultRetryConfig returns sensible defaults for beacon API calls
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		BackoffFactor:  2.0,
	}
}

// RetryableHTTPClient wraps an HTTP client with retry logic and exponential backoff
type RetryableHTTPClient struct {
	client *http.Client
	config RetryConfig
}

// NewRetryableHTTPClient creates a new HTTP client with retry capabilities
func NewRetryableHTTPClient(timeout time.Duration, config RetryConfig) *RetryableHTTPClient {
	return &RetryableHTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		config: config,
	}
}

// Do executes an HTTP request with retry logic and exponential backoff
func (r *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error
	backoff := r.config.InitialBackoff

	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		// Clone the request for retry attempts (body may have been consumed)
		reqClone := req.Clone(req.Context())

		resp, err := r.client.Do(reqClone)

		// Success - return immediately
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		// Capture error for potential retry
		if err != nil {
			lastErr = fmt.Errorf("attempt %d failed: %w", attempt+1, err)
		} else {
			// Server error - read body for context then close
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("attempt %d failed with status %d: %s", attempt+1, resp.StatusCode, string(body))
		}

		// Don't sleep after last attempt
		if attempt < r.config.MaxRetries {
			// Check if context was cancelled
			select {
			case <-req.Context().Done():
				return nil, fmt.Errorf("request cancelled during retry: %w", req.Context().Err())
			case <-time.After(backoff):
				// Calculate next backoff with exponential growth
				backoff = time.Duration(float64(backoff) * r.config.BackoffFactor)
				if backoff > r.config.MaxBackoff {
					backoff = r.config.MaxBackoff
				}
			}
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// calculateBackoff computes the backoff duration for a given attempt
func (r *RetryableHTTPClient) calculateBackoff(attempt int) time.Duration {
	backoff := float64(r.config.InitialBackoff) * math.Pow(r.config.BackoffFactor, float64(attempt))

	if backoff > float64(r.config.MaxBackoff) {
		return r.config.MaxBackoff
	}

	return time.Duration(backoff)
}

// DoWithRetry is a convenience function for executing requests with default retry config
func DoWithRetry(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	retryClient := &RetryableHTTPClient{
		client: client,
		config: DefaultRetryConfig(),
	}

	return retryClient.Do(req.WithContext(ctx))
}
