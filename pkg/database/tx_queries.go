package databaseutil

import (
	"context"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// WithTxBinder is satisfied by sqlc-generated Queries (WithTx rebinding).
type WithTxBinder[T any] interface {
	WithTx(pgx.Tx) T
}

// QueriesTx combines TxRunner with a sqlc binder so callbacks receive
// transaction-scoped queries without manual WithTx calls.
type QueriesTx[T any] struct {
	TxRunner
	binder WithTxBinder[T]
}

// WithTransactionQueries returns a TxRunner plus sqlc binder; call Run or RunWithDBTx.
func WithTransactionQueries[T any](
	ctx context.Context, db DBTX, logger *zap.Logger,
	binder WithTxBinder[T],
) QueriesTx[T] {
	return QueriesTx[T]{
		TxRunner: WithTransaction(ctx, db, logger),
		binder:   binder,
	}
}

// Run executes fn with sqlc queries bound to the transaction.
// Equivalent to manually calling queries.WithTx(tx) inside WithTransaction.Run.
//
// Use Run when every database access in the callback goes through qtx.
// Returning an error rolls back; nil commits (when this call owns the tx).
func (q QueriesTx[T]) Run(fn func(T) error) error {
	return q.TxRunner.Run(func(tx pgx.Tx) error {
		return fn(q.binder.WithTx(tx))
	})
}

// RunWithDBTx executes fn with both pgx.Tx and transaction-scoped queries.
//
// Use RunWithDBTx when the same transaction must be shared with another
// sqlc-backed service or any API that accepts pgx.Tx
func (q QueriesTx[T]) RunWithDBTx(fn func(pgx.Tx, T) error) error {
	return q.TxRunner.Run(func(tx pgx.Tx) error {
		return fn(tx, q.binder.WithTx(tx))
	})
}
