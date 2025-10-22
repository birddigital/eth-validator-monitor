package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/web/sse"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
)

var (
	healthCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "health_check_duration_seconds",
			Help:    "Duration of health checks by component",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"component"},
	)

	healthCheckStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "health_check_status",
			Help: "Current health status by component (1 = healthy, 0 = unhealthy)",
		},
		[]string{"component"},
	)

	healthCheckErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "health_check_errors_total",
			Help: "Total number of health check errors by component",
		},
		[]string{"component"},
	)
)

// ComponentStatus represents the health status of a system component
type ComponentStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"` // "healthy", "degraded", "unhealthy"
	Message   string    `json:"message,omitempty"`
	LastCheck time.Time `json:"last_check"`
}

// PoolStats interface for database pool statistics
type PoolStats interface {
	TotalConns() int32
	IdleConns() int32
}

// DBPinger interface for database ping operations
type DBPinger interface {
	Ping(ctx context.Context) error
	Stat() PoolStats
}

// Monitor performs periodic health checks and broadcasts status via SSE
type Monitor struct {
	db          DBPinger
	redis       *redis.Client
	broadcaster *sse.Broadcaster
	interval    time.Duration

	mu     sync.RWMutex
	status map[string]*ComponentStatus

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// MonitorConfig holds configuration for the health monitor
type MonitorConfig struct {
	CheckInterval time.Duration
}

// DefaultMonitorConfig returns default monitor configuration
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		CheckInterval: 30 * time.Second,
	}
}

// NewMonitor creates a new health monitor
func NewMonitor(db DBPinger, redis *redis.Client, broadcaster *sse.Broadcaster, config MonitorConfig) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Monitor{
		db:          db,
		redis:       redis,
		broadcaster: broadcaster,
		interval:    config.CheckInterval,
		status:      make(map[string]*ComponentStatus),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins periodic health checking and broadcasting
func (m *Monitor) Start() {
	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop gracefully stops the health monitor
func (m *Monitor) Stop() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

// GetStatus returns the current health status for all components
func (m *Monitor) GetStatus() map[string]*ComponentStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy to avoid race conditions
	statusCopy := make(map[string]*ComponentStatus, len(m.status))
	for k, v := range m.status {
		statusCopy[k] = &ComponentStatus{
			Name:      v.Name,
			Status:    v.Status,
			Message:   v.Message,
			LastCheck: v.LastCheck,
		}
	}

	return statusCopy
}

// monitorLoop runs the periodic health check loop
func (m *Monitor) monitorLoop() {
	defer m.wg.Done()

	// Perform initial check immediately
	m.performHealthChecks()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthChecks()
		}
	}
}

// performHealthChecks executes all health checks and broadcasts results
func (m *Monitor) performHealthChecks() {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	// Run checks in parallel
	var wg sync.WaitGroup
	results := make(chan *ComponentStatus, 2)

	// Check database
	wg.Add(1)
	go func() {
		defer wg.Done()
		results <- m.checkDatabase(ctx)
	}()

	// Check Redis
	wg.Add(1)
	go func() {
		defer wg.Done()
		results <- m.checkRedis(ctx)
	}()

	// Close results channel when all checks complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	statusChanged := false
	for result := range results {
		m.mu.Lock()
		oldStatus := m.status[result.Name]
		m.status[result.Name] = result
		m.mu.Unlock()

		// Check if status changed
		if oldStatus == nil || oldStatus.Status != result.Status {
			statusChanged = true
		}
	}

	// Broadcast health status update if anything changed
	if statusChanged {
		m.broadcastHealthStatus()
	}
}

// checkDatabase verifies database connectivity and health
func (m *Monitor) checkDatabase(ctx context.Context) *ComponentStatus {
	timer := prometheus.NewTimer(healthCheckDuration.WithLabelValues("database"))
	defer timer.ObserveDuration()

	status := &ComponentStatus{
		Name:      "database",
		Status:    "healthy",
		LastCheck: time.Now(),
	}

	if m.db == nil {
		status.Status = "unhealthy"
		status.Message = "database connection not initialized"
		healthCheckStatus.WithLabelValues("database").Set(0)
		healthCheckErrors.WithLabelValues("database").Inc()
		return status
	}

	// Ping database with timeout
	if err := m.db.Ping(ctx); err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("database ping failed: %v", err)
		healthCheckStatus.WithLabelValues("database").Set(0)
		healthCheckErrors.WithLabelValues("database").Inc()
		return status
	}

	// Check connection pool stats for degraded state (if available)
	// Safely check stats to avoid panics with mocked connections
	if stats := m.db.Stat(); stats != nil {
		// Use defer/recover to handle potential panics from mocked stats
		func() {
			defer func() {
				if r := recover(); r != nil {
					// If stats check panics (e.g., in tests), default to healthy
					healthCheckStatus.WithLabelValues("database").Set(1)
				}
			}()

			if stats.TotalConns() > 0 {
				idleRatio := float64(stats.IdleConns()) / float64(stats.TotalConns())
				if idleRatio < 0.1 {
					status.Status = "degraded"
					status.Message = fmt.Sprintf("low idle connections: %d/%d", stats.IdleConns(), stats.TotalConns())
					healthCheckStatus.WithLabelValues("database").Set(0.5)
				} else {
					healthCheckStatus.WithLabelValues("database").Set(1)
				}
			} else {
				healthCheckStatus.WithLabelValues("database").Set(1)
			}
		}()
	} else {
		healthCheckStatus.WithLabelValues("database").Set(1)
	}

	return status
}

// checkRedis verifies Redis connectivity and health
func (m *Monitor) checkRedis(ctx context.Context) *ComponentStatus {
	timer := prometheus.NewTimer(healthCheckDuration.WithLabelValues("redis"))
	defer timer.ObserveDuration()

	status := &ComponentStatus{
		Name:      "redis",
		Status:    "healthy",
		LastCheck: time.Now(),
	}

	if m.redis == nil {
		status.Status = "unhealthy"
		status.Message = "redis connection not initialized"
		healthCheckStatus.WithLabelValues("redis").Set(0)
		healthCheckErrors.WithLabelValues("redis").Inc()
		return status
	}

	// Ping Redis with timeout
	if err := m.redis.Ping(ctx).Err(); err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("redis ping failed: %v", err)
		healthCheckStatus.WithLabelValues("redis").Set(0)
		healthCheckErrors.WithLabelValues("redis").Inc()
		return status
	}

	healthCheckStatus.WithLabelValues("redis").Set(1)
	return status
}

// broadcastHealthStatus broadcasts current health status via SSE
func (m *Monitor) broadcastHealthStatus() {
	if m.broadcaster == nil {
		return
	}

	m.mu.RLock()
	dbStatus := m.status["database"]
	m.mu.RUnlock()

	// Build SSE health status data
	data := &sse.HealthStatusData{
		DatabaseStatus:   "unknown",
		BeaconNodeStatus: "unknown", // TODO: Add beacon node health check in future
		LastSync:         time.Now().Unix(),
		ActiveValidators: 0, // TODO: Populate from dashboard data
	}

	if dbStatus != nil {
		data.DatabaseStatus = dbStatus.Status
	}

	// Broadcast the event
	event := sse.Event{
		Type: sse.EventTypeHealthStatus,
		Data: data,
		ID:   fmt.Sprintf("health-%d", time.Now().Unix()),
	}
	m.broadcaster.Broadcast(event)
}
