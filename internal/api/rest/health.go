package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// HealthChecker provides health check functionality
type HealthChecker struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *pgxpool.Pool, redis *redis.Client) *HealthChecker {
	return &HealthChecker{
		db:    db,
		redis: redis,
	}
}

// HealthStatus represents the health status response
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]Check  `json:"checks"`
}

// Check represents a single health check result
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HandleHealth returns an HTTP handler for health checks
func (hc *HealthChecker) HandleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		status := HealthStatus{
			Timestamp: time.Now(),
			Checks:    make(map[string]Check),
		}

		// Check database
		dbCheck := hc.checkDatabase(ctx)
		status.Checks["database"] = dbCheck

		// Check Redis
		redisCheck := hc.checkRedis(ctx)
		status.Checks["redis"] = redisCheck

		// Determine overall status
		if dbCheck.Status == "healthy" && redisCheck.Status == "healthy" {
			status.Status = "healthy"
			w.WriteHeader(http.StatusOK)
		} else {
			status.Status = "unhealthy"
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

// HandleReadiness returns an HTTP handler for readiness checks
func (hc *HealthChecker) HandleReadiness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Quick database ping
		if err := hc.db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
				"reason": "database unavailable",
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	}
}

// HandleLiveness returns an HTTP handler for liveness checks
func (hc *HealthChecker) HandleLiveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple liveness check - if we can respond, we're alive
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	}
}

func (hc *HealthChecker) checkDatabase(ctx context.Context) Check {
	if hc.db == nil {
		return Check{
			Status:  "unhealthy",
			Message: "database connection not initialized",
		}
	}

	if err := hc.db.Ping(ctx); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return Check{
		Status: "healthy",
	}
}

func (hc *HealthChecker) checkRedis(ctx context.Context) Check {
	if hc.redis == nil {
		return Check{
			Status:  "unhealthy",
			Message: "redis connection not initialized",
		}
	}

	if err := hc.redis.Ping(ctx).Err(); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	return Check{
		Status: "healthy",
	}
}
