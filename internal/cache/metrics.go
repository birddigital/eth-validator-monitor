package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// CacheMetrics tracks cache performance metrics
type CacheMetrics struct {
	// Hit/miss counters by data type
	validatorHits     atomic.Uint64
	validatorMisses   atomic.Uint64
	snapshotHits      atomic.Uint64
	snapshotMisses    atomic.Uint64
	performanceHits   atomic.Uint64
	performanceMisses atomic.Uint64
	networkStatsHits  atomic.Uint64
	networkStatsMisses atomic.Uint64
	alertHits         atomic.Uint64
	alertMisses       atomic.Uint64

	// Operation metrics
	getOperations    atomic.Uint64
	setOperations    atomic.Uint64
	deleteOperations atomic.Uint64
	totalLatencyUs   atomic.Uint64

	// Error tracking
	errorCount    atomic.Uint64
	errorsByType  map[string]*atomic.Uint64
	errorMu       sync.RWMutex

	// Memory tracking
	memoryUsedBytes   atomic.Uint64
	memoryPeakBytes   atomic.Uint64
	lastMemoryCheck   time.Time
	memoryCheckMu     sync.RWMutex

	// Rate limiting metrics
	rateLimitHits atomic.Uint64

	// Start time for rate calculations
	startTime time.Time
}

// NewCacheMetrics creates a new cache metrics collector
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{
		errorsByType: make(map[string]*atomic.Uint64),
		startTime:    time.Now(),
		lastMemoryCheck: time.Now(),
	}
}

// RecordHit records a cache hit for a specific data type
func (cm *CacheMetrics) RecordHit(dataType CacheDataType) {
	switch dataType {
	case CacheDataTypeValidator:
		cm.validatorHits.Add(1)
	case CacheDataTypeSnapshot:
		cm.snapshotHits.Add(1)
	case CacheDataTypePerformance:
		cm.performanceHits.Add(1)
	case CacheDataTypeNetworkStats:
		cm.networkStatsHits.Add(1)
	case CacheDataTypeAlert:
		cm.alertHits.Add(1)
	}
}

// RecordMiss records a cache miss for a specific data type
func (cm *CacheMetrics) RecordMiss(dataType CacheDataType) {
	switch dataType {
	case CacheDataTypeValidator:
		cm.validatorMisses.Add(1)
	case CacheDataTypeSnapshot:
		cm.snapshotMisses.Add(1)
	case CacheDataTypePerformance:
		cm.performanceMisses.Add(1)
	case CacheDataTypeNetworkStats:
		cm.networkStatsMisses.Add(1)
	case CacheDataTypeAlert:
		cm.alertMisses.Add(1)
	}
}

// RecordGet records a GET operation with latency
func (cm *CacheMetrics) RecordGet(latency time.Duration) {
	cm.getOperations.Add(1)
	cm.totalLatencyUs.Add(uint64(latency.Microseconds()))
}

// RecordSet records a SET operation
func (cm *CacheMetrics) RecordSet() {
	cm.setOperations.Add(1)
}

// RecordDelete records a DELETE operation
func (cm *CacheMetrics) RecordDelete() {
	cm.deleteOperations.Add(1)
}

// RecordError records a cache error by type
func (cm *CacheMetrics) RecordError(errorType string) {
	cm.errorCount.Add(1)

	cm.errorMu.Lock()
	counter, exists := cm.errorsByType[errorType]
	if !exists {
		counter = &atomic.Uint64{}
		cm.errorsByType[errorType] = counter
	}
	cm.errorMu.Unlock()

	counter.Add(1)
}

// RecordRateLimitHit records a rate limit hit
func (cm *CacheMetrics) RecordRateLimitHit() {
	cm.rateLimitHits.Add(1)
}

// UpdateMemoryUsage updates the current memory usage
func (cm *CacheMetrics) UpdateMemoryUsage(bytes uint64) {
	cm.memoryUsedBytes.Store(bytes)

	// Update peak if necessary
	for {
		current := cm.memoryPeakBytes.Load()
		if bytes <= current || cm.memoryPeakBytes.CompareAndSwap(current, bytes) {
			break
		}
	}

	cm.memoryCheckMu.Lock()
	cm.lastMemoryCheck = time.Now()
	cm.memoryCheckMu.Unlock()
}

