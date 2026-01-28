package databaseutil

import (
	"context"
	"errors"
	"fmt"

	errorPkg "github.com/NYCU-SDC/summer/pkg/handler"
	logutil "github.com/NYCU-SDC/summer/pkg/log"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

const (
	PGErrUniqueViolation     = "23505"
	PGErrForeignKeyViolation = "23503"
	PGErrDeadlockDetected    = "40P01"
)

var (
	ErrUniqueViolation     = errors.New("unique constraint violation")
	ErrForeignKeyViolation = errors.New("foreign key violation")
	ErrDeadlockDetected    = errors.New("deadlock detected")
	ErrQueryTimeout        = errors.New("query timed out")
)

type InternalServerError struct {
	Source error
}

func (e InternalServerError) Error() string {
	return fmt.Sprintf("internal server error: %s", e.Source.Error())
}

func WrapDBError(err error, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

	var wrappedErr error

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		wrappedErr = fmt.Errorf("%w: %v", errorPkg.ErrNotFound, err)
	case errors.Is(err, context.DeadlineExceeded):
		wrappedErr = fmt.Errorf("%w: %v", ErrQueryTimeout, err)
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case PGErrUniqueViolation:
				wrappedErr = fmt.Errorf("%w: %v", ErrUniqueViolation, err)
			case PGErrForeignKeyViolation:
				wrappedErr = fmt.Errorf("%w: %v", ErrForeignKeyViolation, err)
			case PGErrDeadlockDetected:
				wrappedErr = fmt.Errorf("%w: %v", ErrDeadlockDetected, err)
			}
		}
	}

	isUnknownError := false
	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
		isUnknownError = true
	}

	logger.Warn("Failed to "+operation, zap.Error(wrappedErr), zap.String("operation", operation), zap.Bool("unknown_error", isUnknownError))

	return wrappedErr
}

func WrapDBErrorWithKeyValue(err error, table, key, value string, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

	var wrappedErr error

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		wrappedErr = errorPkg.NewNotFoundError(table, key, value, "")
	case errors.Is(err, context.DeadlineExceeded):
		wrappedErr = fmt.Errorf("%w: %v", ErrQueryTimeout, err)
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case PGErrUniqueViolation:
				wrappedErr = fmt.Errorf("%w: %v", ErrUniqueViolation, err)
			case PGErrForeignKeyViolation:
				wrappedErr = fmt.Errorf("%w: %v", ErrForeignKeyViolation, err)
			case PGErrDeadlockDetected:
				wrappedErr = fmt.Errorf("%w: %v", ErrDeadlockDetected, err)
			}
		}
	}

	isUnknownError := false
	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
		isUnknownError = true
	}

	logger.Warn("Failed to "+operation, zap.Error(wrappedErr), zap.String("table", table), zap.String("key", key), zap.String("value", value), zap.String("operation", operation), zap.Bool("unknown_error", isUnknownError))

	return wrappedErr
}

func WrapDBErrorWithTracker(err error, tracker *logutil.DBTracker, opDescription string) error {
	if err == nil {
		return nil
	}

	var wrappedErr error

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		wrappedErr = fmt.Errorf("%w: %v", errorPkg.ErrNotFound, err)
	case errors.Is(err, context.DeadlineExceeded):
		wrappedErr = fmt.Errorf("%w: %v", ErrQueryTimeout, err)
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case PGErrUniqueViolation:
				wrappedErr = fmt.Errorf("%w: %v", ErrUniqueViolation, err)
			case PGErrForeignKeyViolation:
				wrappedErr = fmt.Errorf("%w: %v", ErrForeignKeyViolation, err)
			case PGErrDeadlockDetected:
				wrappedErr = fmt.Errorf("%w: %v", ErrDeadlockDetected, err)
			}
		}
	}

	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
	}

	tracker.Fail(wrappedErr)
	finalErr := fmt.Errorf("failed to %s", opDescription)

	return finalErr
}
