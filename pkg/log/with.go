package logutil

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextFieldsKey struct{}

type contextFields map[string]zap.Field

// WithFields returns a child context enriched with request-scoped logging fields.
// Fields with the same key replace earlier fields to avoid duplicate log keys.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	existing, _ := ctx.Value(contextFieldsKey{}).(contextFields)

	next := make(contextFields, len(existing)+len(fields))
	for key, field := range existing {
		next[key] = field
	}

	for _, field := range fields {
		next[field.Key] = field
	}

	return context.WithValue(ctx, contextFieldsKey{}, next)
}

func WithOutcome(outcome string, logger *zap.Logger) *zap.Logger {
	if logger == nil || outcome == "" {
		return logger
	}
	return logger.With(zap.String("event.outcome", outcome))
}

func WithEventName(actionName string, logger *zap.Logger) *zap.Logger {
	if logger == nil || actionName == "" {
		return logger
	}
	return logger.With(zap.String("event.name", actionName))
}

func WithReason(reason string, logger *zap.Logger) *zap.Logger {
	if logger == nil || reason == "" {
		return logger
	}
	return logger.With(zap.String("event.kind", reason))
}

func WithErrorType(errorKind ErrorType, logger *zap.Logger) *zap.Logger {
	if logger == nil || errorKind == "" {
		return logger
	}
	return logger.With(zap.String("error.type", string(errorKind)))
}

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
