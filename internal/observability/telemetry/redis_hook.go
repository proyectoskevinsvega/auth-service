package telemetry

import (
	"context"
	"net"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	redisInstrumentationName = "github.com/vertercloud/auth-service/redis"
)

// RedisHook implements redis.Hook for OpenTelemetry tracing
type RedisHook struct {
	tracer trace.Tracer
}

// NewRedisHook creates a new Redis tracing hook
func NewRedisHook() *RedisHook {
	return &RedisHook{
		tracer: otel.Tracer(redisInstrumentationName),
	}
}

// DialHook implements redis.Hook.DialHook
func (h *RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// ProcessHook implements redis.Hook.ProcessHook for individual commands
func (h *RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if !trace.SpanFromContext(ctx).IsRecording() {
			return next(ctx, cmd)
		}

		cmdName := cmd.Name()
		spanName := "redis." + strings.ToLower(cmdName)

		ctx, span := h.tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("db.system", "redis"),
				semconv.DBStatement(sanitizeRedisCommand(cmd.String())),
				attribute.String("db.redis.command", cmdName),
			),
		)
		defer span.End()

		// Execute command
		err := next(ctx, cmd)

		// Record result
		if err != nil && err != redis.Nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			span.AddEvent("redis_error", trace.WithAttributes(
				attribute.String("error.message", err.Error()),
			))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// ProcessPipelineHook implements redis.Hook.ProcessPipelineHook for pipelined commands
func (h *RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if !trace.SpanFromContext(ctx).IsRecording() {
			return next(ctx, cmds)
		}

		spanName := "redis.pipeline"

		ctx, span := h.tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				attribute.String("db.system", "redis"),
				attribute.Int("db.redis.pipeline.length", len(cmds)),
			),
		)
		defer span.End()

		// Add command names as attributes
		if len(cmds) > 0 && len(cmds) <= 10 {
			commandNames := make([]string, len(cmds))
			for i, cmd := range cmds {
				commandNames[i] = cmd.Name()
			}
			span.SetAttributes(attribute.StringSlice("db.redis.pipeline.commands", commandNames))
		}

		// Execute pipeline
		err := next(ctx, cmds)

		// Record result
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			span.AddEvent("redis_pipeline_error", trace.WithAttributes(
				attribute.String("error.message", err.Error()),
			))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

// sanitizeRedisCommand removes sensitive data from Redis commands
func sanitizeRedisCommand(cmd string) string {
	// Limit length to avoid huge spans
	maxLen := 500
	if len(cmd) > maxLen {
		cmd = cmd[:maxLen] + "..."
	}

	// Remove excessive whitespace
	cmd = strings.Join(strings.Fields(cmd), " ")

	return cmd
}
