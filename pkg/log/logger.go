package logutil

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"strings"
)

// ZapProductionConfig returns a zap.Config same as zap.NewProduction() but without sampling
func ZapProductionConfig() zap.Config {
	return zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:       false,
		DisableStacktrace: true,
		Encoding:          "json",
		EncoderConfig:     zap.NewProductionEncoderConfig(),
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stdout"},
	}
}

// ZapDevelopmentConfig returns a zap.Config same as zap.NewProduction() but with more pretty output
func ZapDevelopmentConfig() zap.Config {
	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(zap.DebugLevel),
		Development:       true,
		DisableStacktrace: true,
		Encoding:          "console",
		EncoderConfig:     zap.NewDevelopmentEncoderConfig(),
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
	}

	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeCaller = prettyEncodeCaller

	return config
}

// WithContext parses the context and adds the trace ID to the logger if available
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

	return logger
}

// prettyEncodeCaller add padding to the caller string
func prettyEncodeCaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	const fixedWidth = 25
	callerStr := caller.TrimmedPath()
	if len(callerStr) < fixedWidth {
		callerStr += strings.Repeat(" ", fixedWidth-len(callerStr))
	}
	callerStr += "\t"
	enc.AppendString(callerStr)
}
