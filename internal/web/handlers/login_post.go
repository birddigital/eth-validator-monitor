package handlers

import (
	"net/http"

	formflow "github.com/birddigital/formflow/go-formflow"
	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// LoginPostHandler handles POST /login form submissions using formflow
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

// ServeHTTP implements http.Handler for POST form submissions with formflow
func (h *LoginPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = "/dashboard"
	}

	// Define login form declaratively using formflow
	form := formflow.NewForm("login-form").
		AddField(formflow.Field{
			Name:            "email",
			Type:            "email",
			Label:           "Email",
			Required:        true,
			ValidationRules: "required,email",
		}).
		AddField(formflow.Field{
			Name:            "password",
			Type:            "password",
			Label:           "Password",
			Required:        true,
			ValidationRules: "required,min=8",
		}).
		OnSuccess(func(data map[string]interface{}) error {
			// Extract form data
			email := data["email"].(string)
			password := data["password"].(string)

			// Authenticate user
			user, err := h.authService.LoginByEmail(r.Context(), email, password)
			if err != nil {
				return err
			}

			// Create session
			session, err := h.sessionStore.Get(r)
			if err != nil {
				return err
			}

			h.sessionStore.SetUserSession(session, user.ID, user.Username)

			if err := h.sessionStore.Save(r, w, session); err != nil {
				return err
			}

			return nil
		}).
		Build()

	// Validate request using formflow
	data, errors := form.ValidateRequest(r)
	if len(errors) > 0 {
		// For HTMX requests, formflow handles partial rendering automatically
		// For traditional requests, re-render full form
		if formflow.IsHTMXRequest(r) {
			form.RenderError(w, r, errors, data)
		} else {
			h.renderErrorPage(w, r, aggregateErrors(errors), redirectURL)
		}
		return
	}

	// Execute success handler (authentication + session creation)
	if err := form.OnSuccess(data); err != nil {
		// Handle authentication errors
		if err == auth.ErrInvalidCredentials {
			errors := formflow.ValidationErrors{
				"_form": "Invalid email or password",
			}
			if formflow.IsHTMXRequest(r) {
				form.RenderError(w, r, errors, data)
			} else {
				h.renderErrorPage(w, r, "Invalid email or password", redirectURL)
			}
			return
		}

		// Other errors (session, network, etc.)
		errors := formflow.ValidationErrors{
			"_form": "Login failed. Please try again.",
		}
		if formflow.IsHTMXRequest(r) {
			form.RenderError(w, r, errors, data)
		} else {
			h.renderErrorPage(w, r, "Login failed. Please try again.", redirectURL)
		}
		return
	}

	// Success - redirect using formflow
	form.RenderSuccess(w, r, redirectURL)
}

// renderErrorPage renders the full login page with error (for non-HTMX requests)
func (h *LoginPostHandler) renderErrorPage(w http.ResponseWriter, r *http.Request, errorMsg, redirectURL string) {
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

// aggregateErrors combines multiple field errors into a single message
func aggregateErrors(errors formflow.ValidationErrors) string {
	if msg, ok := errors["_form"]; ok {
		return msg
	}
	for _, msg := range errors {
		return msg // Return first error
	}
	return "Validation failed"
}
