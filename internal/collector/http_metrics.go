package collector

import (
	"net/http"
	"sync/atomic"
	"time"
)

// HTTPMetrics tracks metrics for HTTP API calls
type HTTPMetrics struct {
	// Request counters by status code
	requestsTotal       atomic.Uint64
	requestsSuccess     atomic.Uint64 // 2xx status codes
	requestsClientError atomic.Uint64 // 4xx status codes
	requestsServerError atomic.Uint64 // 5xx status codes
	requestsTimeout     atomic.Uint64

	// Latency tracking
	totalLatencyMs atomic.Uint64
	minLatencyMs   atomic.Uint64
	maxLatencyMs   atomic.Uint64

	// Retry tracking
	retriesTotal atomic.Uint64

	// Endpoint-specific tracking
	endpointCalls map[string]*atomic.Uint64
}

// NewHTTPMetrics creates a new HTTP metrics tracker
func NewHTTPMetrics() *HTTPMetrics {
	m := &HTTPMetrics{
		endpointCalls: make(map[string]*atomic.Uint64),
	}
	m.minLatencyMs.Store(^uint64(0)) // Initialize to max value
	return m
}

// RecordRequest records an HTTP request with its outcome
func (m *HTTPMetrics) RecordRequest(statusCode int, latency time.Duration, retries int) {
	m.requestsTotal.Add(1)

	// Categorize by status code
	if statusCode >= 200 && statusCode < 300 {
		m.requestsSuccess.Add(1)
	} else if statusCode >= 400 && statusCode < 500 {
		m.requestsClientError.Add(1)
	} else if statusCode >= 500 {
		m.requestsServerError.Add(1)
	}

	// Record latency
	latencyMs := uint64(latency.Milliseconds())
	m.totalLatencyMs.Add(latencyMs)

	// Update min/max latency
	for {
		current := m.minLatencyMs.Load()
		if latencyMs >= current || m.minLatencyMs.CompareAndSwap(current, latencyMs) {
			break
		}
	}

	for {
		current := m.maxLatencyMs.Load()
		if latencyMs <= current || m.maxLatencyMs.CompareAndSwap(current, latencyMs) {
			break
		}
	}

	// Record retries
	if retries > 0 {
		m.retriesTotal.Add(uint64(retries))
	}
}

// RecordTimeout records an HTTP timeout
func (m *HTTPMetrics) RecordTimeout() {
	m.requestsTotal.Add(1)
	m.requestsTimeout.Add(1)
}

// GetSnapshot returns a snapshot of current metrics
func (m *HTTPMetrics) GetSnapshot() HTTPMetricsSnapshot {
	total := m.requestsTotal.Load()
	totalLatency := m.totalLatencyMs.Load()
	avgLatency := float64(0)
	if total > 0 {
		avgLatency = float64(totalLatency) / float64(total)
	}

	return HTTPMetricsSnapshot{
		RequestsTotal:       total,
		RequestsSuccess:     m.requestsSuccess.Load(),
		RequestsClientError: m.requestsClientError.Load(),
		RequestsServerError: m.requestsServerError.Load(),
		RequestsTimeout:     m.requestsTimeout.Load(),
		AvgLatencyMs:        avgLatency,
		MinLatencyMs:        float64(m.minLatencyMs.Load()),
		MaxLatencyMs:        float64(m.maxLatencyMs.Load()),
		RetriesTotal:        m.retriesTotal.Load(),
		SuccessRate:         m.calculateSuccessRate(),
	}
}

// calculateSuccessRate returns the percentage of successful requests
func (m *HTTPMetrics) calculateSuccessRate() float64 {
	total := m.requestsTotal.Load()
	if total == 0 {
		return 0
	}
	success := m.requestsSuccess.Load()
	return (float64(success) / float64(total)) * 100.0
}

// HTTPMetricsSnapshot represents a point-in-time view of HTTP metrics
type HTTPMetricsSnapshot struct {
	RequestsTotal       uint64  `json:"requests_total"`
	RequestsSuccess     uint64  `json:"requests_success"`
	RequestsClientError uint64  `json:"requests_client_error"`
	RequestsServerError uint64  `json:"requests_server_error"`
	RequestsTimeout     uint64  `json:"requests_timeout"`
	AvgLatencyMs        float64 `json:"avg_latency_ms"`
	MinLatencyMs        float64 `json:"min_latency_ms"`
	MaxLatencyMs        float64 `json:"max_latency_ms"`
	RetriesTotal        uint64  `json:"retries_total"`
	SuccessRate         float64 `json:"success_rate_percent"`
}

// MetricsTransport wraps an HTTP transport with metrics collection
type MetricsTransport struct {
	Transport http.RoundTripper
	Metrics   *HTTPMetrics
}

// RoundTrip implements the http.RoundTripper interface with metrics
func (t *MetricsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	resp, err := t.Transport.RoundTrip(req)
	latency := time.Since(start)

	if err != nil {
		t.Metrics.RecordTimeout()
		return resp, err
	}

	// Record successful request (retries are tracked elsewhere)
	t.Metrics.RecordRequest(resp.StatusCode, latency, 0)

	return resp, nil
}

// NewMetricsHTTPClient creates an HTTP client with metrics tracking
func NewMetricsHTTPClient(timeout time.Duration, metrics *HTTPMetrics) *http.Client {
	transport := &MetricsTransport{
		Transport: http.DefaultTransport,
		Metrics:   metrics,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}
