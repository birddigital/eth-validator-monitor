package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticFileServing(t *testing.T) {
	// Setup: create temporary static file directory
	tmpDir := t.TempDir()
	staticDir := filepath.Join(tmpDir, "web", "static")
	require.NoError(t, os.MkdirAll(staticDir, 0755))

	// Create test files
	cssContent := []byte("body { color: red; }")
	cssPath := filepath.Join(staticDir, "style.css")
	require.NoError(t, os.WriteFile(cssPath, cssContent, 0644))

	jsContent := []byte("console.log('test');")
	jsPath := filepath.Join(staticDir, "app.js")
	require.NoError(t, os.WriteFile(jsPath, jsContent, 0644))

	htmlContent := []byte("<html><body>Test</body></html>")
	htmlPath := filepath.Join(staticDir, "index.html")
	require.NoError(t, os.WriteFile(htmlPath, htmlContent, 0644))

	// Setup router
	r := chi.NewRouter()

	// Create file server with cache control
	fileServer := http.FileServer(http.Dir(staticDir))
	r.With(CacheControl(31536000)).Handle("/static/*",
		http.StripPrefix("/static/", fileServer))

	tests := []struct {
		name               string
		path               string
		expectedStatus     int
		expectedContent    string
		expectedTypePrefix string
		checkCache         bool
	}{
		{
			name:               "serve CSS file",
			path:               "/static/style.css",
			expectedStatus:     http.StatusOK,
			expectedContent:    "body { color: red; }",
			expectedTypePrefix: "text/css",
			checkCache:         true,
		},
		{
			name:               "serve JS file",
			path:               "/static/app.js",
			expectedStatus:     http.StatusOK,
			expectedContent:    "console.log('test');",
			expectedTypePrefix: "text/javascript",
			checkCache:         true,
		},
		{
			name:           "non-existent file returns 404",
			path:           "/static/nonexistent.css",
			expectedStatus: http.StatusNotFound,
			checkCache:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code,
				"Response status should match expected")

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedContent, rec.Body.String(),
					"Response body should match expected content")

				if tt.expectedTypePrefix != "" {
					contentType := rec.Header().Get("Content-Type")
					assert.Contains(t, contentType, tt.expectedTypePrefix,
						"Content-Type should contain expected prefix")
				}
			}

			if tt.checkCache {
				cacheControl := rec.Header().Get("Cache-Control")
				assert.Equal(t, "public, max-age=31536000", cacheControl,
					"Cache-Control header should be set for successful responses")
			}
		})
	}
}

func TestStaticFileServingCacheHeaders(t *testing.T) {
	// Focused test on cache header behavior
	tmpDir := t.TempDir()
	staticDir := filepath.Join(tmpDir, "web", "static")
	require.NoError(t, os.MkdirAll(staticDir, 0755))

	testFile := filepath.Join(staticDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	r := chi.NewRouter()
	fileServer := http.FileServer(http.Dir(staticDir))
	r.With(CacheControl(31536000)).Handle("/static/*",
		http.StripPrefix("/static/", fileServer))

	req := httptest.NewRequest(http.MethodGet, "/static/test.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "public, max-age=31536000", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "test content", rec.Body.String())
}

func TestStaticFileServingSubdirectories(t *testing.T) {
	// Test that subdirectories work correctly
	tmpDir := t.TempDir()
	staticDir := filepath.Join(tmpDir, "web", "static")
	cssDir := filepath.Join(staticDir, "css")
	jsDir := filepath.Join(staticDir, "js")
	require.NoError(t, os.MkdirAll(cssDir, 0755))
	require.NoError(t, os.MkdirAll(jsDir, 0755))

	cssContent := []byte("/* main styles */")
	require.NoError(t, os.WriteFile(filepath.Join(cssDir, "main.css"), cssContent, 0644))

	jsContent := []byte("// app code")
	require.NoError(t, os.WriteFile(filepath.Join(jsDir, "main.js"), jsContent, 0644))

	r := chi.NewRouter()
	fileServer := http.FileServer(http.Dir(staticDir))
	r.With(CacheControl(31536000)).Handle("/static/*",
		http.StripPrefix("/static/", fileServer))

	tests := []struct {
		name            string
		path            string
		expectedContent string
	}{
		{
			name:            "serve CSS from subdirectory",
			path:            "/static/css/main.css",
			expectedContent: "/* main styles */",
		},
		{
			name:            "serve JS from subdirectory",
			path:            "/static/js/main.js",
			expectedContent: "// app code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.expectedContent, rec.Body.String())
			assert.Equal(t, "public, max-age=31536000", rec.Header().Get("Cache-Control"))
		})
	}
}
