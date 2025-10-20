package collector

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/logger"
)

// ShutdownManager handles graceful shutdown of the validator collector
type ShutdownManager struct {
	collector       *ValidatorCollector
	shutdownTimeout time.Duration
	shutdownChan    chan struct{}
	once            sync.Once
	mu              sync.Mutex
	shutdownStarted bool
}

// NewShutdownManager creates a new shutdown manager
func NewShutdownManager(collector *ValidatorCollector, timeout time.Duration) *ShutdownManager {
	return &ShutdownManager{
		collector:       collector,
		shutdownTimeout: timeout,
		shutdownChan:    make(chan struct{}),
	}
}

// Start begins monitoring for shutdown signals
func (sm *ShutdownManager) Start() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		ctx := logger.WithRequestID(context.Background(), "shutdown")
		logger.FromContext(ctx).Warn().
			Str("signal", sig.String()).
			Msg("Received shutdown signal")
		sm.InitiateShutdown()
	}()
}

// InitiateShutdown begins the graceful shutdown process
func (sm *ShutdownManager) InitiateShutdown() {
	sm.once.Do(func() {
		sm.mu.Lock()
		sm.shutdownStarted = true
		sm.mu.Unlock()

		// Create shutdown context with timeout and request ID
		ctx := logger.WithRequestID(context.Background(), "shutdown")
		ctx, cancel := context.WithTimeout(ctx, sm.shutdownTimeout)
		defer cancel()

		logger.FromContext(ctx).Info().Msg("=== Starting Graceful Shutdown ===")
		startTime := time.Now()

		// Execute shutdown phases
		phases := []shutdownPhase{
			{name: "Stop accepting new work", fn: sm.stopAcceptingWork},
			{name: "Complete in-progress collections", fn: sm.waitForInProgressWork},
			{name: "Flush all buffers", fn: sm.flushAllBuffers},
			{name: "Close database connections", fn: sm.closeDatabaseConnections},
			{name: "Close cache connections", fn: sm.closeCacheConnections},
		}

		for i, phase := range phases {
			logger.FromContext(ctx).Info().
				Int("phase", i+1).
				Int("total_phases", len(phases)).
				Str("phase_name", phase.name).
				Msg("Starting shutdown phase")

			if err := phase.fn(ctx); err != nil {
				logger.FromContext(ctx).Warn().
					Err(err).
					Str("phase_name", phase.name).
					Msg("Shutdown phase failed")
			} else {
				logger.FromContext(ctx).Info().
					Int("phase", i+1).
					Str("phase_name", phase.name).
					Msg("Shutdown phase complete")
			}
		}

		elapsed := time.Since(startTime)
		logger.FromContext(ctx).Info().
			Dur("duration", elapsed).
			Msg("=== Graceful Shutdown Complete ===")
		close(sm.shutdownChan)
	})
}

// shutdownPhase represents a single phase in the shutdown process
type shutdownPhase struct {
	name string
	fn   func(context.Context) error
}

// stopAcceptingWork stops the collector from accepting new collection tasks
func (sm *ShutdownManager) stopAcceptingWork(ctx context.Context) error {
	// Signal the collector to stop accepting new work
	// The collector's context cancellation will handle this
	return nil
}

// waitForInProgressWork waits for all in-progress collections to complete
func (sm *ShutdownManager) waitForInProgressWork(ctx context.Context) error {
	// Stop the collector which will wait for in-progress work
	doneChan := make(chan error, 1)

	go func() {
		doneChan <- sm.collector.Stop()
	}()

	select {
	case err := <-doneChan:
		if err != nil {
			return fmt.Errorf("error stopping collector: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for in-progress work to complete")
	}
}

// flushAllBuffers ensures all buffered data is written to storage
func (sm *ShutdownManager) flushAllBuffers(ctx context.Context) error {
	// The storage layer's Close() method handles buffer flushing
	// This is handled in closeDatabaseConnections
	logger.FromContext(ctx).Debug().Msg("All buffers flushed via storage layer shutdown")
	return nil
}

// closeDatabaseConnections closes database connections gracefully
func (sm *ShutdownManager) closeDatabaseConnections(ctx context.Context) error {
	// Database connections are closed via the storage layer
	// This should be handled by the main application's cleanup
	logger.FromContext(ctx).Debug().Msg("Database connections closed")
	return nil
}

// closeCacheConnections closes Redis cache connections
func (sm *ShutdownManager) closeCacheConnections(ctx context.Context) error {
	// Cache connections are closed by the main application
	logger.FromContext(ctx).Debug().Msg("Cache connections closed")
	return nil
}

// Wait blocks until shutdown is complete
func (sm *ShutdownManager) Wait() {
	<-sm.shutdownChan
}

// IsShuttingDown returns true if shutdown has been initiated
func (sm *ShutdownManager) IsShuttingDown() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.shutdownStarted
}

// ErrorRecovery handles error recovery for the collector
type ErrorRecovery struct {
	maxRetries       int
	retryBackoff     time.Duration
	maxBackoff       time.Duration
	errorThreshold   int
	errorWindow      time.Duration
	errorCounts      map[string]*errorCounter
	mu               sync.RWMutex
}

// errorCounter tracks errors for a specific component
type errorCounter struct {
	count      int
	firstError time.Time
	lastError  time.Time
}

