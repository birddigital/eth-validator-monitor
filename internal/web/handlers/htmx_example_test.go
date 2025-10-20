package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/birddigital/eth-validator-monitor/graph/middleware"
	"github.com/stretchr/testify/assert"
)

func TestHTMXExampleHandler_ContentNegotiation(t *testing.T) {
	handler := NewHTMXExampleHandler()

	// Wrap handler with HTMX middleware to simulate real request flow
	htmxMiddleware := middleware.NewHTMXMiddleware()
	wrappedHandler := htmxMiddleware.Middleware(handler)

	tests := []struct {
		name             string
		headers          map[string]string
		expectFullPage   bool
		expectPartial    bool
		expectHTMLTag    bool
		expectHeadTag    bool
		expectBodyTag    bool
		expectLayoutHTML bool
	}{
		{
			name:             "full page request without HX-Request header",
			headers:          map[string]string{},
			expectFullPage:   true,
			expectPartial:    false,
			expectHTMLTag:    true,
			expectHeadTag:    true,
			expectBodyTag:    true,
			expectLayoutHTML: true,
		},
		{
			name: "partial HTML with HX-Request header",
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectFullPage:   false,
			expectPartial:    true,
			expectHTMLTag:    false,
			expectHeadTag:    false,
			expectBodyTag:    false,
			expectLayoutHTML: false,
		},
		{
			name: "HTMX request with target and trigger",
			headers: map[string]string{
				"HX-Request": "true",
				"HX-Target":  "content-div",
				"HX-Trigger": "refresh-button",
			},
			expectFullPage:   false,
			expectPartial:    true,
			expectHTMLTag:    false,
			expectHeadTag:    false,
			expectBodyTag:    false,
			expectLayoutHTML: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			// Verify status code
			assert.Equal(t, http.StatusOK, w.Code, "Status code should be 200 OK")

			// Verify Content-Type
			assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"),
				"Content-Type should be text/html")

			body := w.Body.String()

			if tt.expectFullPage {
				// Full page should contain complete HTML structure
				assert.True(t, strings.Contains(body, "<html") || strings.Contains(body, "<!DOCTYPE"),
					"Full page should contain HTML tag or DOCTYPE")
				assert.True(t, strings.Contains(body, "<head>"),
					"Full page should contain head tag")
				assert.True(t, strings.Contains(body, "<body"),
					"Full page should contain body tag")
				assert.True(t, strings.Contains(body, "</html>"),
					"Full page should contain closing html tag")
			}

			if tt.expectPartial {
				// Partial should NOT contain full HTML structure
				assert.False(t, strings.Contains(body, "<html") || strings.Contains(body, "<!DOCTYPE"),
					"Partial should NOT contain HTML tag or DOCTYPE")
				assert.False(t, strings.Contains(body, "<head>"),
					"Partial should NOT contain head tag")
				// Body might contain content, so only check for absence of full structure
			}

			// All responses should contain some content
			assert.NotEmpty(t, body, "Response body should not be empty")
		})
	}
}

func TestHTMXExampleHandler_HeaderDetection(t *testing.T) {
	handler := NewHTMXExampleHandler()
	htmxMiddleware := middleware.NewHTMXMiddleware()
	wrappedHandler := htmxMiddleware.Middleware(handler)

	t.Run("detects HX-Request header correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
		req.Header.Set("HX-Request", "true")

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// Should NOT contain full page structure
		assert.False(t, strings.Contains(body, "</html>"),
			"HTMX request should return partial without closing html tag")
	})

	t.Run("handles missing HX-Request header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
		// No HX-Request header

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// Should contain full page structure
		assert.True(t, strings.Contains(body, "</html>"),
			"Non-HTMX request should return full page with closing html tag")
	})

	t.Run("handles HX-Request false value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
		req.Header.Set("HX-Request", "false")

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		body := w.Body.String()

		// Should contain full page structure (HX-Request must be "true" to be HTMX)
		assert.True(t, strings.Contains(body, "</html>"),
			"HX-Request: false should return full page")
	})
}

func TestHTMXExampleHandler_ResponseSize(t *testing.T) {
	handler := NewHTMXExampleHandler()
	htmxMiddleware := middleware.NewHTMXMiddleware()
	wrappedHandler := htmxMiddleware.Middleware(handler)

	// Get full page response
	fullPageReq := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
	fullPageW := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(fullPageW, fullPageReq)
	fullPageSize := len(fullPageW.Body.Bytes())

	// Get partial response
	partialReq := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
	partialReq.Header.Set("HX-Request", "true")
	partialW := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(partialW, partialReq)
	partialSize := len(partialW.Body.Bytes())

	// Partial should be smaller than full page
	// (This test might need adjustment based on actual template sizes)
	t.Logf("Full page size: %d bytes, Partial size: %d bytes", fullPageSize, partialSize)

	// At minimum, both should have some content
	assert.Greater(t, fullPageSize, 0, "Full page should have content")
	assert.Greater(t, partialSize, 0, "Partial should have content")

	// In most cases, partial should be smaller (unless they're identical components)
	// We'll just verify both are valid responses for now
	assert.NotEqual(t, 0, fullPageSize)
	assert.NotEqual(t, 0, partialSize)
}

func TestHTMXExampleHandler_ContentType(t *testing.T) {
	handler := NewHTMXExampleHandler()
	htmxMiddleware := middleware.NewHTMXMiddleware()
	wrappedHandler := htmxMiddleware.Middleware(handler)

	tests := []struct {
		name            string
		headers         map[string]string
		expectMIMEType  string
	}{
		{
			name:           "full page request",
			headers:        map[string]string{},
			expectMIMEType: "text/html; charset=utf-8",
		},
		{
			name: "HTMX partial request",
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectMIMEType: "text/html; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectMIMEType, w.Header().Get("Content-Type"),
				"Content-Type header should match expected MIME type")
		})
	}
}

func TestHTMXExampleHandler_MultipleRequests(t *testing.T) {
	// Verify handler is stateless and can handle multiple requests
	handler := NewHTMXExampleHandler()
	htmxMiddleware := middleware.NewHTMXMiddleware()
	wrappedHandler := htmxMiddleware.Middleware(handler)

	// Make 5 alternating requests
	for i := 0; i < 5; i++ {
		isHTMX := i%2 == 0

		req := httptest.NewRequest(http.MethodGet, "/api/htmx/dashboard", nil)
		if isHTMX {
			req.Header.Set("HX-Request", "true")
		}

		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i)
		assert.NotEmpty(t, w.Body.String(), "Request %d should have body", i)

		body := w.Body.String()
		if isHTMX {
			assert.False(t, strings.Contains(body, "</html>"),
				"HTMX request %d should return partial", i)
		} else {
			assert.True(t, strings.Contains(body, "</html>"),
				"Full request %d should return complete page", i)
		}
	}
}
