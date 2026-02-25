package telemetry

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	httpInstrumentationName = "github.com/vertercloud/auth-service/http"
)

// HTTPMiddleware creates an OpenTelemetry tracing middleware for HTTP requests
func HTTPMiddleware(serviceName string) func(next http.Handler) http.Handler {
	tracer := otel.Tracer(httpInstrumentationName)
	propagator := otel.GetTextMapPropagator()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract trace context from HTTP headers (W3C Trace Context)
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			// Create span name from method and path
			spanName := r.Method + " " + r.URL.Path

			// Start span
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPMethod(r.Method),
					semconv.HTTPTarget(r.URL.Path),
					attribute.String("http.route", r.URL.Path),
					attribute.String("http.scheme", r.URL.Scheme),
					semconv.NetHostName(r.Host),
					attribute.String("http.user_agent", r.UserAgent()),
					attribute.String("net.peer.ip", r.RemoteAddr),
					attribute.String("http.url", r.URL.String()),
				),
			)
			defer span.End()

			// Get request ID from chi middleware if available
			if reqID := middleware.GetReqID(ctx); reqID != "" {
				span.SetAttributes(attribute.String("http.request_id", reqID))
			}

			// Wrap response writer to capture status code and response size
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Record start time
			start := time.Now()

			// Process request
			next.ServeHTTP(ww, r.WithContext(ctx))

			// Calculate duration
			duration := time.Since(start)
			statusCode := ww.Status()

			// Record response metadata
			span.SetAttributes(
				semconv.HTTPStatusCode(statusCode),
				semconv.HTTPResponseContentLength(int(ww.BytesWritten())),
				attribute.Int64("http.duration_ms", duration.Milliseconds()),
				attribute.String("http.status_text", http.StatusText(statusCode)),
			)

			// Set span status based on HTTP status code
			switch {
			case statusCode >= 500:
				span.SetStatus(codes.Error, http.StatusText(statusCode))
			case statusCode >= 400:
				// Client errors are not span errors, but we can mark them
				span.SetStatus(codes.Error, http.StatusText(statusCode))
			default:
				span.SetStatus(codes.Ok, "")
			}

			// Add span events for important status codes
			switch statusCode {
			case http.StatusUnauthorized:
				span.AddEvent("authentication_failed")
			case http.StatusForbidden:
				span.AddEvent("authorization_failed")
			case http.StatusTooManyRequests:
				span.AddEvent("rate_limit_exceeded")
			default:
				if statusCode >= 500 {
					span.AddEvent("server_error", trace.WithAttributes(
						attribute.String("error.type", strconv.Itoa(statusCode)),
					))
				}
			}
		})
	}
}
