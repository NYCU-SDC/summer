package databaseutil

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type stubQueries struct {
	gotTx pgx.Tx
}

func (q *stubQueries) WithTx(tx pgx.Tx) *stubQueries {
	q.gotTx = tx
	return q
}

func TestWithTransactionQueries_Run(t *testing.T) {
	t.Parallel()

	tx := &trackTx{rollbackErr: pgx.ErrTxClosed}
	beginner := &mockBeginner{tx: tx}
	queries := &stubQueries{}

	err := WithTransactionQueries(context.Background(), beginner, zap.NewNop(), queries).
		Run(func(qtx *stubQueries) error {
			if qtx.gotTx != tx {
				t.Fatal("expected queries bound to active tx")
			}
			return nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tx.committed {
		t.Fatal("Commit not called")
	}
}

func TestQueriesTx_RunWithDBTx(t *testing.T) {
	t.Parallel()

	tx := &trackTx{rollbackErr: pgx.ErrTxClosed}
	beginner := &mockBeginner{tx: tx}
	queries := &stubQueries{}

	err := WithTransactionQueries(context.Background(), beginner, zap.NewNop(), queries).
		RunWithDBTx(func(gotTx pgx.Tx, qtx *stubQueries) error {
			if gotTx != tx {
				t.Fatal("expected same tx instance")
			}

			if qtx.gotTx != tx {
				t.Fatal("expected queries bound to active tx")
			}

			return nil
		})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !tx.committed {
		t.Fatal("Commit not called")
	}
}
