package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTMXMiddleware_DetectsHTMXRequest(t *testing.T) {
	middleware := NewHTMXMiddleware()

	tests := []struct {
		name          string
		hxRequest     string
		expectedHTMX  bool
	}{
		{
			name:         "HTMX request with HX-Request: true",
			hxRequest:    "true",
			expectedHTMX: true,
		},
		{
			name:         "Non-HTMX request without header",
			hxRequest:    "",
			expectedHTMX: false,
		},
		{
			name:         "Non-HTMX request with HX-Request: false",
			hxRequest:    "false",
			expectedHTMX: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that checks context
			var capturedContext context.Context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedContext = r.Context()
			})

			// Wrap with middleware
			wrapped := middleware.Middleware(handler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.hxRequest != "" {
				req.Header.Set("HX-Request", tt.hxRequest)
			}
			w := httptest.NewRecorder()

			// Execute
			wrapped.ServeHTTP(w, req)

			// Verify
			if IsHTMXRequest(capturedContext) != tt.expectedHTMX {
				t.Errorf("IsHTMXRequest() = %v, want %v", IsHTMXRequest(capturedContext), tt.expectedHTMX)
			}
		})
	}
}

func TestHTMXMiddleware_CapturesHeaders(t *testing.T) {
	middleware := NewHTMXMiddleware()

	tests := []struct {
		name            string
		headers         map[string]string
		expectedTrigger string
		expectedTarget  string
		expectedPrompt  string
		hasTrigger      bool
		hasTarget       bool
		hasPrompt       bool
	}{
		{
			name: "All HTMX headers present",
			headers: map[string]string{
				"HX-Request": "true",
				"HX-Trigger": "button-click",
				"HX-Target":  "main-content",
				"HX-Prompt":  "Are you sure?",
			},
			expectedTrigger: "button-click",
			expectedTarget:  "main-content",
			expectedPrompt:  "Are you sure?",
			hasTrigger:      true,
			hasTarget:       true,
			hasPrompt:       true,
		},
		{
			name: "Only HX-Trigger present",
			headers: map[string]string{
				"HX-Request": "true",
				"HX-Trigger": "refresh",
			},
			expectedTrigger: "refresh",
			hasTrigger:      true,
			hasTarget:       false,
			hasPrompt:       false,
		},
		{
			name: "Non-HTMX request ignores headers",
			headers: map[string]string{
				"HX-Trigger": "should-be-ignored",
			},
			hasTrigger: false,
			hasTarget:  false,
			hasPrompt:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedContext context.Context
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedContext = r.Context()
			})

			wrapped := middleware.Middleware(handler)

			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			// Check trigger
			trigger, ok := HTMXTrigger(capturedContext)
			if ok != tt.hasTrigger {
				t.Errorf("HTMXTrigger() ok = %v, want %v", ok, tt.hasTrigger)
			}
			if tt.hasTrigger && trigger != tt.expectedTrigger {
				t.Errorf("HTMXTrigger() = %v, want %v", trigger, tt.expectedTrigger)
			}

			// Check target
			target, ok := HTMXTarget(capturedContext)
			if ok != tt.hasTarget {
				t.Errorf("HTMXTarget() ok = %v, want %v", ok, tt.hasTarget)
			}
			if tt.hasTarget && target != tt.expectedTarget {
				t.Errorf("HTMXTarget() = %v, want %v", target, tt.expectedTarget)
			}

			// Check prompt
			prompt, ok := HTMXPrompt(capturedContext)
			if ok != tt.hasPrompt {
				t.Errorf("HTMXPrompt() ok = %v, want %v", ok, tt.hasPrompt)
			}
			if tt.hasPrompt && prompt != tt.expectedPrompt {
				t.Errorf("HTMXPrompt() = %v, want %v", prompt, tt.expectedPrompt)
			}
		})
	}
}

