package server

import (
	"encoding/json"
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
)

// AuthHandlers handles HTTP authentication endpoints
type AuthHandlers struct {
	authService  *auth.Service
	sessionStore *auth.SessionStore
}

// NewAuthHandlers creates new authentication handlers
func NewAuthHandlers(authService *auth.Service, sessionStore *auth.SessionStore) *AuthHandlers {
	return &AuthHandlers{
		authService:  authService,
		sessionStore: sessionStore,
	}
}

// RegisterRequest is the request body for user registration
type RegisterRequest struct {
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	ConfirmPassword string   `json:"confirmPassword"`
	Email           string   `json:"email"`
	Roles           []string `json:"roles,omitempty"` // Optional, defaults to ["user"]
}

// LoginRequest is the request body for user login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserResponse is the response body for authenticated user info
type UserResponse struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// ErrorResponse is the standard error response with optional field-level errors
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"` // Field-specific errors
}

// Register handles POST /api/auth/register
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondValidationError(w, "Invalid request body", map[string]string{"body": "invalid JSON"}, http.StatusBadRequest)
		return
	}

	// Register user (validation happens in service layer)
	user, err := h.authService.Register(r.Context(), req.Username, req.Password, req.ConfirmPassword, req.Email, req.Roles)
	if err != nil {
		// Handle validation errors with field-level details
		if verr, ok := err.(*auth.ValidationError); ok {
			respondValidationError(w, "Validation failed", verr.Fields, http.StatusBadRequest)
			return
		}

		// Handle duplicate user error
		if err == auth.ErrUserAlreadyExists {
			respondValidationError(w, "User already exists", map[string]string{
				"username": "username or email already exists",
			}, http.StatusConflict)
			return
		}

		// Handle other errors
		respondError(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	// Create session for new user
	session, err := h.sessionStore.Get(r)
	if err != nil {
		respondError(w, "Session error", http.StatusInternalServerError)
		return
	}

	h.sessionStore.SetUserSession(session, user.ID, user.Username)

	if err := h.sessionStore.Save(r, w, session); err != nil {
		respondError(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return user info
	respondJSON(w, UserResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
	}, http.StatusCreated)
}

// Login handles POST /api/auth/login
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondValidationError(w, "Invalid request body", map[string]string{"body": "invalid JSON"}, http.StatusBadRequest)
		return
	}

	// Authenticate user (validation happens in service layer)
	user, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		// Handle validation errors
		if verr, ok := err.(*auth.ValidationError); ok {
			respondValidationError(w, "Validation failed", verr.Fields, http.StatusBadRequest)
			return
		}

		// Handle invalid credentials (don't reveal if user exists)
		if err == auth.ErrInvalidCredentials {
			respondValidationError(w, "Invalid credentials", map[string]string{
				"credentials": "invalid username or password",
			}, http.StatusUnauthorized)
			return
		}

		// Handle other errors
		respondError(w, "Login failed", http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := h.sessionStore.Get(r)
	if err != nil {
		respondError(w, "Session error", http.StatusInternalServerError)
		return
	}

	h.sessionStore.SetUserSession(session, user.ID, user.Username)

	if err := h.sessionStore.Save(r, w, session); err != nil {
		respondError(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Return user info
	respondJSON(w, UserResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
	}, http.StatusOK)
}

// Logout handles POST /api/auth/logout
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessionStore.Get(r)
	if err != nil {
		respondError(w, "Session error", http.StatusInternalServerError)
		return
	}

	h.sessionStore.Destroy(session)

	if err := h.sessionStore.Save(r, w, session); err != nil {
		respondError(w, "Failed to clear session", http.StatusInternalServerError)
		return
	}

	respondJSON(w, map[string]string{"message": "Logged out successfully"}, http.StatusOK)
}

// Me handles GET /api/auth/me - returns current authenticated user
func (h *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetSessionUserIDFromContext(r.Context())
	if !ok {
		respondError(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	// Fetch full user details
	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		respondError(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	respondJSON(w, UserResponse{
		ID:       user.ID.String(),
		Username: user.Username,
		Email:    user.Email,
		Roles:    user.Roles,
	}, http.StatusOK)
}

// Helper functions for consistent JSON responses

func respondJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, statusCode int) {
	respondJSON(w, ErrorResponse{Error: message}, statusCode)
}

func respondValidationError(w http.ResponseWriter, message string, fields map[string]string, statusCode int) {
	respondJSON(w, ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Fields:  fields,
	}, statusCode)
}
