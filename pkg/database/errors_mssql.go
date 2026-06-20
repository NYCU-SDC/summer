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

// Deprecated: database errors are classified here without logging. Return domain errors directly for new code.
func WrapMSSQLError(err error, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

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

	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
	}

	return wrappedErr
}

// Deprecated: database errors are classified here without logging. Return domain errors directly for new code.
func WrapMSSQLErrorWithKeyValue(err error, table, key, value string, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

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

	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
	}

	return wrappedErr
}
