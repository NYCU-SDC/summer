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

// ZapProductionConfig returns the default production zap config for the project.
//
// It is equivalent to zap.NewProductionConfig with sampling disabled, JSON
// encoding, stdout as both the normal and error output, Info as the default
// level, and stack traces disabled. Use this config for services where logs are
// consumed by a collector or log aggregation system.
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

// ZapDevelopmentConfig returns a development zap config optimized for local use.
//
// It writes colorized console logs, enables Debug level logs, writes internal
// zap errors to stderr, and formats caller paths relative to the current working
// directory so IDE consoles can open the referenced source location.
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

// Constructs returns a logger enriched with fields carried by ctx.
//
// The returned logger includes fields from WithFields, active OpenTelemetry
// trace/span metadata, and user/request metadata recorded through the context
// helper functions in with.go. A nil logger is treated as zap.NewNop so callers
// can use this function safely in optional logging paths.
//
// Constructs does not add caller code fields. The level helpers in this file add
// those fields at the actual log call site.
func Constructs(ctx context.Context, logger *zap.Logger) *zap.Logger {
	if logger == nil {
		logger = zap.NewNop()
	}

	if ctx == nil {
		return logger
	}

	if fields, ok := ctx.Value(contextFieldsKey{}).(contextFields); ok && len(fields) > 0 {
		zapFields := make([]zap.Field, 0, len(fields))
		for _, field := range fields {
			zapFields = append(zapFields, field)
		}

		logger = logger.With(zapFields...)
	}

	logger = WithTraceContext(ctx, logger)
	logger = WithUserContext(ctx, logger)

	return logger
}

// Debug logs msg at Debug level with fields from ctx and the call site.
//
// Use Debug for diagnostic details that are useful during development or
// investigation but too noisy for normal production operation.
func Debug(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	logger = Constructs(ctx, logger)
	fields = append(fields, codeFields(1)...)
	logger.Debug(msg, fields...)
}

// Info logs msg at Info level with fields from ctx and the call site.
//
// Use Info for successful lifecycle events and expected state transitions that
// operators may need to understand normal system behavior.
func Info(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	logger = Constructs(ctx, logger)
	fields = append(fields, codeFields(1)...)
	logger.Info(msg, fields...)
}

// Warn logs msg at Warn level with fields from ctx and the call site.
//
// Use Warn for recoverable failures, degraded behavior, retries, or rejected
// inputs that do not represent a server-side fault.
func Warn(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	logger = Constructs(ctx, logger)
	fields = append(fields, codeFields(1)...)
	logger.Warn(msg, fields...)
}

// Error logs msg at Error level with fields from ctx, err, and the call site.
//
// When err is non-nil, Error adds zap.Error, error.message,
// exception.message, exception.type, exception.stacktrace, any error.type from
// ErrorTypeCarrier, and any structured error.info fields from InfoCarrier.
func Error(ctx context.Context, logger *zap.Logger, msg string, err error, fields ...zap.Field) {
	logger = Constructs(ctx, logger)

	if err != nil {
		fields = append(fields, ErrorFieldsWithStacktrace(err)...)
	}

	fields = append(fields, codeFields(1)...)

	logger.Error(msg, fields...)
}

// DPanic logs msg at DPanic level with fields from ctx, err, and the call site.
//
// In zap development mode this panics after writing the log entry. In production
// mode it logs as an error. Use this for impossible states that should fail fast
// in development but should not necessarily terminate production services.
func DPanic(ctx context.Context, logger *zap.Logger, msg string, err error, fields ...zap.Field) {
	logger = Constructs(ctx, logger)

	if err != nil {
		fields = append(fields, ErrorFieldsWithStacktrace(err)...)
	}

	fields = append(fields, codeFields(1)...)

	logger.DPanic(msg, fields...)
}

// Panic logs msg at Panic level with fields from ctx, err, and the call site.
//
// It writes the log entry and then panics. Use this only when the caller is
// intentionally aborting the current control flow.
func Panic(ctx context.Context, logger *zap.Logger, msg string, err error, fields ...zap.Field) {
	logger = Constructs(ctx, logger)

	if err != nil {
		fields = append(fields, ErrorFieldsWithStacktrace(err)...)
	}

	fields = append(fields, codeFields(1)...)

	logger.Panic(msg, fields...)
}

// Fatal logs msg at Fatal level with fields from ctx, err, and the call site.
//
// It writes the log entry and then terminates the process through zap's fatal
// behavior. Use this only for unrecoverable process startup or runtime failures.
func Fatal(ctx context.Context, logger *zap.Logger, msg string, err error, fields ...zap.Field) {
	logger = Constructs(ctx, logger)

	if err != nil {
		fields = append(fields, ErrorFieldsWithStacktrace(err)...)
	}

	fields = append(fields, codeFields(1)...)

	logger.Fatal(msg, fields...)
}

// codeFields returns source location fields for the caller.
//
// The fields follow OpenTelemetry-style names where possible:
// code.file.path, code.file.name, code.line.number, code.function.name, and
// code.namespace.
func codeFields(skip int) []zap.Field {
	pcs := make([]uintptr, 1)

	n := runtime.Callers(skip+2, pcs)
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()

	namespace, function := splitFunction(frame.Function)

	return []zap.Field{
		zap.String("code.file.path", trimWorkdir(frame.File)),
		zap.String("code.file.name", filepath.Base(frame.File)),
		zap.Int("code.line.number", frame.Line),
		zap.String("code.function.name", function),
		zap.String("code.namespace", namespace),
	}
}

// splitFunction splits a fully qualified Go function name into namespace and name.
func splitFunction(full string) (namespace string, function string) {
	i := strings.LastIndex(full, ".")
	if i == -1 {
		return "", full
	}

	return full[:i], full[i+1:]
}

// trimWorkdir returns path relative to the current working directory when possible.
func trimWorkdir(path string) string {
	wd, err := os.Getwd()
	if err != nil || wd == "" {
		return path
	}

	rel, err := filepath.Rel(wd, path)
	if err != nil {
		return path
	}

	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return path
	}

	return rel
}

// relativePrettyCallerEncoder formats zap caller paths for local development.
//
// Paths inside rootDir are rendered as relative file:line strings. External
// paths keep only the last few path components and are prefixed with external/.
func relativePrettyCallerEncoder(rootDir string) zapcore.CallerEncoder {
	const fixedWidth = 40

	return func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		relPath, err := filepath.Rel(rootDir, caller.File)
		callerStr := ""

		if err == nil && !strings.HasPrefix(relPath, "..") && !filepath.IsAbs(relPath) {
			callerStr = fmt.Sprintf("%s:%d", relPath, caller.Line)
		} else {
			parts := strings.Split(caller.File, string(filepath.Separator))

			const lastN = 3
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