// GetMetrics returns a snapshot of current metrics
func (cm *CacheMetrics) GetMetrics() CacheMetricsSnapshot {
	cm.errorMu.RLock()
	errors := make(map[string]uint64, len(cm.errorsByType))
	for k, v := range cm.errorsByType {
		errors[k] = v.Load()
	}
	cm.errorMu.RUnlock()

	cm.memoryCheckMu.RLock()
	lastCheck := cm.lastMemoryCheck
	cm.memoryCheckMu.RUnlock()

	// Calculate totals
	totalHits := cm.validatorHits.Load() + cm.snapshotHits.Load() +
		cm.performanceHits.Load() + cm.networkStatsHits.Load() + cm.alertHits.Load()
	totalMisses := cm.validatorMisses.Load() + cm.snapshotMisses.Load() +
		cm.performanceMisses.Load() + cm.networkStatsMisses.Load() + cm.alertMisses.Load()

	totalOps := totalHits + totalMisses
	hitRate := 0.0
	if totalOps > 0 {
		hitRate = float64(totalHits) / float64(totalOps) * 100.0
	}

	// Calculate average latency
	avgLatencyUs := uint64(0)
	getOps := cm.getOperations.Load()
	if getOps > 0 {
		avgLatencyUs = cm.totalLatencyUs.Load() / getOps
	}

	return CacheMetricsSnapshot{
		// Hit/miss by type
		ValidatorHits:     cm.validatorHits.Load(),
		ValidatorMisses:   cm.validatorMisses.Load(),
		SnapshotHits:      cm.snapshotHits.Load(),
		SnapshotMisses:    cm.snapshotMisses.Load(),
		PerformanceHits:   cm.performanceHits.Load(),
		PerformanceMisses: cm.performanceMisses.Load(),
		NetworkStatsHits:  cm.networkStatsHits.Load(),
		NetworkStatsMisses: cm.networkStatsMisses.Load(),
		AlertHits:         cm.alertHits.Load(),
		AlertMisses:       cm.alertMisses.Load(),

		// Aggregate metrics
		TotalHits:   totalHits,
		TotalMisses: totalMisses,
		HitRate:     hitRate,

		// Operations
		GetOperations:    getOps,
		SetOperations:    cm.setOperations.Load(),
		DeleteOperations: cm.deleteOperations.Load(),
		AvgLatencyUs:     avgLatencyUs,
		AvgLatencyMs:     float64(avgLatencyUs) / 1000.0,

		// Errors
		ErrorCount:   cm.errorCount.Load(),
		ErrorsByType: errors,

		// Memory
		MemoryUsedBytes: cm.memoryUsedBytes.Load(),
		MemoryPeakBytes: cm.memoryPeakBytes.Load(),
		MemoryUsedMB:    float64(cm.memoryUsedBytes.Load()) / 1024.0 / 1024.0,
		MemoryPeakMB:    float64(cm.memoryPeakBytes.Load()) / 1024.0 / 1024.0,
		LastMemoryCheck: lastCheck,

		// Rate limiting
		RateLimitHits: cm.rateLimitHits.Load(),

		// Uptime
		UptimeSeconds: time.Since(cm.startTime).Seconds(),
	}
}

// CacheMetricsSnapshot represents a point-in-time snapshot of cache metrics
type CacheMetricsSnapshot struct {
	// Hit/miss by type
	ValidatorHits      uint64 `json:"validator_hits"`
	ValidatorMisses    uint64 `json:"validator_misses"`
	SnapshotHits       uint64 `json:"snapshot_hits"`
	SnapshotMisses     uint64 `json:"snapshot_misses"`
	PerformanceHits    uint64 `json:"performance_hits"`
	PerformanceMisses  uint64 `json:"performance_misses"`
	NetworkStatsHits   uint64 `json:"network_stats_hits"`
	NetworkStatsMisses uint64 `json:"network_stats_misses"`
	AlertHits          uint64 `json:"alert_hits"`
	AlertMisses        uint64 `json:"alert_misses"`

	// Aggregate metrics
	TotalHits   uint64  `json:"total_hits"`
	TotalMisses uint64  `json:"total_misses"`
	HitRate     float64 `json:"hit_rate_percent"`

	// Operations
	GetOperations    uint64  `json:"get_operations"`
	SetOperations    uint64  `json:"set_operations"`
	DeleteOperations uint64  `json:"delete_operations"`
	AvgLatencyUs     uint64  `json:"avg_latency_us"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`

	// Errors
	ErrorCount   uint64            `json:"error_count"`
	ErrorsByType map[string]uint64 `json:"errors_by_type"`

	// Memory
	MemoryUsedBytes uint64    `json:"memory_used_bytes"`
	MemoryPeakBytes uint64    `json:"memory_peak_bytes"`
	MemoryUsedMB    float64   `json:"memory_used_mb"`
	MemoryPeakMB    float64   `json:"memory_peak_mb"`
	LastMemoryCheck time.Time `json:"last_memory_check"`

	// Rate limiting
	RateLimitHits uint64 `json:"rate_limit_hits"`

	// Uptime
	UptimeSeconds float64 `json:"uptime_seconds"`
}

