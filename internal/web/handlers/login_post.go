package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// LoginPostHandler handles POST /login form submissions
type LoginPostHandler struct {
	authService  *auth.Service
	sessionStore *auth.SessionStore
}

// NewLoginPostHandler creates a new login POST handler
func NewLoginPostHandler(authService *auth.Service, sessionStore *auth.SessionStore) *LoginPostHandler {
	return &LoginPostHandler{
		authService:  authService,
		sessionStore: sessionStore,
	}
}

// ServeHTTP implements http.Handler for POST form submissions
func (h *LoginPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, "Invalid form data", "")
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	redirectURL := r.FormValue("redirect")

	// Validate required fields
	if email == "" || password == "" {
		h.renderError(w, r, "Email and password are required", redirectURL)
		return
	}

	// Authenticate user using email (since form uses email field)
	// Note: auth.Service.Login() expects username, so we'll use LoginByEmail
	// which we'll add to the auth service
	user, err := h.authService.LoginByEmail(r.Context(), email, password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			h.renderError(w, r, "Invalid email or password", redirectURL)
		} else {
			h.renderError(w, r, "Login failed. Please try again.", redirectURL)
		}
		return
	}

	// Create session
	session, err := h.sessionStore.Get(r)
	if err != nil {
		h.renderError(w, r, "Session error. Please try again.", redirectURL)
		return
	}

	h.sessionStore.SetUserSession(session, user.ID, user.Username)

	if err := h.sessionStore.Save(r, w, session); err != nil {
		h.renderError(w, r, "Failed to save session. Please try again.", redirectURL)
		return
	}

	// Redirect to dashboard or specified URL
	target := "/dashboard"
	if redirectURL != "" {
		target = redirectURL
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// renderError renders the login page with an error message
func (h *LoginPostHandler) renderError(w http.ResponseWriter, r *http.Request, errorMsg, redirectURL string) {
	data := pages.LoginPageData{
		ErrorMessage: errorMsg,
		RedirectURL:  redirectURL,
	}

	w.WriteHeader(http.StatusUnauthorized)
	component := pages.LoginPageWithLayout(data)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
