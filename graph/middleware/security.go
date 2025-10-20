package middleware

import (
	"fmt"
	"net/http"
)

// SecurityHeaders configuration for HTTP security headers
type SecurityHeaders struct {
	// HSTS configuration
	HSTSMaxAge            int  // Max age in seconds (default: 31536000 = 1 year)
	HSTSIncludeSubdomains bool // Include subdomains in HSTS (default: true)

	// Content Security Policy directives
	CSPDirectives string // CSP directives string

	// Enable/disable specific headers (all enabled by default)
	EnableHSTS        bool
	EnableXFrameOpts  bool
	EnableXContentOpts bool
	EnableXXSSProt    bool
	EnableCSP         bool
	EnableReferrer    bool
	EnablePermissions bool
}

// DefaultSecurityHeaders returns security headers with secure defaults
func DefaultSecurityHeaders() *SecurityHeaders {
	return &SecurityHeaders{
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		// Basic CSP - adjust based on frontend needs
		// This policy is restrictive but secure by default
		CSPDirectives: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self';",

		// All headers enabled by default
		EnableHSTS:        true,
		EnableXFrameOpts:  true,
		EnableXContentOpts: true,
		EnableXXSSProt:    true,
		EnableCSP:         true,
		EnableReferrer:    true,
		EnablePermissions: true,
	}
}

// Middleware returns an HTTP middleware function that adds security headers
func (sh *SecurityHeaders) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strict-Transport-Security (HSTS)
		// Only set if request is over HTTPS (either via TLS or behind a proxy)
		if sh.EnableHSTS {
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				hstsValue := fmt.Sprintf("max-age=%d", sh.HSTSMaxAge)
				if sh.HSTSIncludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				// Add preload directive for even stronger security
				hstsValue += "; preload"
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}
		}

		// X-Frame-Options - Prevent clickjacking attacks
		if sh.EnableXFrameOpts {
			w.Header().Set("X-Frame-Options", "DENY")
		}

		// X-Content-Type-Options - Prevent MIME type sniffing
		if sh.EnableXContentOpts {
			w.Header().Set("X-Content-Type-Options", "nosniff")
		}

		// X-XSS-Protection - Legacy XSS protection for older browsers
		// Modern browsers use CSP instead, but this provides defense in depth
		if sh.EnableXXSSProt {
			w.Header().Set("X-XSS-Protection", "1; mode=block")
		}

		// Content-Security-Policy - Modern defense against XSS and data injection
		if sh.EnableCSP && sh.CSPDirectives != "" {
			w.Header().Set("Content-Security-Policy", sh.CSPDirectives)
		}

		// Referrer-Policy - Limit information leak via Referer header
		if sh.EnableReferrer {
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		}

		// Permissions-Policy (formerly Feature-Policy)
		// Disable potentially dangerous browser features
		if sh.EnablePermissions {
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()")
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// SecureHeaders is a convenience function for quick middleware setup with defaults
func SecureHeaders(next http.Handler) http.Handler {
	return DefaultSecurityHeaders().Middleware(next)
}
