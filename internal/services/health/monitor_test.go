package health

import (
	"context"
	"testing"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/web/sse"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDBPinger wraps pgxmock to implement DBPinger interface
type mockDBPinger struct {
	pgxmock.PgxPoolIface
}

func (m mockDBPinger) Stat() PoolStats {
	return m.PgxPoolIface.Stat()
}

func TestNewMonitor(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	broadcaster := sse.NewBroadcaster(context.Background())

	config := DefaultMonitorConfig()
	monitor := NewMonitor(mockDBPinger{mock}, redisClient, broadcaster, config)

	assert.NotNil(t, monitor)
	assert.Equal(t, config.CheckInterval, monitor.interval)
	assert.NotNil(t, monitor.status)
	assert.NotNil(t, monitor.ctx)
}

func TestMonitor_CheckDatabase_Healthy(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Expect successful ping
	mock.ExpectPing()

	monitor := NewMonitor(mockDBPinger{mock}, nil, nil, DefaultMonitorConfig())
	ctx := context.Background()

	status := monitor.checkDatabase(ctx)

	assert.Equal(t, "database", status.Name)
	assert.Equal(t, "healthy", status.Status)
	assert.Empty(t, status.Message)
	assert.False(t, status.LastCheck.IsZero())

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMonitor_CheckDatabase_Unhealthy(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Expect ping to fail
	mock.ExpectPing().WillReturnError(assert.AnError)

	monitor := NewMonitor(mockDBPinger{mock}, nil, nil, DefaultMonitorConfig())
	ctx := context.Background()

	status := monitor.checkDatabase(ctx)

	assert.Equal(t, "database", status.Name)
	assert.Equal(t, "unhealthy", status.Status)
	assert.Contains(t, status.Message, "database ping failed")
	assert.False(t, status.LastCheck.IsZero())

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMonitor_CheckDatabase_NilConnection(t *testing.T) {
	monitor := NewMonitor(nil, nil, nil, DefaultMonitorConfig())
	ctx := context.Background()

	status := monitor.checkDatabase(ctx)

	assert.Equal(t, "database", status.Name)
	assert.Equal(t, "unhealthy", status.Status)
	assert.Equal(t, "database connection not initialized", status.Message)
}

func TestMonitor_CheckRedis_NilConnection(t *testing.T) {
	monitor := NewMonitor(nil, nil, nil, DefaultMonitorConfig())
	ctx := context.Background()

	status := monitor.checkRedis(ctx)

	assert.Equal(t, "redis", status.Name)
	assert.Equal(t, "unhealthy", status.Status)
	assert.Equal(t, "redis connection not initialized", status.Message)
}

func TestMonitor_GetStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	monitor := NewMonitor(mockDBPinger{mock}, nil, nil, DefaultMonitorConfig())

	// Manually set some status
	monitor.mu.Lock()
	monitor.status["database"] = &ComponentStatus{
		Name:      "database",
		Status:    "healthy",
		LastCheck: time.Now(),
	}
	monitor.status["redis"] = &ComponentStatus{
		Name:      "redis",
		Status:    "degraded",
		Message:   "high latency",
		LastCheck: time.Now(),
	}
	monitor.mu.Unlock()

	// Get status copy
	status := monitor.GetStatus()

	assert.Len(t, status, 2)
	assert.Equal(t, "healthy", status["database"].Status)
	assert.Equal(t, "degraded", status["redis"].Status)
	assert.Equal(t, "high latency", status["redis"].Message)

	// Verify it's a copy (modifying returned status shouldn't affect internal state)
	status["database"].Status = "unhealthy"

	monitor.mu.RLock()
	assert.Equal(t, "healthy", monitor.status["database"].Status)
	monitor.mu.RUnlock()
}

func TestMonitor_PerformHealthChecks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Set up mock expectations
	mock.ExpectPing()

	broadcaster := sse.NewBroadcaster(context.Background())
	monitor := NewMonitor(mockDBPinger{mock}, nil, broadcaster, DefaultMonitorConfig())

	// Perform health checks
	monitor.performHealthChecks()

	// Wait a bit for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Verify status was updated
	status := monitor.GetStatus()
	assert.NotNil(t, status["database"])
	assert.Equal(t, "healthy", status["database"].Status)

	// Redis should be unhealthy (nil connection)
	assert.NotNil(t, status["redis"])
	assert.Equal(t, "unhealthy", status["redis"].Status)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMonitor_StartStop(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Expect at least one ping during the test
	mock.ExpectPing()

	config := MonitorConfig{
		CheckInterval: 100 * time.Millisecond, // Fast interval for testing
	}

	broadcaster := sse.NewBroadcaster(context.Background())
	monitor := NewMonitor(mockDBPinger{mock}, nil, broadcaster, config)

	// Start monitor
	monitor.Start()

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Stop monitor
	err = monitor.Stop()
	assert.NoError(t, err)

	// Verify status was checked
	status := monitor.GetStatus()
	assert.NotNil(t, status["database"])
}

func TestMonitor_BroadcastHealthStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	broadcaster := sse.NewBroadcaster(context.Background())
	monitor := NewMonitor(mockDBPinger{mock}, nil, broadcaster, DefaultMonitorConfig())

	// Set up some status
	monitor.mu.Lock()
	monitor.status["database"] = &ComponentStatus{
		Name:      "database",
		Status:    "healthy",
		LastCheck: time.Now(),
	}
	monitor.status["redis"] = &ComponentStatus{
		Name:      "redis",
		Status:    "degraded",
		Message:   "slow response",
		LastCheck: time.Now(),
	}
	monitor.mu.Unlock()

	// Broadcast should not panic
	assert.NotPanics(t, func() {
		monitor.broadcastHealthStatus()
	})
}

func TestDefaultMonitorConfig(t *testing.T) {
	config := DefaultMonitorConfig()

	assert.Equal(t, 30*time.Second, config.CheckInterval)
}

func TestComponentStatus_ThreadSafety(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	monitor := NewMonitor(mockDBPinger{mock}, nil, nil, DefaultMonitorConfig())

	// Concurrent reads and writes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			monitor.mu.Lock()
			monitor.status["test"] = &ComponentStatus{
				Name:      "test",
				Status:    "healthy",
				LastCheck: time.Now(),
			}
			monitor.mu.Unlock()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = monitor.GetStatus()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Should not panic or race
}
