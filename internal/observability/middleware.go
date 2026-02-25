package observability

import (
	"net/http"
	"strconv"
	"time"
)

// PrometheusMiddleware creates an HTTP middleware that records HTTP metrics
func PrometheusMiddleware(metrics *Metrics) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Increment in-flight requests
			metrics.IncrementInFlightRequests()
			defer metrics.DecrementInFlightRequests()

			// Record start time
			start := time.Now()

			// Wrap response writer to capture status and size
			wrapper := NewResponseWriterWrapper(w)

			// Process request
			next.ServeHTTP(wrapper, r)

			// Calculate duration
			duration := time.Since(start)

			// Get response info
			status := strconv.Itoa(wrapper.Status())
			method := r.Method
			endpoint := normalizeEndpoint(r.URL.Path)
			responseSize := wrapper.Size()

			// Record metrics
			metrics.RecordHTTPRequest(method, endpoint, status, duration, responseSize)

			// Check for rate limit errors
			if wrapper.Status() == http.StatusTooManyRequests {
				identifier := r.RemoteAddr
				metrics.RecordRateLimitExceeded(endpoint, identifier)
			}
		})
	}
}

// normalizeEndpoint converts dynamic routes to static patterns
// Example: /api/v1/users/123 -> /api/v1/users/:id
func normalizeEndpoint(path string) string {
	if path == "" {
		return "unknown"
	}
	return path
}

// ResponseWriterWrapper wraps http.ResponseWriter to capture response size
type ResponseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	size       int
}

// NewResponseWriterWrapper creates a new wrapper
func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code
func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the response size
func (w *ResponseWriterWrapper) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

// Status returns the captured status code
func (w *ResponseWriterWrapper) Status() int {
	return w.statusCode
}

// Size returns the captured response size
func (w *ResponseWriterWrapper) Size() int {
	return w.size
}
