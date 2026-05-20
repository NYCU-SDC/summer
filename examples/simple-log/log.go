package main

import (
	"context"
	"errors"

	errutil "github.com/NYCU-SDC/summer/pkg/error"
	logutil "github.com/NYCU-SDC/summer/pkg/log"
	"go.uber.org/zap"
)

func main() {
	logger := zap.NewExample()

	ctx, eventLogger := logutil.SetupFlow(
		context.Background(),
		logger,
		"user.create",
		zap.String("request.id", "req-7"),
		zap.String("enduser.id", "user-42"),
		zap.String("service.name", "account-api"),
	)
	eventLogger = logutil.WithEventOutcome(logutil.EventOutcomeFailure, eventLogger)

	baseErr := errors.New("email already exists")

	// Wrap the error when it needs to carry detail to the logging layer.
	err := errutil.WrapTypedInfoError(errutil.ALREADY_EXISTS, baseErr, map[errutil.ErrorInfoKey]any{
		errutil.ErrorInfoOperation: "create_user",
		errutil.ErrorInfoField:     "email",
		errutil.ErrorInfoRetryable: false,
	})

	err = errutil.WrapInfoError(err, map[string]any{
		"test": "helloworld",
	})

	logutil.Error(ctx, eventLogger, "create user rejected", err, zap.String("email.domain", "example.com"))
}
