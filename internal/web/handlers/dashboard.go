package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/services/dashboard"
	"github.com/birddigital/eth-validator-monitor/internal/services/health"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/components"
)

// DashboardHandler handles dashboard HTTP requests
type DashboardHandler struct {
	service        *dashboard.Service
	healthMonitor  *health.Monitor
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(service *dashboard.Service, healthMonitor *health.Monitor) *DashboardHandler {
	return &DashboardHandler{
		service:       service,
		healthMonitor: healthMonitor,
	}
}

// GetDashboard handles GET /api/dashboard
// Returns complete dashboard data including metrics, alerts, top validators, and system health
func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := h.service.GetDashboardData(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=10")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetMetrics handles GET /api/dashboard/metrics
// Returns HTML metrics cards for HTMX
func (h *DashboardHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metrics, err := h.service.GetAggregateMetrics(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=10")

	component := components.MetricsCards(metrics)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render metrics", http.StatusInternalServerError)
		return
	}
}

// GetAlerts handles GET /api/dashboard/alerts
// Returns HTML alert list for HTMX
func (h *DashboardHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	alerts, err := h.service.GetRecentAlerts(ctx, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=5")

	component := components.AlertList(alerts)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render alerts", http.StatusInternalServerError)
		return
	}
}

// GetTopValidators handles GET /api/dashboard/validators
// Returns HTML validator grid for HTMX
func (h *DashboardHandler) GetTopValidators(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	validators, err := h.service.GetTopValidators(ctx, 10)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=10")

	component := components.ValidatorGrid(validators)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render validators", http.StatusInternalServerError)
		return
	}
}

// GetSystemHealth handles GET /api/dashboard/health
// Returns system health status
func (h *DashboardHandler) GetSystemHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health, err := h.service.GetSystemHealth(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=5")

	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetHealthIndicators handles GET /api/dashboard/health-indicators
// Returns HTML health indicators component for HTMX
func (h *DashboardHandler) GetHealthIndicators(w http.ResponseWriter, r *http.Request) {
	if h.healthMonitor == nil {
		http.Error(w, "health monitor not available", http.StatusServiceUnavailable)
		return
	}

	// Get current health status from monitor
	status := h.healthMonitor.GetStatus()

	// Build component data
	data := components.HealthIndicatorsData{
		Updated: time.Now(),
	}

	// Map database status
	if dbStatus, ok := status["database"]; ok {
		data.Database = components.ComponentHealth{
			Status:  dbStatus.Status,
			Message: dbStatus.Message,
		}
	} else {
		data.Database = components.ComponentHealth{
			Status: "unknown",
		}
	}

	// Map Redis status
	if redisStatus, ok := status["redis"]; ok {
		data.Redis = components.ComponentHealth{
			Status:  redisStatus.Status,
			Message: redisStatus.Message,
		}
	} else {
		data.Redis = components.ComponentHealth{
			Status: "unknown",
		}
	}

	// Render Templ component
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	component := components.HealthIndicators(data)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render health indicators", http.StatusInternalServerError)
		return
	}
}
