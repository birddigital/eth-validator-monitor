package collector

import (
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects operational metrics for the data collection service
type MetricsCollector struct {
	// Collection metrics
	collectionsTotal     atomic.Uint64
	collectionsSuccessful atomic.Uint64
	collectionsFailed     atomic.Uint64

	// Latency metrics (microseconds)
	totalLatency   atomic.Uint64
	minLatency     atomic.Uint64
	maxLatency     atomic.Uint64
	latencyBuckets *LatencyHistogram

	// Throughput metrics
	validatorsProcessedTotal atomic.Uint64
	snapshotsStoredTotal     atomic.Uint64
	bytesProcessedTotal      atomic.Uint64

	// Error metrics by type
	errorCounts map[string]*atomic.Uint64
	errorMu     sync.RWMutex

	// Rate limiting metrics
	rateLimitHits atomic.Uint64

	// Resource usage tracking
	goroutineCount atomic.Int32
	startTime      time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		errorCounts:    make(map[string]*atomic.Uint64),
		latencyBuckets: NewLatencyHistogram(),
		startTime:      time.Now(),
	}

	// Initialize min latency to max value
	mc.minLatency.Store(^uint64(0))

	return mc
}

// RecordCollection records a collection operation
func (mc *MetricsCollector) RecordCollection(success bool, latency time.Duration) {
	mc.collectionsTotal.Add(1)

	if success {
		mc.collectionsSuccessful.Add(1)
	} else {
		mc.collectionsFailed.Add(1)
	}

	// Record latency in microseconds
	latencyUs := uint64(latency.Microseconds())
	mc.totalLatency.Add(latencyUs)
	mc.latencyBuckets.Record(latency)

	// Update min/max latency
	for {
		current := mc.minLatency.Load()
		if latencyUs >= current || mc.minLatency.CompareAndSwap(current, latencyUs) {
			break
		}
	}

	for {
		current := mc.maxLatency.Load()
		if latencyUs <= current || mc.maxLatency.CompareAndSwap(current, latencyUs) {
			break
		}
	}
}

// RecordError records an error by type
func (mc *MetricsCollector) RecordError(errorType string) {
	mc.errorMu.Lock()
	counter, exists := mc.errorCounts[errorType]
	if !exists {
		counter = &atomic.Uint64{}
		mc.errorCounts[errorType] = counter
	}
	mc.errorMu.Unlock()

	counter.Add(1)
}

// RecordValidatorProcessed increments validator count
func (mc *MetricsCollector) RecordValidatorProcessed() {
	mc.validatorsProcessedTotal.Add(1)
}

// RecordSnapshotStored increments snapshot count
func (mc *MetricsCollector) RecordSnapshotStored() {
	mc.snapshotsStoredTotal.Add(1)
}

// RecordBytesProcessed adds to bytes processed
func (mc *MetricsCollector) RecordBytesProcessed(bytes uint64) {
	mc.bytesProcessedTotal.Add(bytes)
}

// RecordRateLimitHit increments rate limit counter
func (mc *MetricsCollector) RecordRateLimitHit() {
	mc.rateLimitHits.Add(1)
}

// SetGoroutineCount updates goroutine count
func (mc *MetricsCollector) SetGoroutineCount(count int32) {
	mc.goroutineCount.Store(count)
}