func TestWantsJSON(t *testing.T) {
	tests := []struct {
		name        string
		acceptHeader string
		wantsJSON   bool
	}{
		{
			name:         "Prefers JSON",
			acceptHeader: "application/json",
			wantsJSON:    true,
		},
		{
			name:         "Prefers HTML",
			acceptHeader: "text/html",
			wantsJSON:    false,
		},
		{
			name:         "Prefers HTML over JSON",
			acceptHeader: "text/html,application/json;q=0.9",
			wantsJSON:    false,
		},
		{
			name:         "Prefers JSON over HTML",
			acceptHeader: "application/json,text/html;q=0.9",
			wantsJSON:    true,
		},
		{
			name:         "Empty Accept header",
			acceptHeader: "",
			wantsJSON:    false,
		},
		{
			name:         "Wildcard Accept",
			acceptHeader: "*/*",
			wantsJSON:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			result := WantsJSON(req)
			if result != tt.wantsJSON {
				t.Errorf("WantsJSON() = %v, want %v", result, tt.wantsJSON)
			}
		})
	}
}

func TestSetHTMXResponse(t *testing.T) {
	tests := []struct {
		name            string
		trigger         string
		retarget        string
		reswap          string
		expectedHeaders map[string]string
	}{
		{
			name:     "All headers set",
			trigger:  "item-updated",
			retarget: "#main",
			reswap:   "outerHTML",
			expectedHeaders: map[string]string{
				"HX-Trigger":  "item-updated",
				"HX-Retarget": "#main",
				"HX-Reswap":   "outerHTML",
			},
		},
		{
			name:    "Only trigger set",
			trigger: "notification",
			expectedHeaders: map[string]string{
				"HX-Trigger": "notification",
			},
		},
		{
			name: "No headers set",
			expectedHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetHTMXResponse(w, tt.trigger, tt.retarget, tt.reswap)

			for k, v := range tt.expectedHeaders {
				if got := w.Header().Get(k); got != v {
					t.Errorf("Header %s = %v, want %v", k, got, v)
				}
			}

			// Verify unexpected headers are not set
			if tt.trigger == "" && w.Header().Get("HX-Trigger") != "" {
				t.Errorf("Expected HX-Trigger to be empty")
			}
			if tt.retarget == "" && w.Header().Get("HX-Retarget") != "" {
				t.Errorf("Expected HX-Retarget to be empty")
			}
			if tt.reswap == "" && w.Header().Get("HX-Reswap") != "" {
				t.Errorf("Expected HX-Reswap to be empty")
			}
		})
	}
}

func TestSetHTMXRedirect(t *testing.T) {
	w := httptest.NewRecorder()
	SetHTMXRedirect(w, "/dashboard")

	if got := w.Header().Get("HX-Redirect"); got != "/dashboard" {
		t.Errorf("HX-Redirect = %v, want /dashboard", got)
	}
}

func TestSetHTMXRefresh(t *testing.T) {
	w := httptest.NewRecorder()
	SetHTMXRefresh(w)

	if got := w.Header().Get("HX-Refresh"); got != "true" {
		t.Errorf("HX-Refresh = %v, want true", got)
	}
}

func TestIsHTMXRequest_WithoutMiddleware(t *testing.T) {
	// Test that IsHTMXRequest returns false for context without middleware
	ctx := context.Background()
	if IsHTMXRequest(ctx) {
		t.Errorf("IsHTMXRequest() = true for empty context, want false")
	}
}

func TestHTMXHelpers_WithoutMiddleware(t *testing.T) {
	// Test that helper functions handle missing context values gracefully
	ctx := context.Background()

	if _, ok := HTMXTrigger(ctx); ok {
		t.Errorf("HTMXTrigger() ok = true for empty context, want false")
	}

	if _, ok := HTMXTarget(ctx); ok {
		t.Errorf("HTMXTarget() ok = true for empty context, want false")
	}

	if _, ok := HTMXPrompt(ctx); ok {
		t.Errorf("HTMXPrompt() ok = true for empty context, want false")
	}
}
