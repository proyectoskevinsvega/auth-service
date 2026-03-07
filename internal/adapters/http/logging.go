package http

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// accessLogger wraps http.ResponseWriter to capture the status code and
// bytes written without buffering the entire response body.
type accessLogger struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

// WriteHeader captures the status code.
func (al *accessLogger) WriteHeader(code int) {
	if !al.wroteHeader {
		al.statusCode = code
		al.wroteHeader = true
	}
	al.ResponseWriter.WriteHeader(code)
}

// Write captures the number of bytes written.
func (al *accessLogger) Write(b []byte) (int, error) {
	n, err := al.ResponseWriter.Write(b)
	al.bytesWritten += n
	if !al.wroteHeader {
		al.statusCode = http.StatusOK
		al.wroteHeader = true
	}
	return n, err
}

// Flush implements http.Flusher for SSE support.
func (al *accessLogger) Flush() {
	if f, ok := al.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker for WebSocket support.
func (al *accessLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := al.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// HTTPAccessLogger returns a middleware that logs HTTP requests to stdout
// and captures request metadata with proper status code tracking.
func HTTPAccessLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract client IP
			ip := extractClientIP(r)

			// Wrap response writer to capture status and bytes
			wrapped := &accessLogger{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Extract user ID from context if available
			userID := ""
			if uid, ok := r.Context().Value(UserContextKey).(string); ok {
				userID = uid
			}

			// Extract tenant ID from context if available
			tenantID := ""
			if tid, ok := r.Context().Value(TenantContextKey).(string); ok {
				tenantID = tid
			}

			// Log the request
			logAccessRequest(logger, r.Method, r.URL.Path, ip, wrapped.statusCode, duration, userID, tenantID)
		})
	}
}

// extractClientIP extracts the real client IP from the request,
// considering X-Forwarded-For and X-Real-IP headers.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (load balancer)
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// logAccessRequest logs an HTTP request in Apache/Nginx style format to stdout
// with color-coded status codes for better readability.
func logAccessRequest(logger zerolog.Logger, method, path, remoteAddr string, statusCode int, duration time.Duration, userID, tenantID string) {
	// Color codes for status codes
	var statusColor string
	switch {
	case statusCode >= 200 && statusCode < 300:
		statusColor = "\033[32m" // Green
	case statusCode >= 300 && statusCode < 400:
		statusColor = "\033[36m" // Cyan
	case statusCode >= 400 && statusCode < 500:
		statusColor = "\033[33m" // Yellow
	case statusCode >= 500:
		statusColor = "\033[31m" // Red
	default:
		statusColor = "\033[0m" // Default
	}
	reset := "\033[0m"

	// Format: METHOD PATH STATUS DURATION [USER_ID] [TENANT_ID]
	msg := fmt.Sprintf("%s %s %d %dms", method, path, statusCode, duration.Milliseconds())
	if userID != "" {
		msg += fmt.Sprintf(" user: %s", userID)
	}
	if tenantID != "" {
		msg += fmt.Sprintf(" tenant: %s", tenantID)
	}

	logger.Info().
		Str("method", method).
		Str("path", path).
		Str("remote_addr", remoteAddr).
		Int("status_code", statusCode).
		Int64("duration_ms", duration.Milliseconds()).
		Str("user_id", userID).
		Str("tenant_id", tenantID).
		Msgf("%s%s%s", statusColor, msg, reset)
}
