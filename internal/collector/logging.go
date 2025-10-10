package collector

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// LoggingTransport wraps an HTTP transport with request/response logging
type LoggingTransport struct {
	Transport http.RoundTripper
	Verbose   bool // If true, logs full request/response bodies
}

// RoundTrip implements the http.RoundTripper interface with logging
func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log request
	log.Printf("[BEACON-API] → %s %s", req.Method, req.URL.String())

	if t.Verbose && req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		if len(bodyBytes) > 0 {
			log.Printf("[BEACON-API] → Request Body: %s", truncateString(string(bodyBytes), 500))
		}
	}

	// Execute request
	resp, err := t.Transport.RoundTrip(req)

	duration := time.Since(start)

	// Log response
	if err != nil {
		log.Printf("[BEACON-API] ← %s %s failed after %v: %v", req.Method, req.URL.String(), duration, err)
		return resp, err
	}

	log.Printf("[BEACON-API] ← %s %s → %d in %v", req.Method, req.URL.Path, resp.StatusCode, duration)

	if t.Verbose && resp.Body != nil {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		if len(bodyBytes) > 0 {
			log.Printf("[BEACON-API] ← Response Body: %s", truncateString(string(bodyBytes), 500))
		}
	}

	return resp, nil
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + fmt.Sprintf("... (%d more chars)", len(s)-maxLen)
}

// NewLoggingHTTPClient creates an HTTP client with logging enabled
func NewLoggingHTTPClient(timeout time.Duration, verbose bool) *http.Client {
	transport := &LoggingTransport{
		Transport: http.DefaultTransport,
		Verbose:   verbose,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// NewLoggingRetryableHTTPClient creates a retryable HTTP client with logging
func NewLoggingRetryableHTTPClient(timeout time.Duration, config RetryConfig, verbose bool) *RetryableHTTPClient {
	transport := &LoggingTransport{
		Transport: http.DefaultTransport,
		Verbose:   verbose,
	}

	return &RetryableHTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		config: config,
	}
}