// GetMetrics returns current metrics snapshot
func (mc *MetricsCollector) GetMetrics() Metrics {
	mc.errorMu.RLock()
	errorCounts := make(map[string]uint64, len(mc.errorCounts))
	for k, v := range mc.errorCounts {
		errorCounts[k] = v.Load()
	}
	mc.errorMu.RUnlock()

	total := mc.collectionsTotal.Load()
	totalLatency := mc.totalLatency.Load()
	avgLatency := uint64(0)
	if total > 0 {
		avgLatency = totalLatency / total
	}

	return Metrics{
		// Collection metrics
		CollectionsTotal:     total,
		CollectionsSuccessful: mc.collectionsSuccessful.Load(),
		CollectionsFailed:     mc.collectionsFailed.Load(),

		// Latency metrics (converted to milliseconds)
		AvgLatencyMs: float64(avgLatency) / 1000.0,
		MinLatencyMs: float64(mc.minLatency.Load()) / 1000.0,
		MaxLatencyMs: float64(mc.maxLatency.Load()) / 1000.0,
		LatencyP50:   mc.latencyBuckets.Percentile(50),
		LatencyP95:   mc.latencyBuckets.Percentile(95),
		LatencyP99:   mc.latencyBuckets.Percentile(99),

		// Throughput metrics
		ValidatorsProcessed: mc.validatorsProcessedTotal.Load(),
		SnapshotsStored:     mc.snapshotsStoredTotal.Load(),
		BytesProcessed:      mc.bytesProcessedTotal.Load(),

		// Rate metrics
		CollectionsPerSecond: mc.calculateRate(total),
		ValidatorsPerSecond:  mc.calculateRate(mc.validatorsProcessedTotal.Load()),

		// Error metrics
		ErrorCounts:   errorCounts,
		RateLimitHits: mc.rateLimitHits.Load(),

		// Resource metrics
		GoroutineCount: mc.goroutineCount.Load(),
		UptimeSeconds:  time.Since(mc.startTime).Seconds(),
	}
}

// calculateRate calculates operations per second
func (mc *MetricsCollector) calculateRate(total uint64) float64 {
	elapsed := time.Since(mc.startTime).Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(total) / elapsed
}

// Metrics represents a snapshot of collected metrics
type Metrics struct {
	// Collection metrics
	CollectionsTotal      uint64  `json:"collections_total"`
	CollectionsSuccessful uint64  `json:"collections_successful"`
	CollectionsFailed     uint64  `json:"collections_failed"`

	// Latency metrics (milliseconds)
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	MinLatencyMs float64 `json:"min_latency_ms"`
	MaxLatencyMs float64 `json:"max_latency_ms"`
	LatencyP50   float64 `json:"latency_p50_ms"`
	LatencyP95   float64 `json:"latency_p95_ms"`
	LatencyP99   float64 `json:"latency_p99_ms"`

	// Throughput metrics
	ValidatorsProcessed uint64 `json:"validators_processed"`
	SnapshotsStored     uint64 `json:"snapshots_stored"`
	BytesProcessed      uint64 `json:"bytes_processed"`

	// Rate metrics
	CollectionsPerSecond float64 `json:"collections_per_second"`
	ValidatorsPerSecond  float64 `json:"validators_per_second"`

	// Error metrics
	ErrorCounts   map[string]uint64 `json:"error_counts"`
	RateLimitHits uint64            `json:"rate_limit_hits"`

	// Resource metrics
	GoroutineCount int32   `json:"goroutine_count"`
	UptimeSeconds  float64 `json:"uptime_seconds"`
}

// LatencyHistogram provides percentile calculations
type LatencyHistogram struct {
	buckets []uint64
	counts  []atomic.Uint64
	mu      sync.RWMutex
}

// NewLatencyHistogram creates a new latency histogram
// Buckets: 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 5s, 10s+
func NewLatencyHistogram() *LatencyHistogram {
	buckets := []uint64{
		1000,      // 1ms
		5000,      // 5ms
		10000,     // 10ms
		25000,     // 25ms
		50000,     // 50ms
		100000,    // 100ms
		250000,    // 250ms
		500000,    // 500ms
		1000000,   // 1s
		5000000,   // 5s
		10000000,  // 10s
	}

	counts := make([]atomic.Uint64, len(buckets)+1) // +1 for overflow bucket

	return &LatencyHistogram{
		buckets: buckets,
		counts:  counts,
	}
}