// CacheDataType represents different types of cached data
type CacheDataType string

const (
	CacheDataTypeValidator    CacheDataType = "validator"
	CacheDataTypeSnapshot     CacheDataType = "snapshot"
	CacheDataTypePerformance  CacheDataType = "performance"
	CacheDataTypeNetworkStats CacheDataType = "network_stats"
	CacheDataTypeAlert        CacheDataType = "alert"
)

// MetricsMonitor periodically collects Redis memory stats
type MetricsMonitor struct {
	cache    *RedisCache
	metrics  *CacheMetrics
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewMetricsMonitor creates a new metrics monitor
func NewMetricsMonitor(cache *RedisCache, metrics *CacheMetrics, interval time.Duration) *MetricsMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &MetricsMonitor{
		cache:    cache,
		metrics:  metrics,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins periodic metrics collection
func (mm *MetricsMonitor) Start() {
	mm.wg.Add(1)
	go mm.runMonitoring()
	log.Printf("Cache metrics monitor started (interval: %v)", mm.interval)
}

// Stop stops the metrics monitor
func (mm *MetricsMonitor) Stop() {
	mm.cancel()
	mm.wg.Wait()
	log.Println("Cache metrics monitor stopped")
}

// runMonitoring runs the periodic monitoring loop
func (mm *MetricsMonitor) runMonitoring() {
	defer mm.wg.Done()

	ticker := time.NewTicker(mm.interval)
	defer ticker.Stop()

	// Initial collection
	mm.collectMemoryStats()

	for {
		select {
		case <-mm.ctx.Done():
			return
		case <-ticker.C:
			mm.collectMemoryStats()
			mm.checkHealthThresholds()
		}
	}
}

// collectMemoryStats collects memory statistics from Redis
func (mm *MetricsMonitor) collectMemoryStats() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := mm.cache.client.Info(ctx, "memory").Result()
	if err != nil {
		log.Printf("Failed to get Redis memory info: %v", err)
		mm.metrics.RecordError("memory_info_failed")
		return
	}

	// Parse used_memory from INFO output
	var usedMemory uint64
	if _, err := fmt.Sscanf(info, "used_memory:%d", &usedMemory); err == nil {
		mm.metrics.UpdateMemoryUsage(usedMemory)
	}
}

// checkHealthThresholds checks if metrics exceed health thresholds
func (mm *MetricsMonitor) checkHealthThresholds() {
	snapshot := mm.metrics.GetMetrics()

	// Check hit rate threshold (warn if below 70%)
	if snapshot.HitRate < 70.0 && snapshot.TotalHits+snapshot.TotalMisses > 1000 {
		log.Printf("WARNING: Low cache hit rate: %.2f%% (threshold: 70%%)", snapshot.HitRate)
	}

	// Check memory threshold (warn if above 1GB)
	if snapshot.MemoryUsedMB > 1024.0 {
		log.Printf("WARNING: High Redis memory usage: %.2f MB (threshold: 1024 MB)", snapshot.MemoryUsedMB)
	}

	// Check error rate (warn if above 5%)
	totalOps := snapshot.GetOperations + snapshot.SetOperations + snapshot.DeleteOperations
	if totalOps > 0 {
		errorRate := float64(snapshot.ErrorCount) / float64(totalOps) * 100.0
		if errorRate > 5.0 {
			log.Printf("WARNING: High cache error rate: %.2f%% (threshold: 5%%)", errorRate)
		}
	}

	// Check average latency (warn if above 10ms)
	if snapshot.AvgLatencyMs > 10.0 {
		log.Printf("WARNING: High cache latency: %.2f ms (threshold: 10 ms)", snapshot.AvgLatencyMs)
	}
}

// GetTypeHitRate returns the hit rate for a specific data type
func (cm *CacheMetrics) GetTypeHitRate(dataType CacheDataType) float64 {
	var hits, misses uint64

	switch dataType {
	case CacheDataTypeValidator:
		hits = cm.validatorHits.Load()
		misses = cm.validatorMisses.Load()
	case CacheDataTypeSnapshot:
		hits = cm.snapshotHits.Load()
		misses = cm.snapshotMisses.Load()
	case CacheDataTypePerformance:
		hits = cm.performanceHits.Load()
		misses = cm.performanceMisses.Load()
	case CacheDataTypeNetworkStats:
		hits = cm.networkStatsHits.Load()
		misses = cm.networkStatsMisses.Load()
	case CacheDataTypeAlert:
		hits = cm.alertHits.Load()
		misses = cm.alertMisses.Load()
	}

	total := hits + misses
	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total) * 100.0
}

