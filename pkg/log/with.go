package logutil

import (
	"context"

	errutil "github.com/NYCU-SDC/summer/pkg/error"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextFieldsKey struct{}

type contextFields map[string]zap.Field

type userIDKey struct{}
type usernameKey struct{}
type displayNameKey struct{}
type requestIDKey struct{}

// WithFields returns a child context enriched with structured logging fields.
//
// The fields are later injected into a logger by Constructs or by the level
// helpers in logger.go. Fields with an empty key are ignored. Fields with the
// same key replace earlier values so a log entry does not emit duplicate keys.
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
		if field.Key == "" {
			continue
		}

		next[field.Key] = field
	}

	return context.WithValue(ctx, contextFieldsKey{}, next)
}

// WithUserID returns a child context carrying the authenticated user's ID.
//
// Constructs emits this value as enduser.id. Empty user IDs are ignored.
func WithUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if userID == "" {
		return ctx
	}

	return context.WithValue(ctx, userIDKey{}, userID)
}

// WithUsername returns a child context carrying the authenticated username.
//
// Constructs emits this value as enduser.username. Empty usernames are ignored.
func WithUsername(ctx context.Context, username string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if username == "" {
		return ctx
	}

	return context.WithValue(ctx, usernameKey{}, username)
}

// WithDisplayName returns a child context carrying the user's display name.
//
// Constructs emits this value as enduser.name. Empty names are ignored.
func WithDisplayName(ctx context.Context, name string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if name == "" {
		return ctx
	}

	return context.WithValue(ctx, displayNameKey{}, name)
}

// WithRequestID returns a child context carrying an application request ID.
//
// Constructs emits this value as request.id. Empty request IDs are ignored.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if requestID == "" {
		return ctx
	}

	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// WithReason returns a child context carrying event.reason.
//
// Use event.reason for the structured reason an event took a branch, failed,
// was rejected, or was skipped.
func WithReason(ctx context.Context, reason string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if reason == "" {
		return ctx
	}

	return WithFields(ctx, zap.String("event.reason", reason))
}

// WithErrorType returns a child context carrying error.type.
//
// Use error.type for a stable machine-readable error classification. Prefer the
// errutil.ErrorType constants when the error maps to a canonical status-like
// category.
func WithErrorType(ctx context.Context, errorType errutil.ErrorType) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	if errorType == "" {
		return ctx
	}

	return WithFields(ctx, zap.String("error.type", string(errorType)))
}

// WithTraceContext returns logger enriched with the active span context.
//
// When ctx contains a valid OpenTelemetry span, the returned logger includes
// trace_id, span_id, trace_flags, trace_sampled, and trace_state when present.
// A nil logger is treated as zap.NewNop.
func WithTraceContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if logger == nil {
		logger = zap.NewNop()
	}

	if ctx == nil {
		return logger
	}

	spanCtx := trace.SpanFromContext(ctx).SpanContext()
	if !spanCtx.IsValid() {
		return logger
	}

	fields := []zap.Field{
		zap.String("trace_id", spanCtx.TraceID().String()),
		zap.String("span_id", spanCtx.SpanID().String()),
		zap.String("trace_flags", spanCtx.TraceFlags().String()),
		zap.Bool("trace_sampled", spanCtx.IsSampled()),
	}

	if traceState := spanCtx.TraceState().String(); traceState != "" {
		fields = append(fields, zap.String("trace_state", traceState))
	}

	return logger.With(fields...)
}

// WithUserContext returns logger enriched with user and request fields from ctx.
//
// It reads values written by WithUserID, WithUsername, WithDisplayName, and
// WithRequestID. If no values are present, logger is returned unchanged. A nil
// logger is treated as zap.NewNop.
func WithUserContext(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if logger == nil {
		logger = zap.NewNop()
	}

	if ctx == nil {
		return logger
	}

	fields := make([]zap.Field, 0, 4)

	if userID, ok := ctx.Value(userIDKey{}).(string); ok && userID != "" {
		fields = append(fields, zap.String("enduser.id", userID))
	}

	if username, ok := ctx.Value(usernameKey{}).(string); ok && username != "" {
		fields = append(fields, zap.String("enduser.username", username))
	}

	if name, ok := ctx.Value(displayNameKey{}).(string); ok && name != "" {
		fields = append(fields, zap.String("enduser.name", name))
	}

	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok && requestID != "" {
		fields = append(fields, zap.String("request.id", requestID))
	}

	if len(fields) == 0 {
		return logger
	}

	return logger.With(fields...)
}

// WithOutcome returns logger enriched with event.outcome.
//
// This string-based helper is useful for custom outcome values. Prefer
// WithEventOutcome when the value fits the EventOutcome constants.
func WithOutcome(outcome string, logger *zap.Logger) *zap.Logger {
	if logger == nil || outcome == "" {
		return logger
	}

	return logger.With(zap.String("event.outcome", outcome))
}

// WithEventOutcome returns logger enriched with a typed event.outcome value.
//
// Event outcome describes whether an event succeeded, failed, timed out, was
// cancelled, or has an unknown result.
func WithEventOutcome(outcome EventOutcome, logger *zap.Logger) *zap.Logger {
	if logger == nil || outcome == "" {
		return logger
	}

	return logger.With(zap.String("event.outcome", string(outcome)))
}

// WithEventName returns logger enriched with event.name.
//
// Use event.name for the stable, human-readable event identity, such as
// "user.login" or "profile.update".
func WithEventName(eventName string, logger *zap.Logger) *zap.Logger {
	if logger == nil || eventName == "" {
		return logger
	}

	return logger.With(zap.String("event.name", eventName))
}
