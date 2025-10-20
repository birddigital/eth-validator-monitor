package handlers

import (
	"fmt"
	"net/http"

	formflow "github.com/birddigital/formflow/go-formflow"
	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// RegisterPostHandler handles POST /register form submissions using formflow
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

// ServeHTTP implements http.Handler for POST form submissions with formflow
func (h *RegisterPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var passwordValue string // Captured for password confirmation validation

	// Define registration form declaratively using formflow
	form := formflow.NewForm("register-form").
		AddField(formflow.Field{
			Name:            "email",
			Type:            "email",
			Label:           "Email",
			Required:        true,
			ValidationRules: "required,email",
		}).
		AddField(formflow.Field{
			Name:            "username",
			Type:            "text",
			Label:           "Username",
			Required:        true,
			ValidationRules: "required,min=3,max=20,alphanum",
		}).
		AddField(formflow.Field{
			Name:            "password",
			Type:            "password",
			Label:           "Password",
			Required:        true,
			ValidationRules: "required,min=8",
			CustomValidator: func(value string) error {
				passwordValue = value // Capture for confirmation check
				return nil
			},
		}).
		AddField(formflow.Field{
			Name:     "password_confirm",
			Type:     "password",
			Label:    "Confirm Password",
			Required: true,
			CustomValidator: func(value string) error {
				return formflow.ValidatePasswordMatch(passwordValue, value)
			},
		}).
		AddField(formflow.Field{
			Name:     "terms",
			Type:     "checkbox",
			Label:    "Terms of Service",
			Required: true,
		}).
		OnSuccess(func(data map[string]interface{}) error {
			// Extract form data
			email := data["email"].(string)
			username := data["username"].(string)
			password := data["password"].(string)
			passwordConfirm := data["password_confirm"].(string)

			// Register user
			user, err := h.authService.Register(r.Context(), username, password, passwordConfirm, email, []string{"user"})
			if err != nil {
				return err
			}

			// Create session for new user
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
		// For traditional requests, re-render full form with errors
		if formflow.IsHTMXRequest(r) {
			form.RenderError(w, r, errors, data)
		} else {
			h.renderErrorPage(w, r, aggregateErrors(errors), extractFormData(data))
		}
		return
	}

	// Execute success handler (registration + session creation)
	if err := form.OnSuccess(data); err != nil {
		// Handle validation errors from auth service
		if verr, ok := err.(*auth.ValidationError); ok {
			errors := make(formflow.ValidationErrors)
			for field, msg := range verr.Fields {
				errors[field] = msg
			}

			if formflow.IsHTMXRequest(r) {
				form.RenderError(w, r, errors, data)
			} else {
				h.renderErrorPage(w, r, aggregateAuthErrors(verr), extractFormData(data))
			}
			return
		}

		// Handle specific auth errors
		if err == auth.ErrUserAlreadyExists {
			errors := formflow.ValidationErrors{
				"_form": "Email or username already exists",
			}
			if formflow.IsHTMXRequest(r) {
				form.RenderError(w, r, errors, data)
			} else {
				h.renderErrorPage(w, r, "Email or username already exists", extractFormData(data))
			}
			return
		}

		if err == auth.ErrPasswordTooShort {
			errors := formflow.ValidationErrors{
				"password": err.Error(),
			}
			if formflow.IsHTMXRequest(r) {
				form.RenderError(w, r, errors, data)
			} else {
				h.renderErrorPage(w, r, err.Error(), extractFormData(data))
			}
			return
		}

		// Generic error
		errors := formflow.ValidationErrors{
			"_form": "Registration failed. Please try again.",
		}
		if formflow.IsHTMXRequest(r) {
			form.RenderError(w, r, errors, data)
		} else {
			h.renderErrorPage(w, r, "Registration failed. Please try again.", extractFormData(data))
		}
		return
	}

	// Success - redirect to dashboard using formflow
	form.RenderSuccess(w, r, "/dashboard")
}

// renderErrorPage renders the full registration page with error (for non-HTMX requests)
func (h *RegisterPostHandler) renderErrorPage(w http.ResponseWriter, r *http.Request, errorMsg string, formData map[string]string) {
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

// extractFormData converts map[string]interface{} to map[string]string for template
// Excludes password fields for security
func extractFormData(data map[string]interface{}) map[string]string {
	formData := make(map[string]string)
	for key, val := range data {
		// Don't repopulate password fields
		if key == "password" || key == "password_confirm" {
			continue
		}
		if strVal, ok := val.(string); ok {
			formData[key] = strVal
		}
	}
	return formData
}

// aggregateAuthErrors combines multiple auth service validation errors into a single message
func aggregateAuthErrors(verr *auth.ValidationError) string {
	errorMsg := ""
	for field, msg := range verr.Fields {
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += fmt.Sprintf("%s: %s", field, msg)
	}
	return errorMsg
}
