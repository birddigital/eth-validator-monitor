package handlers

import (
	"net/http"
	"regexp"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// RegisterPostHandler handles POST /register form submissions
type RegisterPostHandler struct {
	authService  *auth.Service
	sessionStore *auth.SessionStore
}

// NewRegisterPostHandler creates a new registration POST handler
func NewRegisterPostHandler(authService *auth.Service, sessionStore *auth.SessionStore) *RegisterPostHandler {
	return &RegisterPostHandler{
		authService:  authService,
		sessionStore: sessionStore,
	}
}

// ServeHTTP implements http.Handler for POST form submissions
func (h *RegisterPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.renderError(w, r, "Invalid form data", nil)
		return
	}

	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")
	passwordConfirm := r.FormValue("password_confirm")
	terms := r.FormValue("terms")

	// Store form data for repopulation on error
	formData := map[string]string{
		"email":    email,
		"username": username,
	}

	// Validate required fields
	if email == "" || username == "" || password == "" || passwordConfirm == "" {
		h.renderError(w, r, "All fields are required", formData)
		return
	}

	// Validate terms accepted
	if terms != "on" {
		h.renderError(w, r, "You must accept the Terms of Service and Privacy Policy", formData)
		return
	}

	// Validate email format
	if !isValidEmail(email) {
		h.renderError(w, r, "Invalid email address", formData)
		return
	}

	// Validate username (3-20 chars, alphanumeric)
	if len(username) < 3 || len(username) > 20 {
		h.renderError(w, r, "Username must be 3-20 characters", formData)
		return
	}
	if !isAlphanumeric(username) {
		h.renderError(w, r, "Username must contain only letters and numbers", formData)
		return
	}

	// Validate password match
	if password != passwordConfirm {
		h.renderError(w, r, "Passwords do not match", formData)
		return
	}

	// Validate password length (minimum 8 characters)
	if len(password) < 8 {
		h.renderError(w, r, "Password must be at least 8 characters", formData)
		return
	}

	// Register user (service signature: username, password, confirmPassword, email, roles)
	user, err := h.authService.Register(r.Context(), username, password, passwordConfirm, email, []string{"user"})
	if err != nil {
		// Handle validation errors with field-level details
		if verr, ok := err.(*auth.ValidationError); ok {
			// Aggregate all field errors into a single message for the web form
			errorMsg := ""
			for field, msg := range verr.Fields {
				if errorMsg != "" {
					errorMsg += "; "
				}
				errorMsg += field + ": " + msg
			}
			h.renderError(w, r, errorMsg, formData)
			return
		}

		if err == auth.ErrUserAlreadyExists {
			h.renderError(w, r, "Email or username already exists", formData)
		} else if err == auth.ErrPasswordTooShort {
			h.renderError(w, r, err.Error(), formData)
		} else {
			h.renderError(w, r, "Registration failed. Please try again.", formData)
		}
		return
	}

	// Create session for new user
	session, err := h.sessionStore.Get(r)
	if err != nil {
		h.renderError(w, r, "Session error. Please try again.", formData)
		return
	}

	h.sessionStore.SetUserSession(session, user.ID, user.Username)

	if err := h.sessionStore.Save(r, w, session); err != nil {
		h.renderError(w, r, "Failed to save session. Please try again.", formData)
		return
	}

	// Redirect to dashboard on success
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// renderError renders the registration page with an error message
func (h *RegisterPostHandler) renderError(w http.ResponseWriter, r *http.Request, errorMsg string, formData map[string]string) {
	if formData == nil {
		formData = make(map[string]string)
	}

	data := pages.RegisterPageData{
		ErrorMessage: errorMsg,
		FormData:     formData,
	}

	w.WriteHeader(http.StatusBadRequest)
	component := pages.RegisterPageWithLayout(data)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// isValidEmail validates email format using regex
func isValidEmail(email string) bool {
	// Simple email regex pattern
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// isAlphanumeric checks if string contains only letters and numbers
func isAlphanumeric(s string) bool {
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return alphanumericRegex.MatchString(s)
}
