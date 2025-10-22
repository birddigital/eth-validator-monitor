package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// DashboardPageHandler handles the dashboard HTML page
type DashboardPageHandler struct{}

// NewDashboardPageHandler creates a new dashboard page handler
func NewDashboardPageHandler() *DashboardPageHandler {
	return &DashboardPageHandler{}
}

// ServeHTTP handles GET /dashboard and renders the dashboard page
func (h *DashboardPageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	component := pages.Dashboard()
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render dashboard", http.StatusInternalServerError)
		return
	}
}
