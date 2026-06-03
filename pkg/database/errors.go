package databaseutil

import (
	"context"
	"errors"
	"fmt"
	errorPkg "github.com/NYCU-SDC/summer/pkg/handler"
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

func (e InternalServerError) Unwrap() error {
	return e.Source
}

// Deprecated: database errors are classified here without logging. Return domain errors directly for new code.
func WrapDBError(err error, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

	var wrappedErr error

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		wrappedErr = fmt.Errorf("%w: %w", errorPkg.ErrNotFound, err)
	case errors.Is(err, context.DeadlineExceeded):
		wrappedErr = fmt.Errorf("%w: %w", ErrQueryTimeout, err)
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case PGErrUniqueViolation:
				wrappedErr = fmt.Errorf("%w: %w", ErrUniqueViolation, err)
			case PGErrForeignKeyViolation:
				wrappedErr = fmt.Errorf("%w: %w", ErrForeignKeyViolation, err)
			case PGErrDeadlockDetected:
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
func WrapDBErrorWithKeyValue(err error, table, key, value string, logger *zap.Logger, operation string) error {
	if err == nil {
		return nil
	}

	var wrappedErr error

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		wrappedErr = errorPkg.NewNotFoundError(table, key, value, "")
	case errors.Is(err, context.DeadlineExceeded):
		wrappedErr = fmt.Errorf("%w: %w", ErrQueryTimeout, err)
	default:
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case PGErrUniqueViolation:
				wrappedErr = fmt.Errorf("%w: %w", ErrUniqueViolation, err)
			case PGErrForeignKeyViolation:
				wrappedErr = fmt.Errorf("%w: %w", ErrForeignKeyViolation, err)
			case PGErrDeadlockDetected:
				wrappedErr = fmt.Errorf("%w: %w", ErrDeadlockDetected, err)
			}
		}
	}

	if wrappedErr == nil {
		wrappedErr = InternalServerError{Source: err}
	}

	return wrappedErr
}
