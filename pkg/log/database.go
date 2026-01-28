package logutil

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type DBTracker struct {
	logger *zap.Logger
	op     string
	params map[string]interface{}
}

func StartDBOperation(ctx context.Context, logger *zap.Logger, op string, params map[string]interface{}) *DBTracker {
	if params == nil {
		params = make(map[string]interface{})
	}

	logger = logger.WithOptions(zap.AddCallerSkip(1))

	return &DBTracker{
		logger: logger,
		op:     op,
		params: params,
	}
}

func (t *DBTracker) SuccessWrite(pk string) {
	msg := fmt.Sprintf("DB operation %s completed (PK: %s)", t.op, pk)

	t.logger.Info(msg,
		zap.String("db.operation", t.op),
		zap.String("db.pk", pk),
	)
}

func (t *DBTracker) SuccessWriteBulk(rowsAffected int) {
	msg := fmt.Sprintf("DB operation %s completed: affected %d row(s)", t.op, rowsAffected)

	t.logger.Info(msg,
		zap.String("db.operation", t.op),
		zap.Int("db.rows_affected", rowsAffected),
	)
}

func (t *DBTracker) SuccessRead(rowsAffected int, pk string) {
	var msg string
	fields := []zap.Field{
		zap.String("db.operation", t.op),
		zap.Int("db.rows_affected", rowsAffected),
	}

	if pk != "" {
		msg = fmt.Sprintf("DB operation %s completed: retrieved %d row(s) (PK: %s)", t.op, rowsAffected, pk)
		fields = append(fields, zap.String("db.pk", pk))
	} else {
		msg = fmt.Sprintf("DB operation %s completed: retrieved %d row(s)", t.op, rowsAffected)
	}

	t.logger.Debug(msg, fields...)
}

func (t *DBTracker) Fail(err error) {
	msg := fmt.Sprintf("DB operation %s failed: %v (Params: %v)", t.op, err, t.params)

	t.logger.Warn(msg,
		zap.String("db.operation", t.op),
		zap.Any("db.parameters", t.params),
		zap.String("error", err.Error()),
	)
}
