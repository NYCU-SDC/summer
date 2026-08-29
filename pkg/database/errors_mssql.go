package databaseutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	errorPkg "github.com/NYCU-SDC/summer/pkg/handler"
	mssql "github.com/microsoft/go-mssqldb"
	"go.uber.org/zap"
)

const (
	// SQL Server error numbers
	MSSQLErrUniqueViolation     = 2627 // Unique constraint violation
	MSSQLErrUniqueIndex         = 2601 // Duplicate key in unique index
	MSSQLErrForeignKeyViolation = 547  // Foreign key violation
	MSSQLErrDeadlockDetected    = 1205 // Deadlock detected
)

// WrapMSSQLError is the legacy helper: it logs the error and classifies it into a domain error.
// New code should return domain errors directly and log through pkg/log.
func WrapMSSQLError(err error, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

	logger.Error("Failed to "+operation, zap.Error(err))

	var wrappedErr error

	switch {
	case errors.Is(err, sql.ErrNoRows):
		wrappedErr = fmt.Errorf("%w: %w", errorPkg.ErrNotFound, err)
	case errors.Is(err, context.DeadlineExceeded):
		wrappedErr = fmt.Errorf("%w: %w", ErrQueryTimeout, err)
	default:
		var mssqlErr mssql.Error
		if errors.As(err, &mssqlErr) {
			switch mssqlErr.Number {
			case MSSQLErrUniqueViolation, MSSQLErrUniqueIndex:
				wrappedErr = fmt.Errorf("%w: %w", ErrUniqueViolation, err)
			case MSSQLErrForeignKeyViolation:
				wrappedErr = fmt.Errorf("%w: %w", ErrForeignKeyViolation, err)
			case MSSQLErrDeadlockDetected:
				wrappedErr = fmt.Errorf("%w: %w", ErrDeadlockDetected, err)
			}
		}
	}

	isUnknownError := false
	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
		isUnknownError = true
	}

	logger.Warn("Wrapped database error", zap.Error(wrappedErr), zap.String("operation", operation), zap.Bool("unknown_error", isUnknownError))

	return wrappedErr
}

// WrapMSSQLErrorWithKeyValue is the legacy helper: it logs the error and classifies it into a domain error,
// using table/key/value to build the not-found error. New code should return domain errors directly
// and log through pkg/log.
func WrapMSSQLErrorWithKeyValue(err error, table, key, value string, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

	logger.Error("Failed to "+operation, zap.Error(err))

	var wrappedErr error

	switch {
	case errors.Is(err, sql.ErrNoRows):
		wrappedErr = errorPkg.NewNotFoundError(table, key, value, "")
	case errors.Is(err, context.DeadlineExceeded):
		wrappedErr = fmt.Errorf("%w: %w", ErrQueryTimeout, err)
	default:
		var mssqlErr mssql.Error
		if errors.As(err, &mssqlErr) {
			switch mssqlErr.Number {
			case MSSQLErrUniqueViolation, MSSQLErrUniqueIndex:
				wrappedErr = fmt.Errorf("%w: %w", ErrUniqueViolation, err)
			case MSSQLErrForeignKeyViolation:
				wrappedErr = fmt.Errorf("%w: %w", ErrForeignKeyViolation, err)
			case MSSQLErrDeadlockDetected:
				wrappedErr = fmt.Errorf("%w: %w", ErrDeadlockDetected, err)
			}
		}
	}

	isUnknownError := false
	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
		isUnknownError = true
	}

	logger.Warn("Wrapped database error with key value", zap.Error(wrappedErr), zap.String("table", table), zap.String("key", key), zap.String("value", value), zap.String("operation", operation), zap.Bool("unknown_error", isUnknownError))

	return wrappedErr
}
