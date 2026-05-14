package logutil_test

import (
	"context"
	"errors"

	logutil "github.com/NYCU-SDC/summer/pkg/log"
	"go.uber.org/zap"
)

func main() {
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

	baseErr := errors.New("email already exists")

	// Wrap the error when it needs to carry detail to the logging layer.
	err := logutil.WrapTypedInfoError(logutil.ALREADY_EXISTS, baseErr, map[logutil.ErrorInfoKey]any{
		logutil.ErrorInfoOperation: "create_user",
		logutil.ErrorInfoField:     "email",
		logutil.ErrorInfoRetryable: false,
	})

	logutil.Error(ctx, eventLogger, "create user rejected", err, zap.String("email.domain", "example.com"))
}