// PrometheusMetrics provides metrics in Prometheus format
type PrometheusMetrics struct {
	metrics *CacheMetrics
}

// NewPrometheusMetrics creates a Prometheus metrics provider
func NewPrometheusMetrics(metrics *CacheMetrics) *PrometheusMetrics {
	return &PrometheusMetrics{
		metrics: metrics,
	}
}

// GetMetricsText returns metrics in Prometheus text format
func (pm *PrometheusMetrics) GetMetricsText() string {
	snapshot := pm.metrics.GetMetrics()

	return fmt.Sprintf(`# HELP cache_hits_total Total number of cache hits by type
# TYPE cache_hits_total counter
cache_hits_total{type="validator"} %d
cache_hits_total{type="snapshot"} %d
cache_hits_total{type="performance"} %d
cache_hits_total{type="network_stats"} %d
cache_hits_total{type="alert"} %d

# HELP cache_misses_total Total number of cache misses by type
# TYPE cache_misses_total counter
cache_misses_total{type="validator"} %d
cache_misses_total{type="snapshot"} %d
cache_misses_total{type="performance"} %d
cache_misses_total{type="network_stats"} %d
cache_misses_total{type="alert"} %d

# HELP cache_hit_rate Cache hit rate percentage
# TYPE cache_hit_rate gauge
cache_hit_rate %.2f

# HELP cache_operations_total Total number of cache operations by type
# TYPE cache_operations_total counter
cache_operations_total{operation="get"} %d
cache_operations_total{operation="set"} %d
cache_operations_total{operation="delete"} %d

# HELP cache_latency_microseconds Average cache operation latency
# TYPE cache_latency_microseconds gauge
cache_latency_microseconds %d

# HELP cache_errors_total Total number of cache errors by type
# TYPE cache_errors_total counter
%s

# HELP cache_memory_bytes Redis memory usage in bytes
# TYPE cache_memory_bytes gauge
cache_memory_bytes{type="used"} %d
cache_memory_bytes{type="peak"} %d

# HELP cache_rate_limit_hits_total Total rate limit hits
# TYPE cache_rate_limit_hits_total counter
cache_rate_limit_hits_total %d

# HELP cache_uptime_seconds Cache uptime in seconds
# TYPE cache_uptime_seconds gauge
cache_uptime_seconds %.2f
`,
		snapshot.ValidatorHits, snapshot.SnapshotHits, snapshot.PerformanceHits,
		snapshot.NetworkStatsHits, snapshot.AlertHits,
		snapshot.ValidatorMisses, snapshot.SnapshotMisses, snapshot.PerformanceMisses,
		snapshot.NetworkStatsMisses, snapshot.AlertMisses,
		snapshot.HitRate,
		snapshot.GetOperations, snapshot.SetOperations, snapshot.DeleteOperations,
		snapshot.AvgLatencyUs,
		pm.formatErrorMetrics(snapshot.ErrorsByType),
		snapshot.MemoryUsedBytes, snapshot.MemoryPeakBytes,
		snapshot.RateLimitHits,
		snapshot.UptimeSeconds,
	)
}

// formatErrorMetrics formats error metrics for Prometheus
func (pm *PrometheusMetrics) formatErrorMetrics(errors map[string]uint64) string {
	result := ""
	for errorType, count := range errors {
		result += fmt.Sprintf(`cache_errors_total{type="%s"} %d%s`, errorType, count, "\n")
	}
	if result == "" {
		result = "cache_errors_total{type=\"none\"} 0\n"
	}
	return result
}
