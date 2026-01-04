package traceutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"

	handlerutil "github.com/NYCU-SDC/summer/pkg/handler"
	logutil "github.com/NYCU-SDC/summer/pkg/log"
	"github.com/NYCU-SDC/summer/pkg/problem"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type CustomResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	Body       *bytes.Buffer
}

func (w *CustomResponseWriter) WriteHeader(code int) {
	w.StatusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *CustomResponseWriter) Write(b []byte) (int, error) {
	if w.Body != nil {
		w.Body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func writeBodyHandlingError(w http.ResponseWriter, err error, logger *zap.Logger) {
	p := problem.NewInternalServerProblem("Internal server error")

	logger = logger.WithOptions(zap.AddCallerSkip(1))

	logger.Warn("Handling I/O with request body in TraceMiddleware", zap.Error(err))

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)

	jsonBytes, err := json.Marshal(p)
	if err != nil {
		logger.Error("Failed to marshal problem response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(jsonBytes)
	if err != nil {
		logger.Error("Failed to write problem response", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// TraceMiddleware provides OpenTelemetry tracing and structured logging for HTTP handlers.
// It creates a new span for each request, linking it to any upstream traces, and enriches
// the logger with trace and span IDs for context propagation.
//
// The middleware logs request completion with varying severity levels (Info, Warn, Error)
// based on the HTTP status code. When the debug flag is enabled, it enhances error logs
// for 4xx and 5xx responses by including full request/response headers and bodies.
//
// Note: Enabling debug mode causes the entire request body to be read into memory. This
// can lead to high memory consumption for large payloads, such as file uploads, and is a
// known limitation. Use with caution in environments that handle large requests.
func TraceMiddleware(next http.HandlerFunc, logger *zap.Logger, debug bool) http.HandlerFunc {
	name := "internal/middleware"
	tracer := otel.Tracer(name)
	propagator := otel.GetTextMapPropagator()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		upstream := trace.SpanFromContext(ctx).SpanContext()

		ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
		defer span.End()

		span.SetAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
			attribute.String("query", r.URL.RawQuery),
		)
		span.AddEvent("HTTPRequestStarted")

		logger = logutil.WithContext(ctx, logger)
		if upstream.HasTraceID() {
			logger.Debug("Upstream trace available", zap.String("trace_id", upstream.TraceID().String()))
		} else {
			logger.Debug("No upstream trace available, creating a new one", zap.String("trace_id", span.SpanContext().TraceID().String()))
		}

		var bodyBytes []byte

		if debug {
			var err error
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				writeBodyHandlingError(w, fmt.Errorf("failed to read request body: %w", err), logger)
				return
			}

			err = r.Body.Close()
			if err != nil {
				writeBodyHandlingError(w, fmt.Errorf("failed to close request body: %w", err), logger)
				return
			}

			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		crw := &CustomResponseWriter{ResponseWriter: w}
		next(crw, r.WithContext(ctx))

		status := crw.StatusCode

		fields := []zap.Field{
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.Int("status", status),
		}

		if status >= 100 && status < 400 {
			logger.Debug("Request completed", fields...)
		} else if status >= 400 && status < 500 {
			logger.Error("Client request rejected", fields...)
		} else {
			if status == 500 && debug {
				fields = append(fields,
					zap.Any("request_headers", r.Header),
					zap.String("request_body", string(bodyBytes)),
					zap.Any("response_headers", crw.Header()),
					zap.String("response_body", crw.Body.String()),
				)
			}
			logger.Error("Internal server error occurred", fields...)
		}
	}
}

func RecoverMiddleware(next http.HandlerFunc, logger *zap.Logger, debug bool) http.HandlerFunc {
	name := "internal/middleware"
	tracer := otel.Tracer(name)

	return func(w http.ResponseWriter, r *http.Request) {
		traceCtx, span := tracer.Start(r.Context(), "RecoverMiddleware")
		logger = logutil.WithContext(traceCtx, logger)

		defer func() {
			needRecovery, errString, caller := PanicRecoveryError(recover())
			if needRecovery {
				span.AddEvent("PanicRecovered", trace.WithAttributes(attribute.String("panic", fmt.Sprintf("%v", errString))))
				logger.Error("Recovered from panic", zap.Any("error", errString), zap.String("trace", fmt.Sprintf("%s", caller)))
				if debug {
					for _, line := range caller {
						fmt.Printf("\t%s\n", line)
					}
				}

				problem.New().WriteError(context.Background(), w, handlerutil.ErrInternalServer, logger)
			}

			span.End()
		}()

		next(w, r.WithContext(traceCtx))
	}
}

func PanicRecoveryError(err any) (bool, string, []string) {
	if err == nil {
		return false, "", nil
	}

	var callers []string
	for i := 2; ; /* 1 for New() 2 for NewPanicRecoveryError */ i++ {
		_, file, line, got := runtime.Caller(i)
		if !got {
			break
		}

		callers = append(callers, fmt.Sprintf("%s:%d", file, line))
	}

	if parseErr, ok := err.(error); ok {
		return true, parseErr.Error(), callers
	}

	return true, fmt.Sprintf("%v", err), callers
}
