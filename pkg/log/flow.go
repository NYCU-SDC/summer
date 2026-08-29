package logutil

import (
	"context"

	"go.uber.org/zap"
)

// SetupFlow initializes the context and logger fields used by a service flow.
//
// It stores request-scoped fields on ctx, decorates logger with event.name, and
// returns both values for subsequent logutil level calls. A nil ctx becomes
// context.Background, and a nil logger becomes zap.NewNop.
func SetupFlow(ctx context.Context, logger *zap.Logger, eventName string, fields ...zap.Field) (context.Context, *zap.Logger) {
	if ctx == nil {
		ctx = context.Background()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	if len(fields) > 0 {
		ctx = WithFields(ctx, fields...)
	}

	logger = WithEventName(eventName, logger)

	return ctx, logger
}
