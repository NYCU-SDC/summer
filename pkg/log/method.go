package logutil

import (
	"context"
	"fmt"
	"sort"

	"go.uber.org/zap"
)

type MethodTracker struct {
	logger      *zap.Logger
	serviceName string
}

func StartMethod(ctx context.Context, logger *zap.Logger, name string, params map[string]interface{}) *MethodTracker {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	logger = logger.WithOptions(zap.AddCallerSkip(1))

	logger.Info(fmt.Sprintf("Method %s started with params: %v", name, keys),
		zap.String("method.name", name),
		zap.Any("method.params", params),
	)

	return &MethodTracker{
		logger:      logger,
		serviceName: name,
	}
}

func (t *MethodTracker) Complete(result map[string]interface{}) {
	loggerMsg := fmt.Sprintf("Method %s completed: %v", t.serviceName, result)

	t.logger.Info(loggerMsg,
		zap.String("method.name", t.serviceName),
		zap.Any("method.result", result),
	)
}
