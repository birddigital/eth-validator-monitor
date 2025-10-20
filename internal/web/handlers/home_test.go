package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   []string // Strings that should be in the response
	}{
		{
			name:           "GET home page returns 200 OK",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<!doctype html>",
				"<html lang=\"en\"",
				"Ethereum Validator Monitor",
				"Total Validators",
				"Active Validators",
				"Total Balance",
				"Avg Effectiveness",
			},
		},
		{
			name:           "GET home page contains navigation",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<nav",
				"Home",
				"Validators",
				"Metrics",
				"Login",
			},
		},
		{
			name:           "GET home page contains footer",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<footer",
				"Ethereum Validator Monitor",
			},
		},
		{
			name:           "GET home page contains action buttons",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"View All Validators",
				"View Metrics",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handler
			handler := NewHomeHandler()

			// Create test request
			req := httptest.NewRequest(tt.method, "/", nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Execute request
			handler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response body contains expected strings
			body := w.Body.String()
			for _, expected := range tt.expectedBody {
				if !strings.Contains(body, expected) {
					t.Errorf("expected body to contain %q, but it didn't.\nBody: %s", expected, body)
				}
			}

			// Verify Content-Type is HTML
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "text/html") {
				t.Logf("Warning: expected Content-Type to contain 'text/html', got %q", contentType)
			}
		})
	}
}

func TestHomeHandlerHTMLStructure(t *testing.T) {
	handler := NewHomeHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify proper HTML structure
	requiredElements := []string{
		"<head>",
		"</head>",
		"<body",
		"</body>",
		"<header",
		"</header>",
		"<main",
		"</main>",
		"<footer",
		"</footer>",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(body, elem) {
			t.Errorf("HTML missing required element: %q", elem)
		}
	}
}

func TestHomeHandlerMetaTags(t *testing.T) {
	handler := NewHomeHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify meta tags for proper HTML document
	metaTags := []string{
		`<meta charset="UTF-8"`,
		`<meta name="viewport"`,
		`<meta name="description"`,
	}

	for _, meta := range metaTags {
		if !strings.Contains(body, meta) {
			t.Errorf("HTML missing required meta tag: %q", meta)
		}
	}
}

func TestHomeHandlerStaticResources(t *testing.T) {
	handler := NewHomeHandler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify static resource links
	resources := []string{
		`href="/static/css/output.css"`,
		`src="https://unpkg.com/htmx.org`,
		`src="/static/js/app.js"`,
	}

	for _, resource := range resources {
		if !strings.Contains(body, resource) {
			t.Errorf("HTML missing static resource: %q", resource)
		}
	}
}
