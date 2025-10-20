package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// HomeHandler handles GET / requests
type HomeHandler struct {
	// Dependencies for fetching validator data
	// TODO: Add storage.Storage when ready to fetch real data
}

// NewHomeHandler creates a new home handler
func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

// ServeHTTP implements http.Handler
func (h *HomeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Fetch real data from database/cache
	// For now, using placeholder data
	data := pages.HomePageData{
		ValidatorCount:   150,
		ActiveValidators: 142,
		TotalBalance:     4800.50,
		AvgEffectiveness: 98.7,
	}

	// Render the template
	component := pages.HomePageWithLayout(data)

	// Use templ's Render method with the request context
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
