package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/components"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
	"github.com/rs/zerolog"
)

// AlertsHandler handles the alerts management page
type AlertsHandler struct {
	repo   *repository.AlertRepository
	logger zerolog.Logger
}

// NewAlertsHandler creates a new alerts handler
func NewAlertsHandler(repo *repository.AlertRepository, logger zerolog.Logger) *AlertsHandler {
	return &AlertsHandler{
		repo:   repo,
		logger: logger,
	}
}

// ServeHTTP implements http.Handler
func (h *AlertsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filter := h.parseFilter(r)

	// Fetch alerts from repository
	alerts, err := h.repo.ListAlerts(r.Context(), &filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to fetch alerts")
		h.renderError(w, r, http.StatusInternalServerError, "Failed to fetch alerts")
		return
	}

	// Determine if HTMX partial or full page
	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, r, alerts, filter)
	} else {
		h.renderFull(w, r, alerts, filter)
	}
}

// HandleBatchAction handles batch operations (mark as read, dismiss)
func (h *AlertsHandler) HandleBatchAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.logger.Error().Err(err).Msg("Failed to parse form")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	action := r.FormValue("action")
	alertIDs := r.Form["alert_ids[]"]

	if len(alertIDs) == 0 {
		http.Error(w, "No alerts selected", http.StatusBadRequest)
		return
	}

	// Parse alert IDs
	ids := make([]int64, 0, len(alertIDs))
	for _, idStr := range alertIDs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.logger.Warn().Str("id", idStr).Msg("Invalid alert ID")
			continue
		}
		ids = append(ids, id)
	}

	// Perform batch action
	var newStatus models.AlertStatus
	switch action {
	case "mark_read":
		newStatus = models.AlertStatusAcknowledged
	case "dismiss":
		newStatus = models.AlertStatusIgnored
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Update alerts
	ctx := r.Context()
	successCount := 0
	for _, id := range ids {
		if err := h.repo.UpdateAlertStatus(ctx, int32(id), newStatus); err != nil {
			h.logger.Error().Err(err).Int64("alert_id", id).Msg("Failed to update alert status")
		} else {
			successCount++
		}
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"updated": successCount,
		"total":   len(ids),
	})
}

// HandleAlertCount returns the count of unread alerts for navbar badge
func (h *AlertsHandler) HandleAlertCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Count active alerts
	activeStatus := models.AlertStatusActive
	filter := models.AlertFilter{
		Status: &activeStatus,
	}

	alerts, err := h.repo.ListAlerts(ctx, &filter)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to fetch alert count")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	count := len(alerts)

	// Return count as JSON or HTML based on request type
	if r.Header.Get("HX-Request") == "true" {
		// HTMX partial update for badge
		w.Header().Set("Content-Type", "text/html")
		component := components.AlertBadge(count)
		if err := component.Render(ctx, w); err != nil {
			h.logger.Error().Err(err).Msg("Failed to render alert badge")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	} else {
		// JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"count": count})
	}
}

// parseFilter extracts filter parameters from request
func (h *AlertsHandler) parseFilter(r *http.Request) models.AlertFilter {
	query := r.URL.Query()
	filter := models.AlertFilter{}

	// Severity filter
	if severity := query.Get("severity"); severity != "" {
		sev := models.Severity(severity)
		filter.Severity = &sev
	}

	// Status filter
	if status := query.Get("status"); status != "" {
		st := models.AlertStatus(status)
		filter.Status = &st
	}

	// Alert type filter
	if alertType := query.Get("type"); alertType != "" {
		filter.AlertType = &alertType
	}

	// Validator index filter
	if valIndexStr := query.Get("validator"); valIndexStr != "" {
		if valIndex, err := strconv.ParseInt(valIndexStr, 10, 64); err == nil {
			filter.ValidatorIndex = &valIndex
		}
	}

	// Time range filter
	if since := query.Get("since"); since != "" {
		if duration, err := time.ParseDuration(since); err == nil {
			sinceTime := time.Now().Add(-duration)
			filter.StartTime = &sinceTime
		}
	}

	if until := query.Get("until"); until != "" {
		if untilTime, err := time.Parse(time.RFC3339, until); err == nil {
			filter.EndTime = &untilTime
		}
	}

	// Pagination
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	filter.Limit = limit

	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	filter.Offset = offset

	return filter
}

// renderFull renders the complete alerts page with layout
func (h *AlertsHandler) renderFull(w http.ResponseWriter, r *http.Request, alerts []*models.Alert, filter models.AlertFilter) {
	data := pages.AlertsPageData{
		Alerts: alerts,
		Filter: filter,
	}

	component := pages.AlertsPageWithLayout(data)
	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render alerts page")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderPartial renders only the alerts table or specific components
func (h *AlertsHandler) renderPartial(w http.ResponseWriter, r *http.Request, alerts []*models.Alert, filter models.AlertFilter) {
	target := r.Header.Get("HX-Target")

	hasMore := len(alerts) == filter.Limit
	w.Header().Set("Content-Type", "text/html")

	var err error
	switch {
	case strings.Contains(target, "alerts-table-body"):
		err = components.AlertsTableRows(alerts, filter, hasMore).Render(r.Context(), w)
	case strings.Contains(target, "infinite-scroll-trigger"):
		err = components.AlertsTableRowsWithTrigger(alerts, filter, hasMore).Render(r.Context(), w)
	default:
		err = components.AlertsTable(alerts, filter, hasMore).Render(r.Context(), w)
	}

	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to render partial")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderError renders an error message
func (h *AlertsHandler) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(status)
		w.Write([]byte(fmt.Sprintf(`<div class="error bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">%s</div>`, message)))
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
	}
}
