package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/storage"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/components"
)

// SettingsContentHandler handles the dynamic content loading for settings tabs
type SettingsContentHandler struct {
	userRepo *storage.UserRepository
}

// NewSettingsContentHandler creates a new settings content handler
func NewSettingsContentHandler(userRepo *storage.UserRepository) *SettingsContentHandler {
	return &SettingsContentHandler{
		userRepo: userRepo,
	}
}

// ServeHTTP handles GET /api/settings/content
func (h *SettingsContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the tab from query parameter
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "profile"
	}

	// Get user ID from session context
	userID, ok := auth.GetSessionUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch user from database
	user, err := h.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to load user profile", http.StatusInternalServerError)
		return
	}

	// Set content type to HTML for HTMX
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render the appropriate component based on the tab
	switch tab {
	case "profile":
		err = components.SettingsProfileTab(user.Username, user.Email).Render(r.Context(), w)
	case "notifications":
		err = components.SettingsNotificationsTab().Render(r.Context(), w)
	case "api-keys":
		err = components.SettingsAPIKeysTab().Render(r.Context(), w)
	case "ui-preferences":
		err = components.SettingsUIPreferencesTab().Render(r.Context(), w)
	case "2fa":
		err = components.Settings2FATab().Render(r.Context(), w)
	case "sessions":
		err = components.SettingsSessionsTab().Render(r.Context(), w)
	case "account":
		err = components.SettingsAccountTab().Render(r.Context(), w)
	default:
		// Default to profile tab
		err = components.SettingsProfileTab(user.Username, user.Email).Render(r.Context(), w)
	}

	if err != nil {
		http.Error(w, "Failed to render settings content", http.StatusInternalServerError)
		return
	}
}
