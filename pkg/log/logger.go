package logutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

// relativePrettyCallerEncoder returns a zapcore.CallerEncoder that formats the caller path relative to the root directory
// it enables clickable links in the GoLand console output
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

// Constructs is a convenience function that constructs a new logger with fields extracted from the context
func Constructs(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if fields, ok := ctx.Value(contextFieldsKey{}).(contextFields); ok && len(fields) > 0 {
		zapFields := make([]zap.Field, 0, len(fields))
		for _, field := range fields {
			zapFields = append(zapFields, field)
		}
		logger = logger.With(zapFields...)
	}

	logger = logger.With(codeFields(1)...)

	return logger
}

func codeFields(skip int) []zap.Field {
	pcs := make([]uintptr, 1)

	// Skip runtime.Callers 和 codeFields 自己
	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()

	namespace, function := splitFunction(frame.Function)

	return []zap.Field{
		zap.String("code.file.path", frame.File),
		zap.Int("code.line.number", frame.Line),
		zap.String("code.function.name", function),
		zap.String("code.namespace", namespace),
	}
}

func splitFunction(full string) (namespace string, function string) {
	i := strings.LastIndex(full, ".")
	if i == -1 {
		return "", full
	}

	return full[:i], full[i+1:]
}
