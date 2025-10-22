package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/layouts"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// ValidatorDetailHandler handles the validator detail page and related endpoints
type ValidatorDetailHandler struct {
	repo   *repository.ValidatorDetailRepository
	logger zerolog.Logger
}

// NewValidatorDetailHandler creates a new validator detail handler
func NewValidatorDetailHandler(repo *repository.ValidatorDetailRepository, logger zerolog.Logger) *ValidatorDetailHandler {
	return &ValidatorDetailHandler{
		repo:   repo,
		logger: logger,
	}
}

// ValidatorPageData holds all data for the validator detail page
type ValidatorPageData struct {
	Validator         *repository.ValidatorDetails
	EffectivenessData []repository.EffectivenessPoint
	AttestationStats  []repository.AttestationStats
	Alerts            []repository.Alert
	Timeline          []repository.TimelineEvent
}

// ServeHTTP implements http.Handler for the main validator detail page
func (h *ValidatorDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract validator index from URL params
	validatorIndexStr := chi.URLParam(r, "index")
	validatorIndex, err := strconv.ParseInt(validatorIndexStr, 10, 64)
	if err != nil {
		h.logger.Error().Err(err).Str("index", validatorIndexStr).Msg("Invalid validator index")
		http.Error(w, "Invalid validator index", http.StatusBadRequest)
		return
	}

	// Fetch all data in parallel using errgroup
	g, gctx := errgroup.WithContext(ctx)

	var (
		details       *repository.ValidatorDetails
		effectiveness []repository.EffectivenessPoint
		attestations  []repository.AttestationStats
		alerts        []repository.Alert
		timeline      []repository.TimelineEvent
	)

	g.Go(func() error {
		var err error
		details, err = h.repo.GetValidatorDetails(gctx, validatorIndex)
		if err != nil {
			return fmt.Errorf("get validator details: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		effectiveness, err = h.repo.GetEffectivenessHistory(gctx, validatorIndex, 7)
		if err != nil {
			return fmt.Errorf("get effectiveness history: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		attestations, err = h.repo.GetAttestationStats(gctx, validatorIndex, 6)
		if err != nil {
			return fmt.Errorf("get attestation stats: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		alerts, err = h.repo.GetRecentAlerts(gctx, validatorIndex, 20)
		if err != nil {
			return fmt.Errorf("get recent alerts: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		timeline, err = h.repo.GetValidatorTimeline(gctx, validatorIndex)
		if err != nil {
			return fmt.Errorf("get validator timeline: %w", err)
		}
		return nil
	})

	// Wait for all queries to complete
	if err := g.Wait(); err != nil {
		h.logger.Error().Err(err).Int64("validator", validatorIndex).Msg("Failed to fetch validator data")
		http.Error(w, "Failed to fetch validator data", http.StatusInternalServerError)
		return
	}

	// Prepare template data
	data := ValidatorPageData{
		Validator:         details,
		EffectivenessData: effectiveness,
		AttestationStats:  attestations,
		Alerts:            alerts,
		Timeline:          timeline,
	}

	// Check if this is an HTMX request (partial update)
	if r.Header.Get("HX-Request") == "true" {
		// Return only the updated fragment (metadata partial)
		h.renderPartial(w, r, data)
		return
	}

	// Full page render
	h.renderFull(w, r, data)
}

// renderFull renders the complete validator detail page
func (h *ValidatorDetailHandler) renderFull(w http.ResponseWriter, r *http.Request, data ValidatorPageData) {
	pageContent := pages.ValidatorDetailPage(data.Validator, data.EffectivenessData, data.AttestationStats, data.Alerts, data.Timeline)
	title := fmt.Sprintf("Validator %d", data.Validator.Index)
	component := layouts.Base(title, pageContent)
	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render validator detail page")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// renderPartial renders only the metadata section for HTMX updates
func (h *ValidatorDetailHandler) renderPartial(w http.ResponseWriter, r *http.Request, data ValidatorPageData) {
	// For HTMX partial updates, render only the metadata component
	component := pages.ValidatorMetadataPartial(data.Validator)
	if err := component.Render(r.Context(), w); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render validator metadata partial")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// HandleSSE provides real-time updates via Server-Sent Events
func (h *ValidatorDetailHandler) HandleSSE(w http.ResponseWriter, r *http.Request) {
	validatorIndexStr := chi.URLParam(r, "index")
	validatorIndex, err := strconv.ParseInt(validatorIndexStr, 10, 64)
	if err != nil {
		h.logger.Error().Err(err).Str("index", validatorIndexStr).Msg("Invalid validator index for SSE")
		http.Error(w, "Invalid validator index", http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		h.logger.Error().Msg("Streaming unsupported")
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	ticker := time.NewTicker(10 * time.Second) // Update every 10 seconds
	defer ticker.Stop()

	h.logger.Info().Int64("validator", validatorIndex).Msg("SSE connection established")

	for {
		select {
		case <-ctx.Done():
			h.logger.Info().Int64("validator", validatorIndex).Msg("SSE connection closed")
			return
		case <-ticker.C:
			// Fetch latest data
			details, err := h.repo.GetValidatorDetails(ctx, validatorIndex)
			if err != nil {
				h.logger.Error().Err(err).Int64("validator", validatorIndex).Msg("SSE: failed to fetch validator details")
				continue
			}

			// Encode as JSON for client consumption
			data, err := json.Marshal(details)
			if err != nil {
				h.logger.Error().Err(err).Msg("SSE: failed to marshal data")
				continue
			}

			// Send SSE event
			fmt.Fprintf(w, "event: validator-update\n")
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// HandleExport exports validator data as CSV or JSON
func (h *ValidatorDetailHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	validatorIndexStr := chi.URLParam(r, "index")
	validatorIndex, err := strconv.ParseInt(validatorIndexStr, 10, 64)
	if err != nil {
		h.logger.Error().Err(err).Str("index", validatorIndexStr).Msg("Invalid validator index for export")
		http.Error(w, "Invalid validator index", http.StatusBadRequest)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	// Fetch comprehensive data for export
	effectiveness, err := h.repo.GetEffectivenessHistory(ctx, validatorIndex, 30)
	if err != nil {
		h.logger.Error().Err(err).Int64("validator", validatorIndex).Msg("Failed to fetch data for export")
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		h.exportCSV(w, validatorIndex, effectiveness)
	case "json":
		h.exportJSON(w, validatorIndex, effectiveness)
	default:
		http.Error(w, "Invalid format. Use 'csv' or 'json'", http.StatusBadRequest)
	}
}

// exportCSV exports effectiveness data as CSV
func (h *ValidatorDetailHandler) exportCSV(w http.ResponseWriter, validatorIndex int64, effectiveness []repository.EffectivenessPoint) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=validator-%d-effectiveness.csv", validatorIndex))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write CSV header
	if err := writer.Write([]string{"Date", "Avg Score", "Min Score", "Max Score"}); err != nil {
		h.logger.Error().Err(err).Msg("Failed to write CSV header")
		return
	}

	// Write data rows
	for _, point := range effectiveness {
		row := []string{
			point.Date.Format("2006-01-02"),
			fmt.Sprintf("%.4f", point.AvgScore),
			fmt.Sprintf("%.4f", point.MinScore),
			fmt.Sprintf("%.4f", point.MaxScore),
		}
		if err := writer.Write(row); err != nil {
			h.logger.Error().Err(err).Msg("Failed to write CSV row")
			return
		}
	}
}

// exportJSON exports effectiveness data as JSON
func (h *ValidatorDetailHandler) exportJSON(w http.ResponseWriter, validatorIndex int64, effectiveness []repository.EffectivenessPoint) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=validator-%d-effectiveness.json", validatorIndex))

	if err := json.NewEncoder(w).Encode(effectiveness); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode JSON export")
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}

// HandleAlertsPartial handles HTMX partial updates for the alerts section
func (h *ValidatorDetailHandler) HandleAlertsPartial(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	validatorIndexStr := chi.URLParam(r, "index")
	validatorIndex, err := strconv.ParseInt(validatorIndexStr, 10, 64)
	if err != nil {
		h.logger.Error().Err(err).Str("index", validatorIndexStr).Msg("Invalid validator index for alerts partial")
		http.Error(w, "Invalid validator index", http.StatusBadRequest)
		return
	}

	alerts, err := h.repo.GetRecentAlerts(ctx, validatorIndex, 20)
	if err != nil {
		h.logger.Error().Err(err).Int64("validator", validatorIndex).Msg("Failed to fetch alerts")
		http.Error(w, "Failed to fetch alerts", http.StatusInternalServerError)
		return
	}

	// Render alerts partial component
	component := pages.AlertHistoryPartial(alerts)
	if err := component.Render(ctx, w); err != nil {
		h.logger.Error().Err(err).Msg("Failed to render alerts partial")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
