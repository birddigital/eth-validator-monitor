package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	dashboardQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "dashboard_query_duration_seconds",
			Help: "Duration of dashboard data queries by type",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)

	dashboardCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dashboard_cache_hits_total",
			Help: "Number of dashboard cache hits",
		},
		[]string{"cache_type"},
	)

	dashboardCacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "dashboard_cache_misses_total",
			Help: "Number of dashboard cache misses",
		},
		[]string{"cache_type"},
	)
)

// DashboardData represents the complete dashboard state
type DashboardData struct {
	Metrics       *repository.AggregateMetrics   `json:"metrics"`
	RecentAlerts  []*models.Alert                `json:"recent_alerts"`
	TopValidators []*repository.ValidatorSummary `json:"top_validators"`
	SystemHealth  *repository.SystemHealth       `json:"system_health"`
	LastUpdated   time.Time                      `json:"last_updated"`
}

// Service handles dashboard data aggregation with caching
type Service struct {
	dashboardRepo *repository.DashboardRepository
}

// NewService creates a new dashboard service
func NewService(dashboardRepo *repository.DashboardRepository) *Service {
	return &Service{
		dashboardRepo: dashboardRepo,
	}
}

// queryResult holds the result from a parallel query
type queryResult struct {
	metrics    *repository.AggregateMetrics
	alerts     []*models.Alert
	validators []*repository.ValidatorSummary
	health     *repository.SystemHealth
	err        error
}

// GetDashboardData fetches all dashboard data using parallel queries
// Implements the pattern recommended by /go-crypto for optimal performance
func (s *Service) GetDashboardData(ctx context.Context) (*DashboardData, error) {
	// Execute queries in parallel using goroutines
	resultCh := make(chan queryResult, 4)

	// Query 1: Aggregate metrics
	go func() {
		timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("metrics"))
		defer timer.ObserveDuration()

		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		metrics, err := s.dashboardRepo.GetAggregateMetrics(queryCtx)
		resultCh <- queryResult{metrics: metrics, err: err}
	}()

	// Query 2: Recent alerts
	go func() {
		timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("alerts"))
		defer timer.ObserveDuration()

		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		alerts, err := s.dashboardRepo.GetRecentAlerts(queryCtx, 5)
		resultCh <- queryResult{alerts: alerts, err: err}
	}()

	// Query 3: Top validators
	go func() {
		timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("top_validators"))
		defer timer.ObserveDuration()

		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		validators, err := s.dashboardRepo.GetTopValidators(queryCtx, 10)
		resultCh <- queryResult{validators: validators, err: err}
	}()

	// Query 4: System health
	go func() {
		timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("health"))
		defer timer.ObserveDuration()

		queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		health, err := s.dashboardRepo.GetSystemHealth(queryCtx)
		resultCh <- queryResult{health: health, err: err}
	}()

	// Collect results from all parallel queries
	var (
		metrics    *repository.AggregateMetrics
		alerts     []*models.Alert
		validators []*repository.ValidatorSummary
		health     *repository.SystemHealth
	)

	for i := 0; i < 4; i++ {
		res := <-resultCh
		if res.err != nil {
			return nil, fmt.Errorf("dashboard query failed: %w", res.err)
		}
		if res.metrics != nil {
			metrics = res.metrics
		}
		if res.alerts != nil {
			alerts = res.alerts
		}
		if res.validators != nil {
			validators = res.validators
		}
		if res.health != nil {
			health = res.health
		}
	}

	// Validate all required data was collected
	if metrics == nil || health == nil {
		return nil, fmt.Errorf("failed to collect all dashboard data")
	}

	// Initialize empty slices if no data
	if alerts == nil {
		alerts = []*models.Alert{}
	}
	if validators == nil {
		validators = []*repository.ValidatorSummary{}
	}

	data := &DashboardData{
		Metrics:       metrics,
		RecentAlerts:  alerts,
		TopValidators: validators,
		SystemHealth:  health,
		LastUpdated:   time.Now(),
	}

	return data, nil
}

// GetAggregateMetrics fetches only the aggregate metrics
func (s *Service) GetAggregateMetrics(ctx context.Context) (*repository.AggregateMetrics, error) {
	timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("metrics"))
	defer timer.ObserveDuration()

	return s.dashboardRepo.GetAggregateMetrics(ctx)
}

// GetRecentAlerts fetches only recent alerts
func (s *Service) GetRecentAlerts(ctx context.Context, limit int) ([]*models.Alert, error) {
	timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("alerts"))
	defer timer.ObserveDuration()

	return s.dashboardRepo.GetRecentAlerts(ctx, limit)
}

// GetTopValidators fetches only top validators
func (s *Service) GetTopValidators(ctx context.Context, limit int) ([]*repository.ValidatorSummary, error) {
	timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("top_validators"))
	defer timer.ObserveDuration()

	return s.dashboardRepo.GetTopValidators(ctx, limit)
}

// GetSystemHealth fetches only system health
func (s *Service) GetSystemHealth(ctx context.Context) (*repository.SystemHealth, error) {
	timer := prometheus.NewTimer(dashboardQueryDuration.WithLabelValues("health"))
	defer timer.ObserveDuration()

	return s.dashboardRepo.GetSystemHealth(ctx)
}
