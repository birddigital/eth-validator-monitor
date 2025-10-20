package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/web/templates/components"
)

// ExampleHandler demonstrates templ component rendering in an HTTP handler
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user name from query parameter or use default
	userName := r.URL.Query().Get("name")
	if userName == "" {
		userName = "Validator Operator"
	}

	// Render templ component
	component := components.HelloWorld(userName)

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Write to response
	err := component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render template", http.StatusInternalServerError)
		return
	}
}
