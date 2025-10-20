package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// LoginHandler handles GET /login requests
type LoginHandler struct {
	// TODO: Add auth service dependency when implementing POST handler
}

// NewLoginHandler creates a new login handler
func NewLoginHandler() *LoginHandler {
	return &LoginHandler{}
}

// ServeHTTP implements http.Handler for GET requests
func (h *LoginHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract error message from query parameter (if any)
	errorMsg := r.URL.Query().Get("error")

	// Extract redirect URL from query parameter (for post-login redirect)
	redirectURL := r.URL.Query().Get("redirect")

	data := pages.LoginPageData{
		ErrorMessage: errorMsg,
		RedirectURL:  redirectURL,
	}

	// Render the template
	component := pages.LoginPageWithLayout(data)

	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
