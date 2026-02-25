package http

import (
	"net/http"
	"strings"
)

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(environment string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// DEBUG: Log path to verify
			// fmt.Printf("DEBUG: Request path: %s\n", r.URL.Path)

			// Relaxed CSP for Swagger UI (needs inline scripts and eval)
			if strings.HasPrefix(r.URL.Path, "/swagger/") {
				w.Header().Set("Content-Security-Policy",
					"default-src 'self'; "+
						"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
						"style-src 'self' 'unsafe-inline'; "+
						"img-src 'self' data: https:; "+
						"font-src 'self'; "+
						"connect-src 'self'; "+
						"frame-ancestors 'none'")
			} else {
				// Strict CSP for API endpoints
				w.Header().Set("Content-Security-Policy",
					"default-src 'self'; "+
						"script-src 'self'; "+
						"style-src 'self' 'unsafe-inline'; "+
						"img-src 'self' data: https:; "+
						"font-src 'self'; "+
						"connect-src 'self'; "+
						"frame-ancestors 'none'")
			}

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// XSS Protection (legacy browsers)
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Referrer Policy - Only send referrer for same origin
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions Policy - Disable unnecessary features
			w.Header().Set("Permissions-Policy",
				"geolocation=(), "+
					"microphone=(), "+
					"camera=(), "+
					"payment=(), "+
					"usb=(), "+
					"magnetometer=(), "+
					"gyroscope=()")

			// HSTS - Force HTTPS (only in production)
			if environment == "production" {
				w.Header().Set("Strict-Transport-Security",
					"max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}
