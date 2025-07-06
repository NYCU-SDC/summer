package logutil

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
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
	rootDir, _ := os.Getwd()

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
	config.EncoderConfig.EncodeCaller = relativePrettyCallerEncoder(rootDir)

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

func relativePrettyCallerEncoder(rootDir string) zapcore.CallerEncoder {
	const fixedWidth = 40

	return func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		relPath, err := filepath.Rel(rootDir, caller.File)
		callerStr := ""

		if err == nil && !strings.HasPrefix(relPath, "..") && !filepath.IsAbs(relPath) {
			callerStr = fmt.Sprintf("%s:%d", relPath, caller.Line)
		} else {
			parts := strings.Split(caller.File, string(filepath.Separator))

			lastN := 3
			if len(parts) > lastN {
				parts = parts[len(parts)-lastN:]
			}
			callerStr = fmt.Sprintf("external/%s:%d", filepath.Join(parts...), caller.Line)
		}

		if len(callerStr) < fixedWidth {
			callerStr += strings.Repeat(" ", fixedWidth-len(callerStr))
		}
		callerStr += "\t"
		enc.AppendString(callerStr)
	}
}
