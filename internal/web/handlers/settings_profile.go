package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/google/uuid"
)

// SettingsProfileHandler handles profile update requests
type SettingsProfileHandler struct {
	userRepo  *storage.UserRepository
	validator *auth.Validator
}

// NewSettingsProfileHandler creates a new settings profile handler
func NewSettingsProfileHandler(userRepo *storage.UserRepository, validator *auth.Validator) *SettingsProfileHandler {
	return &SettingsProfileHandler{
		userRepo:  userRepo,
		validator: validator,
	}
}

// ProfileUpdateRequest represents the profile update form data
type ProfileUpdateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ProfileUpdateResponse represents the response after profile update
type ProfileUpdateResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// ServeHTTP handles POST /api/settings/profile
func (h *SettingsProfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from session context
	userID, ok := auth.GetSessionUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	var req ProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, "Invalid request format", nil)
		return
	}

	// Trim whitespace
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)

	// Validate input
	validationErrors := make(map[string]string)

	if err := h.validator.ValidateUsername(req.Username); err != nil {
		if ve, ok := err.(*auth.ValidationError); ok {
			for field, msg := range ve.Fields {
				validationErrors[field] = msg
			}
		}
	}

	if err := h.validator.ValidateEmail(req.Email); err != nil {
		if ve, ok := err.(*auth.ValidationError); ok {
			for field, msg := range ve.Fields {
				validationErrors[field] = msg
			}
		}
	}

	if len(validationErrors) > 0 {
		h.sendErrorResponse(w, "Validation failed", validationErrors)
		return
	}

	// Update profile in database
	err := h.userRepo.UpdateProfile(r.Context(), userID, req.Username, req.Email)
	if err != nil {
		if err == storage.ErrUserAlreadyExists {
			validationErrors["username"] = "Username or email already taken"
			h.sendErrorResponse(w, "Username or email already exists", validationErrors)
			return
		}
		h.sendErrorResponse(w, "Failed to update profile", nil)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ProfileUpdateResponse{
		Success: true,
		Message: "Profile updated successfully",
	})
}

func (h *SettingsProfileHandler) sendErrorResponse(w http.ResponseWriter, message string, errors map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ProfileUpdateResponse{
		Success: false,
		Message: message,
		Errors:  errors,
	})
}

// PasswordChangeHandler handles password change requests
type PasswordChangeHandler struct {
	userRepo  *storage.UserRepository
	validator *auth.Validator
}

// NewPasswordChangeHandler creates a new password change handler
func NewPasswordChangeHandler(userRepo *storage.UserRepository, validator *auth.Validator) *PasswordChangeHandler {
	return &PasswordChangeHandler{
		userRepo:  userRepo,
		validator: validator,
	}
}

// PasswordChangeRequest represents the password change form data
type PasswordChangeRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

// PasswordChangeResponse represents the response after password change
type PasswordChangeResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

// ServeHTTP handles POST /api/settings/password
func (h *PasswordChangeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from session context
	userID, ok := auth.GetSessionUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	var req PasswordChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendErrorResponse(w, "Invalid request format", nil)
		return
	}

	// Validate input
	validationErrors := make(map[string]string)

	// Validate current password is provided
	if strings.TrimSpace(req.CurrentPassword) == "" {
		validationErrors["currentPassword"] = "Current password is required"
	}

	// Validate new password strength
	if err := h.validator.ValidatePassword(req.NewPassword); err != nil {
		if ve, ok := err.(*auth.ValidationError); ok {
			for field, msg := range ve.Fields {
				validationErrors[field] = msg
			}
		}
	}

	// Validate password confirmation
	if err := h.validator.ValidatePasswordMatch(req.NewPassword, req.ConfirmPassword); err != nil {
		if ve, ok := err.(*auth.ValidationError); ok {
			for field, msg := range ve.Fields {
				validationErrors[field] = msg
			}
		}
	}

	if len(validationErrors) > 0 {
		h.sendErrorResponse(w, "Validation failed", validationErrors)
		return
	}

	// Get user from database to verify current password
	user, err := h.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		h.sendErrorResponse(w, "Failed to verify current password", nil)
		return
	}

	// Verify current password
	if err := auth.VerifyPassword(user.PasswordHash, req.CurrentPassword); err != nil {
		validationErrors["currentPassword"] = "Current password is incorrect"
		h.sendErrorResponse(w, "Current password is incorrect", validationErrors)
		return
	}

	// Hash new password
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		h.sendErrorResponse(w, "Failed to update password", nil)
		return
	}

	// Update password in database
	err = h.userRepo.UpdatePassword(r.Context(), userID, newPasswordHash)
	if err != nil {
		h.sendErrorResponse(w, "Failed to update password", nil)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PasswordChangeResponse{
		Success: true,
		Message: "Password updated successfully",
	})
}

func (h *PasswordChangeHandler) sendErrorResponse(w http.ResponseWriter, message string, errors map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(PasswordChangeResponse{
		Success: false,
		Message: message,
		Errors:  errors,
	})
}
