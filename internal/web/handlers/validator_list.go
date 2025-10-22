package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/birddigital/eth-validator-monitor/internal/database/repository"
	"github.com/birddigital/eth-validator-monitor/internal/services/validators"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/components"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

type ValidatorListHandler struct {
	service *validators.ListService
}

func NewValidatorListHandler(service *validators.ListService) *ValidatorListHandler {
	return &ValidatorListHandler{service: service}
}

// ServeHTTP handles both full page and HTMX partial requests
func (h *ValidatorListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := h.parseFilter(r)

	// Query service
	result, err := h.service.List(r.Context(), filter)
	if err != nil {
		log.Printf("list validators error: %v", err)
		h.renderError(w, r, http.StatusInternalServerError, "Failed to fetch validators")
		return
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		h.renderPartial(w, r, result, filter)
	} else {
		h.renderFull(w, r, result, filter)
	}
}

func (h *ValidatorListHandler) parseFilter(r *http.Request) repository.ValidatorListFilter {
	query := r.URL.Query()

	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit <= 0 {
		limit = 20
	}

	offset, _ := strconv.Atoi(query.Get("offset"))
	if offset < 0 {
		offset = 0
	}

	sortOrder := query.Get("order")
	if sortOrder == "" {
		sortOrder = "asc"
	}

	return repository.ValidatorListFilter{
		Search:    query.Get("search"),
		Status:    query.Get("status"),
		SortBy:    query.Get("sort"),
		SortOrder: sortOrder,
		Limit:     limit,
		Offset:    offset,
	}
}

func (h *ValidatorListHandler) renderPartial(w http.ResponseWriter, r *http.Request, result *repository.ValidatorListResult, filter repository.ValidatorListFilter) {
	// Check which partial to render based on HX-Target header
	target := r.Header.Get("HX-Target")

	switch target {
	case "validator-table-body":
		// Render just table rows (for sort/filter)
		h.renderTableRows(w, r, result, filter)
	case "validator-list-container":
		// Render entire list (for search)
		h.renderTable(w, r, result, filter)
	case "infinite-scroll-trigger":
		// Render appended rows for infinite scroll
		h.renderTableRowsWithTrigger(w, r, result, filter)
	default:
		// Default: render table
		h.renderTable(w, r, result, filter)
	}
}

func (h *ValidatorListHandler) renderFull(w http.ResponseWriter, r *http.Request, result *repository.ValidatorListResult, filter repository.ValidatorListFilter) {
	w.Header().Set("Content-Type", "text/html")

	// Render full page template using templ
	pages := pages.ValidatorList()
	if err := pages.Render(r.Context(), w); err != nil {
		log.Printf("render template error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *ValidatorListHandler) renderTable(w http.ResponseWriter, r *http.Request, result *repository.ValidatorListResult, filter repository.ValidatorListFilter) {
	w.Header().Set("Content-Type", "text/html")

	table := components.ValidatorTable(result, filter)
	if err := table.Render(r.Context(), w); err != nil {
		log.Printf("render table error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *ValidatorListHandler) renderTableRows(w http.ResponseWriter, r *http.Request, result *repository.ValidatorListResult, filter repository.ValidatorListFilter) {
	w.Header().Set("Content-Type", "text/html")

	rows := components.ValidatorTableRowsWithInfiniteScroll(result, filter)
	if err := rows.Render(r.Context(), w); err != nil {
		log.Printf("render rows error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *ValidatorListHandler) renderTableRowsWithTrigger(w http.ResponseWriter, r *http.Request, result *repository.ValidatorListResult, filter repository.ValidatorListFilter) {
	w.Header().Set("Content-Type", "text/html")

	rows := components.ValidatorTableRowsWithInfiniteScroll(result, filter)
	if err := rows.Render(r.Context(), w); err != nil {
		log.Printf("render rows with trigger error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (h *ValidatorListHandler) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	if r.Header.Get("HX-Request") == "true" {
		// HTMX error response
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(status)
		w.Write([]byte(fmt.Sprintf(`<div class="error">%s</div>`, message)))
	} else {
		// JSON error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]string{"error": message})
	}
}

// ServeJSON handles JSON API endpoint (for non-HTMX clients)
func (h *ValidatorListHandler) ServeJSON(w http.ResponseWriter, r *http.Request) {
	filter := h.parseFilter(r)

	result, err := h.service.List(r.Context(), filter)
	if err != nil {
		log.Printf("list validators error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to fetch validators"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=30") // 30s cache
	json.NewEncoder(w).Encode(result)
}