// Record records a latency observation
func (lh *LatencyHistogram) Record(latency time.Duration) {
	latencyUs := uint64(latency.Microseconds())

	for i, bucket := range lh.buckets {
		if latencyUs <= bucket {
			lh.counts[i].Add(1)
			return
		}
	}

	// Overflow bucket
	lh.counts[len(lh.buckets)].Add(1)
}

// Percentile calculates the specified percentile (0-100)
func (lh *LatencyHistogram) Percentile(p float64) float64 {
	// Calculate total observations
	var total uint64
	for i := range lh.counts {
		total += lh.counts[i].Load()
	}

	if total == 0 {
		return 0
	}

	// Find the bucket containing the percentile
	target := uint64(float64(total) * p / 100.0)
	var cumulative uint64

	for i, bucket := range lh.buckets {
		cumulative += lh.counts[i].Load()
		if cumulative >= target {
			// Return bucket upper bound in milliseconds
			return float64(bucket) / 1000.0
		}
	}

	// Overflow bucket
	return float64(lh.buckets[len(lh.buckets)-1]) / 1000.0
}

// PerformanceTuner provides runtime performance tuning
type PerformanceTuner struct {
	// Worker pool settings
	minWorkers     int
	maxWorkers     int
	currentWorkers atomic.Int32

	// Queue settings
	targetQueueDepth int
	scaleUpThreshold int
	scaleDownThreshold int

	// Collection settings
	batchSize            int
	collectionInterval   time.Duration
	maxConcurrentBatches int

	// Resource limits
	maxMemoryMB      uint64
	maxGoroutines    int

	mu sync.RWMutex
}

// NewPerformanceTuner creates a new performance tuner
func NewPerformanceTuner() *PerformanceTuner {
	return &PerformanceTuner{
		minWorkers:           5,
		maxWorkers:           50,
		targetQueueDepth:     100,
		scaleUpThreshold:     500,
		scaleDownThreshold:   50,
		batchSize:            100,
		collectionInterval:   12 * time.Second, // One Ethereum epoch
		maxConcurrentBatches: 10,
		maxMemoryMB:          1024, // 1GB
		maxGoroutines:        1000,
	}
}

// ShouldScaleUp determines if workers should be scaled up
func (pt *PerformanceTuner) ShouldScaleUp(queueDepth int) bool {
	current := pt.currentWorkers.Load()
	return queueDepth > pt.scaleUpThreshold && int(current) < pt.maxWorkers
}

// ShouldScaleDown determines if workers should be scaled down
func (pt *PerformanceTuner) ShouldScaleDown(queueDepth int) bool {
	current := pt.currentWorkers.Load()
	return queueDepth < pt.scaleDownThreshold && int(current) > pt.minWorkers
}

// GetWorkerCount returns current worker count
func (pt *PerformanceTuner) GetWorkerCount() int32 {
	return pt.currentWorkers.Load()
}

// SetWorkerCount updates worker count
func (pt *PerformanceTuner) SetWorkerCount(count int32) {
	pt.currentWorkers.Store(count)
}

// GetBatchSize returns optimal batch size
func (pt *PerformanceTuner) GetBatchSize() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.batchSize
}

// GetCollectionInterval returns collection interval
func (pt *PerformanceTuner) GetCollectionInterval() time.Duration {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return pt.collectionInterval
}

// UpdateSettings updates tuner settings
func (pt *PerformanceTuner) UpdateSettings(settings TunerSettings) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if settings.BatchSize > 0 {
		pt.batchSize = settings.BatchSize
	}
	if settings.CollectionInterval > 0 {
		pt.collectionInterval = settings.CollectionInterval
	}
	if settings.MaxWorkers > 0 {
		pt.maxWorkers = settings.MaxWorkers
	}
}

// TunerSettings represents tunable performance settings
type TunerSettings struct {
	BatchSize          int
	CollectionInterval time.Duration
	MaxWorkers         int
}
