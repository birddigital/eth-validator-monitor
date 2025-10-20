package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// RegisterHandler handles GET /register requests
type RegisterHandler struct{}

// NewRegisterHandler creates a new register handler
func NewRegisterHandler() *RegisterHandler {
	return &RegisterHandler{}
}

// ServeHTTP implements http.Handler for GET requests
func (h *RegisterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract success or error message from query parameter (if any)
	successMsg := r.URL.Query().Get("success")
	errorMsg := r.URL.Query().Get("error")

	data := pages.RegisterPageData{
		SuccessMessage: successMsg,
		ErrorMessage:   errorMsg,
		FormData:       make(map[string]string),
	}

	// Render the template
	component := pages.RegisterPageWithLayout(data)

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
