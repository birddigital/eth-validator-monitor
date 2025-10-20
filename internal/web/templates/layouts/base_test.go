package layouts

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestNavRendering verifies the Nav component renders with correct structure
func TestNavRendering(t *testing.T) {
	// Render the Nav component
	component := Nav()

	// Render to string
	buf := new(bytes.Buffer)
	err := component.Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Nav component: %v", err)
	}

	html := buf.String()

	// Verify essential navigation elements
	tests := []struct {
		name     string
		contains string
	}{
		{"has navbar class", "navbar"},
		{"has mobile logo", "ETH Monitor"},
		{"has desktop logo", "Ethereum Validator Monitor"},
		{"has home link", `href="/"`},
		{"has validators link", `href="/validators"`},
		{"has metrics link", `href="/metrics"`},
		{"has graphql link", `href="/graphql"`},
		{"has login link", `href="/login"`},
		{"has hamburger menu", "M4 6h16M4 12h16M4 18h16"}, // SVG path for hamburger icon
		{"has desktop navigation", "navbar-center hidden lg:flex"},
		{"has mobile dropdown", "navbar-end lg:hidden"},
		{"has dropdown menu", "dropdown dropdown-end"},
		{"has aria labels", "aria-label"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.contains) {
				t.Errorf("Nav component missing expected content: %q", tt.contains)
			}
		})
	}
}

// TestNavAccessibility verifies accessibility attributes
func TestNavAccessibility(t *testing.T) {
	component := Nav()

	buf := new(bytes.Buffer)
	err := component.Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Nav component: %v", err)
	}

	html := buf.String()

	// Check for accessibility attributes
	accessibilityTests := []struct {
		name     string
		contains string
	}{
		{"has aria-label on menu button", `aria-label="Open menu"`},
		{"has aria-haspopup", "aria-haspopup"},
		{"has aria-expanded", "aria-expanded"},
		{"has tabindex for keyboard nav", "tabindex"},
	}

	for _, tt := range accessibilityTests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.contains) {
				t.Errorf("Nav component missing accessibility attribute: %q", tt.contains)
			}
		})
	}
}

// TestFooterRendering verifies the Footer component with multi-column structure
func TestFooterRendering(t *testing.T) {
	component := Footer()

	buf := new(bytes.Buffer)
	err := component.Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Footer component: %v", err)
	}

	html := buf.String()

	tests := []struct {
		name     string
		contains string
	}{
		{"has footer class", "footer"},
		{"has project name", "Ethereum Validator Monitor"},
		{"has description", "Beacon Chain Performance Tracking"},
		{"has copyright", "2025"},

		// Services column
		{"has services heading", `id="footer-services"`},
		{"has services title", "Services"},
		{"has dashboard link", `href="/"`},
		{"has validators link", `href="/validators"`},
		{"has metrics link", `href="/metrics"`},
		{"has graphql link", `href="/graphql"`},

		// Company column
		{"has company heading", `id="footer-company"`},
		{"has company title", "Company"},
		{"has about link", `href="/about"`},
		{"has contact link", `href="/contact"`},
		{"has documentation link", `href="https://github.com/ethereum/consensus-specs"`},

		// Legal column
		{"has legal heading", `id="footer-legal"`},
		{"has legal title", "Legal"},
		{"has terms link", `href="/terms"`},
		{"has privacy link", `href="/privacy"`},
		{"has cookie link", `href="/cookies"`},

		// Accessibility
		{"has aria-labelledby for services", `aria-labelledby="footer-services"`},
		{"has aria-labelledby for company", `aria-labelledby="footer-company"`},
		{"has aria-labelledby for legal", `aria-labelledby="footer-legal"`},

		// DaisyUI classes
		{"has footer-title class", "footer-title"},
		{"has link hover class", "link link-hover"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.contains) {
				t.Errorf("Footer component missing expected content: %q", tt.contains)
			}
		})
	}
}

// TestBaseLayout verifies the base layout structure
func TestBaseLayout(t *testing.T) {
	// Use the Nav component itself as test content (it's a valid templ.Component)
	component := Base("Test Page", Nav())

	buf := new(bytes.Buffer)
	err := component.Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Base layout: %v", err)
	}

	html := buf.String()

	tests := []struct {
		name     string
		contains string
	}{
		{"has doctype", "<!doctype html>"},
		{"has html lang", `<html lang="en"`},
		{"has charset", `charset="UTF-8"`},
		{"has viewport", `name="viewport"`},
		{"has title", "Test Page - Ethereum Validator Monitor"},
		{"has tailwind css", "/static/css/output.css"},
		{"has htmx script", "htmx.org@1.9.10"},
		{"has app js", "/static/js/app.js"},
		{"has sticky header", "sticky top-0"},
		{"has main content", "<main"},
		{"has footer", "<footer"},
		{"has data-theme", `data-theme="light"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.contains) {
				t.Errorf("Base layout missing expected content: %q", tt.contains)
			}
		})
	}
}

// TestNavResponsiveClasses verifies responsive CSS classes
func TestNavResponsiveClasses(t *testing.T) {
	component := Nav()

	buf := new(bytes.Buffer)
	err := component.Render(context.Background(), buf)
	if err != nil {
		t.Fatalf("failed to render Nav component: %v", err)
	}

	html := buf.String()

	responsiveTests := []struct {
		name     string
		contains string
		purpose  string
	}{
		{"desktop links hidden on mobile", "hidden lg:flex", "Desktop nav links should be hidden on mobile"},
		{"mobile menu hidden on desktop", "lg:hidden", "Mobile hamburger menu should be hidden on desktop"},
		{"short logo on small screens", "hidden sm:inline", "Full logo should be hidden on small screens"},
		{"full logo on larger screens", "inline sm:hidden", "Short logo should be shown on small screens"},
	}

	for _, tt := range responsiveTests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(html, tt.contains) {
				t.Errorf("%s - missing class: %q", tt.purpose, tt.contains)
			}
		})
	}
}
