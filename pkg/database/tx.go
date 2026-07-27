package databaseutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

// DBTX is the minimal database surface sqlc-generated Queries expect.
// Both *pgxpool.Pool and pgx.Tx implement it.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// TxBeginner can start a new database transaction.
// *pgxpool.Pool satisfies this interface.
type TxBeginner interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

var ErrTransactionNotSupported = errors.New("database connection does not support transactions")

// TxRunner configures a single transactional callback. It is not a database
// transaction itself — nothing runs until Run is called.
//
// Prefer WithTransactionQueries for sqlc call sites; use WithTransaction only
// when the callback needs raw pgx.Tx without sqlc rebinding.
type TxRunner struct {
	ctx    context.Context
	db     DBTX
	logger *zap.Logger
}

// WithTransaction builds a TxRunner for reuses an existing pgx.Tx).
func WithTransaction(ctx context.Context, db DBTX, logger *zap.Logger) TxRunner {
	return TxRunner{ctx: ctx, db: db, logger: logger}
}

// Run executes fn inside a transaction lifecycle:
//
//   - If db is already pgx.Tx, fn runs on that tx (no begin/commit here).
//   - Otherwise db must implement TxBeginner: begin, fn, commit on success,
//     rollback on error or panic (rollback is deferred).
//
// Return a non-nil error from fn to abort and roll back. Do not call Commit or
// Rollback inside fn unless you have a deliberate sub-transaction design.
func (t TxRunner) Run(fn func(pgx.Tx) error) error {
	existingTx, ok := t.db.(pgx.Tx)
	if ok {
		return fn(existingTx)
	}

	beginner, ok := t.db.(TxBeginner)
	if !ok {
		return fmt.Errorf("%w", ErrTransactionNotSupported)
	}

	tx, err := beginner.BeginTx(t.ctx, pgx.TxOptions{})
	if err != nil {
		return WrapDBError(err, t.logger, "begin tx")
	}

	defer func() {
		rollbackErr := tx.Rollback(t.ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			t.logger.Error("rollback failed", zap.Error(rollbackErr))
		}
	}()

	err = fn(tx)
	if err != nil {
		return err
	}

	err = tx.Commit(t.ctx)
	if err != nil {
		return WrapDBError(err, t.logger, "commit tx")
	}

	return nil
}
