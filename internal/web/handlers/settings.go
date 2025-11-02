package handlers

import (
	"net/http"

	"github.com/birddigital/eth-validator-monitor/internal/auth"
	"github.com/birddigital/eth-validator-monitor/internal/web/templates/pages"
)

// SettingsHandler handles the settings page
type SettingsHandler struct {
	// Add dependencies as needed (e.g., user service, etc.)
}

// NewSettingsHandler creates a new settings handler
func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

// ServeHTTP handles GET /settings
func (h *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the active tab from query parameter
	activeTab := r.URL.Query().Get("tab")
	if activeTab == "" {
		activeTab = "profile" // Default to profile tab
	}

	// Validate tab name to prevent XSS
	validTabs := map[string]bool{
		"profile":         true,
		"notifications":   true,
		"api-keys":        true,
		"ui-preferences":  true,
		"2fa":             true,
		"sessions":        true,
		"account":         true,
	}

	if !validTabs[activeTab] {
		activeTab = "profile"
	}

	// Get user info from session context
	username := ""
	email := ""
	if userID := r.Context().Value(auth.SessionUsernameKey); userID != nil {
		if u, ok := userID.(string); ok {
			username = u
			// TODO: Fetch email from user service when available
			email = username + "@example.com" // Placeholder
		}
	}

	// Render the settings page
	data := pages.SettingsPageData{
		ActiveTab: activeTab,
		Username:  username,
		UserEmail: email,
	}

	component := pages.SettingsPageWithLayout(data)
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render settings page", http.StatusInternalServerError)
		return
	}
}
