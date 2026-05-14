package logutil

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// WithContext returns logger enriched with legacy context fields.
//
// It reads OpenTelemetry trace/span IDs from ctx and also reads the old string
// context keys "user_id", "username", and "name". Those user fields are emitted
// as user_id, username, and display-name for compatibility with older callers.
//
// Deprecated: use Constructs for the standard context-first logging path, or use
// WithTraceContext and WithUserContext directly when decorating a logger. New
// code should write user and request values with WithUserID, WithUsername,
// WithDisplayName, and WithRequestID instead of raw string context keys.
func WithContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if ctx == nil {
		return logger
	}

	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if spanCtx.HasTraceID() {
		logger = logger.With(zap.String("trace_id", spanCtx.TraceID().String()))
	}

	if spanCtx.HasSpanID() {
		logger = logger.With(zap.String("span_id", spanCtx.SpanID().String()))
	}

	if ctx.Value("user_id") != nil {
		logger = logger.With(zap.Any("user_id", ctx.Value("user_id")))
	}

	if ctx.Value("username") != nil {
		logger = logger.With(zap.Any("username", ctx.Value("username")))
	}

	if ctx.Value("name") != nil {
		logger = logger.With(zap.Any("display-name", ctx.Value("name")))
	}

	return logger
}
