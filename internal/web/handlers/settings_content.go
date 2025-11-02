package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/components"
)

// SettingsContentHandler handles the dynamic content loading for settings tabs
type SettingsContentHandler struct {
	// Add dependencies as needed
}

// NewSettingsContentHandler creates a new settings content handler
func NewSettingsContentHandler() *SettingsContentHandler {
	return &SettingsContentHandler{}
}

// ServeHTTP handles GET /api/settings/content
func (h *SettingsContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the tab from query parameter
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "profile"
	}

	// Get user info from session context
	username := ""
	if userID := r.Context().Value(auth.SessionUsernameKey); userID != nil {
		if u, ok := userID.(string); ok {
			username = u
		}
	}

	// Set content type to HTML for HTMX
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render the appropriate component based on the tab
	var err error
	switch tab {
	case "profile":
		err = components.SettingsProfileTab(username).Render(r.Context(), w)
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
		err = components.SettingsProfileTab(username).Render(r.Context(), w)
	}

	if err != nil {
		http.Error(w, "Failed to render settings content", http.StatusInternalServerError)
		return
	}
}