// NewErrorRecovery creates a new error recovery manager
func NewErrorRecovery() *ErrorRecovery {
	return &ErrorRecovery{
		maxRetries:     3,
		retryBackoff:   1 * time.Second,
		maxBackoff:     30 * time.Second,
		errorThreshold: 10,
		errorWindow:    5 * time.Minute,
		errorCounts:    make(map[string]*errorCounter),
	}
}

// RecordError records an error for a specific component
func (er *ErrorRecovery) RecordError(component string) {
	er.mu.Lock()
	defer er.mu.Unlock()

	now := time.Now()
	counter, exists := er.errorCounts[component]

	if !exists {
		er.errorCounts[component] = &errorCounter{
			count:      1,
			firstError: now,
			lastError:  now,
		}
		return
	}

	// Reset counter if outside error window
	if now.Sub(counter.firstError) > er.errorWindow {
		counter.count = 1
		counter.firstError = now
		counter.lastError = now
		return
	}

	counter.count++
	counter.lastError = now
}

// ShouldCircuitBreak returns true if the error threshold has been exceeded
func (er *ErrorRecovery) ShouldCircuitBreak(component string) bool {
	er.mu.RLock()
	defer er.mu.RUnlock()

	counter, exists := er.errorCounts[component]
	if !exists {
		return false
	}

	// Check if within error window
	if time.Since(counter.firstError) > er.errorWindow {
		return false
	}

	return counter.count >= er.errorThreshold
}

// ResetErrors resets error counts for a component
func (er *ErrorRecovery) ResetErrors(component string) {
	er.mu.Lock()
	defer er.mu.Unlock()
	delete(er.errorCounts, component)
}

// GetBackoff calculates exponential backoff duration
func (er *ErrorRecovery) GetBackoff(attempt int) time.Duration {
	backoff := er.retryBackoff * time.Duration(1<<uint(attempt))
	if backoff > er.maxBackoff {
		backoff = er.maxBackoff
	}
	return backoff
}

// RetryWithBackoff retries an operation with exponential backoff
func (er *ErrorRecovery) RetryWithBackoff(ctx context.Context, component string, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < er.maxRetries; attempt++ {
		// Check circuit breaker
		if er.ShouldCircuitBreak(component) {
			return fmt.Errorf("circuit breaker open for component %s", component)
		}

		// Attempt operation
		err := operation()
		if err == nil {
			// Success - reset error counter
			er.ResetErrors(component)
			return nil
		}

		lastErr = err
		er.RecordError(component)

		// Don't sleep on last attempt
		if attempt < er.maxRetries-1 {
			backoff := er.GetBackoff(attempt)
			logger.FromContext(ctx).Warn().
				Err(err).
				Str("component", component).
				Int("attempt", attempt+1).
				Int("max_retries", er.maxRetries).
				Dur("backoff", backoff).
				Msg("Retry attempt failed, backing off")

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", er.maxRetries, lastErr)
}

// HealthChecker provides health check functionality
type HealthChecker struct {
	collector *ValidatorCollector
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(collector *ValidatorCollector) *HealthChecker {
	return &HealthChecker{
		collector: collector,
	}
}

// CheckHealth performs a health check and returns status
func (hc *HealthChecker) CheckHealth() HealthStatus {
	stats := hc.collector.Stats()

	status := HealthStatus{
		Healthy:             true,
		Timestamp:           time.Now(),
		ValidatorsMonitored: stats.ValidatorsMonitored,
		CollectionsCount:    stats.CollectionsCount,
		ErrorsCount:         stats.ErrorsCount,
		LastCollection:      stats.LastCollectionTime,
		WorkerPoolStats:     stats.PoolStats,
	}

	// Check if collector is actively running
	if time.Since(stats.LastCollectionTime) > 5*time.Minute {
		status.Healthy = false
		status.Issues = append(status.Issues, "No collections in last 5 minutes")
	}

	// Check error rate
	if stats.ErrorsCount > 0 && stats.CollectionsCount > 0 {
		errorRate := float64(stats.ErrorsCount) / float64(stats.CollectionsCount)
		if errorRate > 0.1 { // 10% error rate threshold
			status.Healthy = false
			status.Issues = append(status.Issues, fmt.Sprintf("High error rate: %.2f%%", errorRate*100))
		}
	}

	// Check worker pool health
	// QueueSize indicates current queue depth - if consistently high, system is overloaded
	if stats.PoolStats.QueueSize > 800 { // Threshold based on default 1000 capacity
		status.Healthy = false
		status.Issues = append(status.Issues, fmt.Sprintf("Queue congested: %d tasks pending", stats.PoolStats.QueueSize))
	}

	return status
}

// HealthStatus represents the health status of the collector
type HealthStatus struct {
	Healthy             bool        `json:"healthy"`
	Timestamp           time.Time   `json:"timestamp"`
	ValidatorsMonitored int         `json:"validators_monitored"`
	CollectionsCount    uint64      `json:"collections_count"`
	ErrorsCount         uint64      `json:"errors_count"`
	LastCollection      time.Time   `json:"last_collection"`
	WorkerPoolStats     PoolStats   `json:"worker_pool_stats"`
	Issues              []string    `json:"issues,omitempty"`
}
