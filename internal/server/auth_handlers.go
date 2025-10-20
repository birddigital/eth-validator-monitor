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
	Username string   `json:"username"`
	Password string   `json:"password"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles,omitempty"` // Optional, defaults to ["user"]
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

// ErrorResponse is the standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// Register handles POST /api/auth/register
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" || req.Email == "" {
		respondError(w, "Username, password, and email are required", http.StatusBadRequest)
		return
	}

	// Register user
	user, err := h.authService.Register(r.Context(), req.Username, req.Password, req.Email, req.Roles)
	if err != nil {
		// Handle specific errors
		switch err {
		case auth.ErrPasswordTooShort:
			respondError(w, err.Error(), http.StatusBadRequest)
		default:
			if err.Error() == "user already exists" {
				respondError(w, "Username or email already exists", http.StatusConflict)
			} else {
				respondError(w, "Registration failed", http.StatusInternalServerError)
			}
		}
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
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		respondError(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Authenticate user
	user, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			respondError(w, "Invalid username or password", http.StatusUnauthorized)
		} else {
			respondError(w, "Login failed", http.StatusInternalServerError)
		}
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
