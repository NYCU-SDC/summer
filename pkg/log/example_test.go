package logutil_test

import (
	"context"
	"errors"

	logutil "github.com/NYCU-SDC/summer/pkg/log"
	"go.uber.org/zap"
)

type createUserError struct {
	userID string
	reason string
}

func (e createUserError) Error() string {
	return "create user failed"
}

func (e createUserError) LogErrorType() logutil.ErrorType {
	return logutil.INVALID_ARGUMENT
}

func (e createUserError) LogInfo() map[string]any {
	return map[string]any{
		string(logutil.ErrorInfoReason): e.reason,
		string(logutil.ErrorInfoUserID): e.userID,
	}
}

func Example() {
	logger := zap.NewExample()

	// Context fields are attached once and reused by every log call in this flow.
	ctx := context.Background()
	ctx = logutil.WithRequestID(ctx, "req-7")
	ctx = logutil.WithUserID(ctx, "user-42")
	ctx = logutil.WithFields(ctx, zap.String("service.name", "account-api"))

	// Logger decorators describe the event without passing the same fields again.
	eventLogger := logutil.WithEventDomain("identity", logger)
	eventLogger = logutil.WithEventName("user.create", eventLogger)
	eventLogger = logutil.WithEventAction("create", eventLogger)
	eventLogger = logutil.WithEventOutcome(logutil.EventOutcomeFailure, eventLogger)

	// Domain errors can expose log metadata directly; no extra error wrapping is needed.
	err := createUserError{
		userID: "user-42",
		reason: "email already exists",
	}

	logutil.Error(ctx, eventLogger, "create user rejected", err, zap.String("email.domain", "example.com"))
}

func ExampleWrapTypedInfoError() {
	logger := zap.NewExample()
	ctx := logutil.WithRequestID(context.Background(), "req-7")

	baseErr := errors.New("email already exists")

	// Wrap helpers are useful when you already have an error, but still need to
	// pass structured detail to the logging layer.
	err := logutil.WrapTypedInfoError(logutil.ALREADY_EXISTS, baseErr, map[logutil.ErrorInfoKey]any{
		logutil.ErrorInfoOperation: "create_user",
		logutil.ErrorInfoField:     "email",
		logutil.ErrorInfoRetryable: false,
	})

	logutil.Error(ctx, logger, "create user failed", err)
}
