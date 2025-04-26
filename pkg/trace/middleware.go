package traceutil

import (
	"context"
	"fmt"
	"github.com/NYCU-SDC/summer/pkg/handler"
	"github.com/NYCU-SDC/summer/pkg/log"
	"github.com/NYCU-SDC/summer/pkg/problem"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
	"runtime"
)

func TraceMiddleware(next http.HandlerFunc, logger *zap.Logger) http.HandlerFunc {
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

		next(w, r.WithContext(ctx))
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

				problem.WriteError(context.Background(), w, handlerutil.ErrInternalServer, logger)
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
