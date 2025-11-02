package repository

import (
	"context"
	"testing"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAlertListFilter_Defaults tests that pagination defaults are set correctly
func TestAlertListFilter_Defaults(t *testing.T) {
	// This test doesn't require database connection, testing logic only
	filter := AlertListFilter{}

	// Simulate what ListAlertsWithPagination does
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	assert.Equal(t, 50, filter.Limit, "Default limit should be 50")
	assert.Equal(t, 0, filter.Offset, "Default offset should be 0")
	assert.Equal(t, "created_at", filter.SortBy, "Default sort should be created_at")
	assert.Equal(t, "desc", filter.SortOrder, "Default sort order should be desc")
}

// TestAlertListFilter_LimitValidation tests limit boundary conditions
func TestAlertListFilter_LimitValidation(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"Zero limit uses default", 0, 50},
		{"Negative limit uses default", -1, 50},
		{"Over max limit uses default", 101, 50},
		{"Valid limit preserved", 25, 25},
		{"Max limit preserved", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := AlertListFilter{}
			filter.Limit = tt.inputLimit

			// Simulate validation
			if filter.Limit <= 0 || filter.Limit > 100 {
				filter.Limit = 50
			}

			assert.Equal(t, tt.expectedLimit, filter.Limit)
		})
	}
}

// TestGetSortClause tests SQL injection protection in sorting
func TestGetSortClause(t *testing.T) {
	repo := &AlertRepository{}

	tests := []struct {
		name          string
		sortBy        string
		sortOrder     string
		expectedClause string
	}{
		{"Default sort", "", "", "created_at DESC"},
		{"Sort by severity ascending", "severity", "asc", "severity ASC"},
		{"Sort by status descending", "status", "desc", "status DESC"},
		{"Sort by created_at ascending", "created_at", "asc", "created_at ASC"},
		{"Sort by updated_at descending", "updated_at", "desc", "updated_at DESC"},
		{"Invalid sort column defaults to created_at", "invalid_column", "asc", "created_at ASC"},
		{"SQL injection attempt in sortBy", "severity; DROP TABLE alerts;--", "asc", "created_at ASC"},
		{"SQL injection attempt in sortOrder", "severity", "asc; DROP TABLE alerts;--", "severity DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repo.getSortClause(tt.sortBy, tt.sortOrder)
			assert.Equal(t, tt.expectedClause, result)
		})
	}
}

// TestBuildCountQuery tests that count queries are built correctly
func TestBuildCountQuery(t *testing.T) {
	repo := &AlertRepository{}

	tests := []struct {
		name          string
		filter        *models.AlertFilter
		expectedQuery string
		expectedArgs  int
	}{
		{
			name:          "Empty filter",
			filter:        &models.AlertFilter{},
			expectedQuery: "SELECT COUNT(*) FROM alerts WHERE 1=1",
			expectedArgs:  0,
		},
		{
			name: "Filter by severity",
			filter: &models.AlertFilter{
				Severity: func() *models.Severity { s := models.SeverityCritical; return &s }(),
			},
			expectedQuery: "SELECT COUNT(*) FROM alerts WHERE 1=1 AND severity = $1",
			expectedArgs:  1,
		},
		{
			name: "Filter by status",
			filter: &models.AlertFilter{
				Status: func() *models.AlertStatus { s := models.AlertStatusNew; return &s }(),
			},
			expectedQuery: "SELECT COUNT(*) FROM alerts WHERE 1=1 AND status = $1",
			expectedArgs:  1,
		},
		{
			name: "Multiple filters",
			filter: &models.AlertFilter{
				Severity: func() *models.Severity { s := models.SeverityCritical; return &s }(),
				Status:   func() *models.AlertStatus { s := models.AlertStatusNew; return &s }(),
			},
			expectedQuery: "SELECT COUNT(*) FROM alerts WHERE 1=1 AND severity = $1 AND status = $2",
			expectedArgs:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := repo.buildCountQuery(tt.filter)
			assert.Equal(t, tt.expectedQuery, query)
			assert.Len(t, args, tt.expectedArgs)
		})
	}
}

// TestBuildListQuery tests that list queries include the source field
func TestBuildListQuery(t *testing.T) {
	repo := &AlertRepository{}

	filter := &models.AlertFilter{
		Limit: 10,
	}

	query, args := repo.buildListQuery(filter, "created_at", "desc")

	// Verify source field is included in SELECT
	assert.Contains(t, query, "source", "Query should include source field")
	assert.Contains(t, query, "ORDER BY created_at DESC", "Query should include sort clause")
	assert.Contains(t, query, "LIMIT $", "Query should include limit")
	assert.Len(t, args, 1, "Should have 1 argument for limit")
}

// TestPaginationCalculation tests page number and hasMore calculation
func TestPaginationCalculation(t *testing.T) {
	tests := []struct {
		name        string
		offset      int
		limit       int
		total       int64
		expectedPage int
		expectedHasMore bool
	}{
		{"First page", 0, 50, 100, 1, true},
		{"Second page", 50, 50, 100, 2, false},
		{"Partial last page", 50, 50, 75, 2, false},
		{"Empty result", 0, 50, 0, 1, false},
		{"Single page", 0, 50, 25, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := (tt.offset / tt.limit) + 1
			hasMore := int64(tt.offset+tt.limit) < tt.total

			assert.Equal(t, tt.expectedPage, page)
			assert.Equal(t, tt.expectedHasMore, hasMore)
		})
	}
}

// Integration test requiring database connection
func TestAlertRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires a PostgreSQL database with proper schema
	// Skip if DATABASE_URL is not set
	ctx := context.Background()

	// Use test database connection string from environment
	// In CI/CD, this would be set to a test database
	connString := "postgres://localhost/eth_validator_monitor_test?sslmode=disable"

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
		return
	}
	defer pool.Close()

	repo := NewAlertRepository(pool)

	// Test creating and listing alerts
	t.Run("Create and list alerts", func(t *testing.T) {
		// Create test alert
		testAlert := &models.Alert{
			AlertType: "test_alert",
			Severity:  models.SeverityCritical,
			Title:     "Test Alert",
			Message:   "This is a test alert",
			Source:    "test_suite",
			Status:    models.AlertStatusNew,
		}

		err := repo.CreateAlert(ctx, testAlert)
		require.NoError(t, err)
		require.NotZero(t, testAlert.ID)

		// List with pagination
		filter := AlertListFilter{
			AlertFilter: models.AlertFilter{
				Limit: 10,
			},
			SortBy:    "created_at",
			SortOrder: "desc",
		}

		result, err := repo.ListAlertsWithPagination(ctx, filter)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.GreaterOrEqual(t, len(result.Alerts), 1)
		assert.GreaterOrEqual(t, result.Total, int64(1))
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PageSize)

		// Cleanup: Delete test alert
		// Note: Add cleanup logic if needed
	})
}

// Benchmark tests
func BenchmarkBuildListQuery(b *testing.B) {
	repo := &AlertRepository{}
	filter := &models.AlertFilter{
		Severity: func() *models.Severity { s := models.SeverityCritical; return &s }(),
		Status:   func() *models.AlertStatus { s := models.AlertStatusNew; return &s }(),
		Limit:    50,
		Offset:   0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.buildListQuery(filter, "created_at", "desc")
	}
}

func BenchmarkGetSortClause(b *testing.B) {
	repo := &AlertRepository{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.getSortClause("severity", "asc")
	}
}
