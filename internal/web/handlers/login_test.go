package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		expectedBody   []string
	}{
		{
			name:           "GET login page without error",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<!doctype html>",
				"<html lang=\"en\"",
				"Login",
				"Access your validator dashboard",
				`<form method="POST" action="/login"`,
				`<input type="text" id="username"`,
				`<input type="password" id="password"`,
				`<button type="submit"`,
			},
		},
		{
			name:           "GET login page with error message",
			method:         http.MethodGet,
			queryParams:    "?error=Invalid+credentials",
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<!doctype html>",
				"Login",
				"Invalid credentials",
				"alert-error",
			},
		},
		{
			name:           "GET login page with redirect URL",
			method:         http.MethodGet,
			queryParams:    "?redirect=/validators",
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<!doctype html>",
				"Login",
				`<input type="hidden" name="redirect" value="/validators"`,
			},
		},
		{
			name:           "GET login page with both error and redirect",
			method:         http.MethodGet,
			queryParams:    "?error=Session+expired&redirect=/metrics",
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<!doctype html>",
				"Session expired",
				"alert-error",
				`<input type="hidden" name="redirect" value="/metrics"`,
			},
		},
		{
			name:           "GET login page contains navigation",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedBody: []string{
				"<nav",
				"Home",
				"Login",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewLoginHandler()

			req := httptest.NewRequest(tt.method, "/login"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			body := w.Body.String()
			for _, expected := range tt.expectedBody {
				if !strings.Contains(body, expected) {
					t.Errorf("expected body to contain %q, but it didn't.\nBody: %s", expected, body)
				}
			}
		})
	}
}

func TestLoginHandlerFormFields(t *testing.T) {
	handler := NewLoginHandler()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify form has required fields with proper attributes
	formFields := []string{
		`name="username"`,
		`type="text"`,
		`required`,
		`autocomplete="username"`,
		`name="password"`,
		`type="password"`,
		`autocomplete="current-password"`,
	}

	for _, field := range formFields {
		if !strings.Contains(body, field) {
			t.Errorf("Form missing required field attribute: %q", field)
		}
	}
}

func TestLoginHandlerHTMLStructure(t *testing.T) {
	handler := NewLoginHandler()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify proper HTML structure
	requiredElements := []string{
		"<head>",
		"</head>",
		"<body",
		"</body>",
		"<form",
		"</form>",
	}

	for _, elem := range requiredElements {
		if !strings.Contains(body, elem) {
			t.Errorf("HTML missing required element: %q", elem)
		}
	}
}

func TestLoginHandlerNoErrorMessageByDefault(t *testing.T) {
	handler := NewLoginHandler()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// When no error query param, should not show error alert
	// The template should conditionally render the error message
	if strings.Count(body, "alert-error") > 0 {
		t.Error("Login page should not show error alert when no error parameter is provided")
	}
}

func TestLoginHandlerAccessibilityAttributes(t *testing.T) {
	handler := NewLoginHandler()
	req := httptest.NewRequest(http.MethodGet, "/login?error=test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify accessibility attributes
	accessibilityAttrs := []string{
		`role="alert"`, // Error message should have alert role
		`<label for="username"`,
		`<label for="password"`,
	}

	for _, attr := range accessibilityAttrs {
		if !strings.Contains(body, attr) {
			t.Errorf("HTML missing accessibility attribute: %q", attr)
		}
	}
}
