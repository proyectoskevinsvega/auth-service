package telemetry

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	dbInstrumentationName = "github.com/vertercloud/auth-service/database"
)

// PgxTracer implements pgx.QueryTracer for OpenTelemetry
type PgxTracer struct {
	tracer trace.Tracer
}

// NewPgxTracer creates a new PostgreSQL tracer
func NewPgxTracer() *PgxTracer {
	return &PgxTracer{
		tracer: otel.Tracer(dbInstrumentationName),
	}
}

type pgxSpanKey struct{}

// TraceQueryStart is called at the beginning of Query, QueryRow, and Exec calls
func (t *PgxTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !trace.SpanFromContext(ctx).IsRecording() {
		return ctx
	}

	// Extract SQL operation (SELECT, INSERT, UPDATE, DELETE, etc.)
	operation := extractSQLOperation(data.SQL)

	// Create span name
	spanName := "db." + strings.ToLower(operation)

	// Start span
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.system", "postgresql"),
			semconv.DBStatement(sanitizeSQL(data.SQL)),
			attribute.String("db.operation", operation),
		),
	)

	// Store span in context for TraceQueryEnd
	return context.WithValue(ctx, pgxSpanKey{}, span)
}

// TraceQueryEnd is called at the end of Query, QueryRow, and Exec calls
func (t *PgxTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	span, ok := ctx.Value(pgxSpanKey{}).(trace.Span)
	if !ok || !span.IsRecording() {
		return
	}
	defer span.End()

	if data.Err != nil {
		span.SetStatus(codes.Error, data.Err.Error())
		span.RecordError(data.Err)
		span.AddEvent("query_error", trace.WithAttributes(
			attribute.String("error.message", data.Err.Error()),
		))
	} else {
		span.SetStatus(codes.Ok, "")
		if data.CommandTag.RowsAffected() > 0 {
			span.SetAttributes(
				attribute.Int64("db.rows_affected", data.CommandTag.RowsAffected()),
			)
		}
	}
}

// extractSQLOperation extracts the SQL operation (SELECT, INSERT, UPDATE, DELETE, etc.)
func extractSQLOperation(sql string) string {
	sql = strings.TrimSpace(sql)
	parts := strings.SplitN(sql, " ", 2)
	if len(parts) > 0 {
		return strings.ToUpper(parts[0])
	}
	return "UNKNOWN"
}

// sanitizeSQL removes sensitive data from SQL statements
func sanitizeSQL(sql string) string {
	// Limit length to avoid huge spans
	maxLen := 1000
	if len(sql) > maxLen {
		sql = sql[:maxLen] + "..."
	}

	// Remove excessive whitespace and newlines
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\t", " ")

	// Remove multiple spaces
	sql = strings.Join(strings.Fields(sql), " ")

	return sql
}
