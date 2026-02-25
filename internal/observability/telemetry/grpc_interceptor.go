package telemetry

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	grpcInstrumentationName = "github.com/vertercloud/auth-service/grpc"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor with tracing
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(grpcInstrumentationName)
	propagator := otel.GetTextMapPropagator()

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract trace context from gRPC metadata
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = propagator.Extract(ctx, &metadataCarrier{md: md})

		// Start span
		start := time.Now()
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				semconv.RPCService(info.FullMethod),
				semconv.RPCMethod(info.FullMethod),
				attribute.String("rpc.grpc.kind", "unary"),
			),
		)
		defer span.End()

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)
		span.SetAttributes(attribute.Int64("rpc.duration_ms", duration.Milliseconds()))

		// Record result
		if err != nil {
			s, _ := status.FromError(err)
			span.SetStatus(codes.Error, s.Message())
			span.SetAttributes(
				attribute.Int("rpc.grpc.status_code", int(s.Code())),
				attribute.String("rpc.grpc.status_text", s.Code().String()),
				attribute.String("error.message", s.Message()),
			)
			span.RecordError(err)

			// Add event for specific error types
			span.AddEvent("grpc_error", trace.WithAttributes(
				attribute.String("grpc.code", s.Code().String()),
				attribute.String("grpc.message", s.Message()),
			))
		} else {
			span.SetStatus(codes.Ok, "")
			span.SetAttributes(attribute.Int("rpc.grpc.status_code", 0))
		}

		return resp, err
	}
}

// metadataCarrier adapts gRPC metadata to propagation.TextMapCarrier
type metadataCarrier struct {
	md metadata.MD
}

func (mc *metadataCarrier) Get(key string) string {
	values := mc.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (mc *metadataCarrier) Set(key, value string) {
	mc.md.Set(key, value)
}

func (mc *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc.md))
	for k := range mc.md {
		keys = append(keys, k)
	}
	return keys
}
