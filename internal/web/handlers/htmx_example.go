package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/graph/middleware"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// HTMXExampleHandler demonstrates HTMX content negotiation
// Returns full page for normal requests, partial HTML for HTMX requests
type HTMXExampleHandler struct{}

// NewHTMXExampleHandler creates a new HTMX example handler
func NewHTMXExampleHandler() *HTMXExampleHandler {
	return &HTMXExampleHandler{}
}

// ServeHTTP handles both full-page requests and HTMX partial requests
func (h *HTMXExampleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Example data structure
	data := pages.HomePageData{
		ValidatorCount:    150,
		ActiveValidators:  148,
		TotalBalance:      4800.0,
		AvgEffectiveness:  98.5,
	}

	// Check if this is an HTMX request
	if middleware.IsHTMXRequest(r.Context()) {
		// Return only the partial HTML fragment
		// This would be a templ component specifically designed for partial updates
		// For now, we'll demonstrate with the existing HomePage component (without layout)

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Render just the content without the layout wrapper
		// In a real implementation, you'd have separate templ components for:
		// - Full page with layout: pages.HomePageWithLayout(data)
		// - Just the content fragment: pages.HomePage(data)
		pages.HomePage(data).Render(r.Context(), w)
		return
	}

	// For non-HTMX requests, return the full HTML page with layout
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pages.HomePageWithLayout(data).Render(r.Context(), w)
}
