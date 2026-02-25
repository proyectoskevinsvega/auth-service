package telemetry

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createExporter creates a trace exporter based on configuration
func createExporter(cfg *Config, logger zerolog.Logger) (trace.SpanExporter, error) {
	switch cfg.ExporterType {
	case "jaeger":
		return createJaegerExporter(cfg, logger)
	case "otlp":
		return createOTLPExporter(cfg, logger)
	case "stdout":
		return createStdoutExporter(logger)
	default:
		return nil, fmt.Errorf("unsupported exporter type: %s", cfg.ExporterType)
	}
}

// createJaegerExporter creates a Jaeger exporter
func createJaegerExporter(cfg *Config, logger zerolog.Logger) (trace.SpanExporter, error) {
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(cfg.JaegerEndpoint),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	logger.Info().
		Str("endpoint", cfg.JaegerEndpoint).
		Msg("Jaeger exporter created")

	return exporter, nil
}

// createOTLPExporter creates an OTLP gRPC exporter
func createOTLPExporter(cfg *Config, logger zerolog.Logger) (trace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}

	if cfg.OTLPInsecure {
		opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithBlock()))

	exporter, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	logger.Info().
		Str("endpoint", cfg.OTLPEndpoint).
		Bool("insecure", cfg.OTLPInsecure).
		Msg("OTLP exporter created")

	return exporter, nil
}

// createStdoutExporter creates a stdout exporter (for testing)
func createStdoutExporter(logger zerolog.Logger) (trace.SpanExporter, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout exporter: %w", err)
	}

	logger.Info().Msg("stdout exporter created")
	return exporter, nil
}

// createSampler creates a sampler based on sampling rate
func createSampler(rate float64) trace.Sampler {
	if rate <= 0 {
		return trace.NeverSample()
	}
	if rate >= 1.0 {
		return trace.AlwaysSample()
	}
	return trace.TraceIDRatioBased(rate)
}
