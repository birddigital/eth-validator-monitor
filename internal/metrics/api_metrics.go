package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// APIMetrics provides Prometheus metrics for API performance monitoring
type APIMetrics struct {
	// API request latency histogram with percentile buckets
	RequestDuration *prometheus.HistogramVec

	// API requests total counter
	RequestsTotal *prometheus.CounterVec

	// API request errors counter
	RequestErrors *prometheus.CounterVec

	// Active API requests gauge
	ActiveRequests *prometheus.GaugeVec

	// Database query duration histogram
	DBQueryDuration *prometheus.HistogramVec

	// Database query errors counter
	DBQueryErrors *prometheus.CounterVec

	// Database connection pool metrics
	DBConnectionsActive   prometheus.Gauge
	DBConnectionsIdle     prometheus.Gauge
	DBConnectionsTotal    prometheus.Gauge
	DBConnectionsWaitTime *prometheus.HistogramVec

	// System resource metrics
	GoroutineCount    prometheus.Gauge
	MemoryAllocBytes  prometheus.Gauge
	MemorySysBytes    prometheus.Gauge
	GCPauseDuration   *prometheus.HistogramVec
	CPUUsagePercent   prometheus.Gauge
	DiskUsageBytes    *prometheus.GaugeVec
	NetworkBytesRecv  prometheus.Counter
	NetworkBytesSent  prometheus.Counter
}

// NewAPIMetrics creates and registers API and system performance metrics
func NewAPIMetrics() *APIMetrics {
	return &APIMetrics{
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "api_request_duration_seconds",
				Help: "API request latency in seconds",
				Buckets: []float64{
					0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
				},
			},
			[]string{"method", "endpoint", "status"},
		),

		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_requests_total",
				Help: "Total number of API requests",
			},
			[]string{"method", "endpoint", "status"},
		),

		RequestErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_request_errors_total",
				Help: "Total number of API request errors",
			},
			[]string{"method", "endpoint", "error_type"},
		),

		ActiveRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "api_requests_active",
				Help: "Number of currently active API requests",
			},
			[]string{"method", "endpoint"},
		),

		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "db_query_duration_seconds",
				Help: "Database query execution time in seconds",
				Buckets: []float64{
					0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0,
				},
			},
			[]string{"query_type", "table"},
		),

		DBQueryErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_query_errors_total",
				Help: "Total number of database query errors",
			},
			[]string{"query_type", "error_type"},
		),

		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
			},
		),

		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_idle",
				Help: "Number of idle database connections",
			},
		),

		DBConnectionsTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_total",
				Help: "Total number of database connections in pool",
			},
		),

		DBConnectionsWaitTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "db_connection_wait_seconds",
				Help: "Time spent waiting for database connection from pool",
				Buckets: []float64{
					0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0,
				},
			},
			[]string{"result"},
		),

		GoroutineCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_goroutines_count",
				Help: "Number of currently running goroutines",
			},
		),

		MemoryAllocBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_memory_alloc_bytes",
				Help: "Number of bytes allocated and still in use",
			},
		),

		MemorySysBytes: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_memory_sys_bytes",
				Help: "Number of bytes obtained from system",
			},
		),

		GCPauseDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "system_gc_pause_seconds",
				Help: "Garbage collection pause duration in seconds",
				Buckets: []float64{
					0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1,
				},
			},
			[]string{"type"},
		),

		CPUUsagePercent: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_cpu_usage_percent",
				Help: "CPU usage percentage (0-100)",
			},
		),

		DiskUsageBytes: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_disk_usage_bytes",
				Help: "Disk usage in bytes",
			},
			[]string{"mount_point", "type"},
		),

		NetworkBytesRecv: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "system_network_bytes_received_total",
				Help: "Total bytes received over network",
			},
		),

		NetworkBytesSent: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "system_network_bytes_sent_total",
				Help: "Total bytes sent over network",
			},
		),
	}
}

// RecordAPIRequest records an API request with its duration and status
func (m *APIMetrics) RecordAPIRequest(method, endpoint, status string, duration float64) {
	m.RequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)
	m.RequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// RecordAPIError records an API error
func (m *APIMetrics) RecordAPIError(method, endpoint, errorType string) {
	m.RequestErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// IncActiveRequests increments active request count
func (m *APIMetrics) IncActiveRequests(method, endpoint string) {
	m.ActiveRequests.WithLabelValues(method, endpoint).Inc()
}

// DecActiveRequests decrements active request count
func (m *APIMetrics) DecActiveRequests(method, endpoint string) {
	m.ActiveRequests.WithLabelValues(method, endpoint).Dec()
}

// RecordDBQuery records a database query with its duration
func (m *APIMetrics) RecordDBQuery(queryType, table string, duration float64) {
	m.DBQueryDuration.WithLabelValues(queryType, table).Observe(duration)
}

// RecordDBError records a database error
func (m *APIMetrics) RecordDBError(queryType, errorType string) {
	m.DBQueryErrors.WithLabelValues(queryType, errorType).Inc()
}

// UpdateDBConnectionStats updates database connection pool statistics
func (m *APIMetrics) UpdateDBConnectionStats(active, idle, total int) {
	m.DBConnectionsActive.Set(float64(active))
	m.DBConnectionsIdle.Set(float64(idle))
	m.DBConnectionsTotal.Set(float64(total))
}

// RecordDBConnectionWait records time spent waiting for a connection
func (m *APIMetrics) RecordDBConnectionWait(result string, duration float64) {
	m.DBConnectionsWaitTime.WithLabelValues(result).Observe(duration)
}

// UpdateSystemMetrics updates system resource metrics
func (m *APIMetrics) UpdateSystemMetrics(goroutines int, memAlloc, memSys uint64, cpuPercent float64) {
	m.GoroutineCount.Set(float64(goroutines))
	m.MemoryAllocBytes.Set(float64(memAlloc))
	m.MemorySysBytes.Set(float64(memSys))
	m.CPUUsagePercent.Set(cpuPercent)
}

// RecordGCPause records a garbage collection pause
func (m *APIMetrics) RecordGCPause(gcType string, duration float64) {
	m.GCPauseDuration.WithLabelValues(gcType).Observe(duration)
}

// UpdateDiskUsage updates disk usage metrics
func (m *APIMetrics) UpdateDiskUsage(mountPoint, usageType string, bytes uint64) {
	m.DiskUsageBytes.WithLabelValues(mountPoint, usageType).Set(float64(bytes))
}

// RecordNetworkTraffic records network bytes sent/received
func (m *APIMetrics) RecordNetworkTraffic(bytesSent, bytesRecv uint64) {
	m.NetworkBytesSent.Add(float64(bytesSent))
	m.NetworkBytesRecv.Add(float64(bytesRecv))
}
