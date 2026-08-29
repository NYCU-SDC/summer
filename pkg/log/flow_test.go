package logutil

import (
	"context"
	"testing"

	errutil "github.com/NYCU-SDC/summer/pkg/error"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// nilContext is passed where a test deliberately exercises the nil-context
// fallback in SetupFlow. Using a variable keeps staticcheck's SA1012 (do not
// pass a nil Context) from firing on an intentional case.
var nilContext context.Context

func TestSetupFlowInitializesContextAndLogger(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	baseLogger := zap.New(core)

	ctx, logger := SetupFlow(
		nilContext,
		baseLogger,
		"user.create",
		zap.String("request.id", "req-7"),
		zap.String("enduser.id", "user-42"),
		zap.String("enduser.username", "alice"),
		zap.String("enduser.name", "Alice"),
		zap.String("service.name", "account-api"),
		zap.String("route", "/users"),
	)
	ctx = WithReason(ctx, "duplicate_email")
	ctx = WithErrorType(ctx, errutil.ALREADY_EXISTS)
	logger = WithEventOutcome(EventOutcomeFailure, logger)

	if ctx == nil {
		t.Fatal("SetupFlow returned nil context")
	}
	if logger == nil {
		t.Fatal("SetupFlow returned nil logger")
	}

	Info(ctx, logger, "flow initialized")

	if logs.Len() != 1 {
		t.Fatalf("expected 1 log entry, got %d", logs.Len())
	}

	fields := logs.All()[0].ContextMap()
	want := map[string]any{
		"request.id":       "req-7",
		"enduser.id":       "user-42",
		"enduser.username": "alice",
		"enduser.name":     "Alice",
		"service.name":     "account-api",
		"event.name":       "user.create",
		"event.outcome":    "failure",
		"event.reason":     "duplicate_email",
		"error.type":       "ALREADY_EXISTS",
		"route":            "/users",
	}

	for key, value := range want {
		if got := fields[key]; got != value {
			t.Fatalf("field %q = %v, want %v", key, got, value)
		}
	}
}

func TestSetupFlowUsesSafeDefaults(t *testing.T) {
	ctx, logger := SetupFlow(nilContext, nil, "")

	if ctx == nil {
		t.Fatal("SetupFlow returned nil context")
	}
	if logger == nil {
		t.Fatal("SetupFlow returned nil logger")
	}
}
