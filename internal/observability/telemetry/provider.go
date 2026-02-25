package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds telemetry configuration
type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	ExporterType   string
	JaegerEndpoint string
	OTLPEndpoint   string
	OTLPInsecure   bool
	SamplingRate   float64
	TraceHTTP      bool
	TraceGRPC      bool
	TraceDatabase  bool
	TraceRedis     bool
}

// Provider wraps the OpenTelemetry tracer provider
type Provider struct {
	tp     *sdktrace.TracerProvider
	logger zerolog.Logger
}

// Initialize creates and configures the OpenTelemetry provider
func Initialize(cfg Config, logger zerolog.Logger) (*Provider, error) {
	if !cfg.Enabled {
		logger.Info().Msg("telemetry disabled")
		return &Provider{logger: logger}, nil
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create exporter based on configuration
	exporter, err := createExporter(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create sampler
	sampler := createSampler(cfg.SamplingRate)

	// Create tracer provider with batching
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator to W3C Trace Context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info().
		Str("service", cfg.ServiceName).
		Str("version", cfg.ServiceVersion).
		Str("environment", cfg.Environment).
		Str("exporter", cfg.ExporterType).
		Float64("sampling_rate", cfg.SamplingRate).
		Bool("trace_http", cfg.TraceHTTP).
		Bool("trace_grpc", cfg.TraceGRPC).
		Bool("trace_database", cfg.TraceDatabase).
		Bool("trace_redis", cfg.TraceRedis).
		Msg("telemetry initialized successfully")

	return &Provider{
		tp:     tp,
		logger: logger,
	}, nil
}

// Shutdown gracefully shuts down the tracer provider
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.tp == nil {
		return nil
	}

	if err := p.tp.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown tracer provider: %w", err)
	}

	p.logger.Info().Msg("telemetry shut down successfully")
	return nil
}

// Tracer returns a tracer for the given instrumentation name
func (p *Provider) Tracer(name string) trace.Tracer {
	if p.tp == nil {
		return otel.Tracer(name)
	}
	return p.tp.Tracer(name)
}
