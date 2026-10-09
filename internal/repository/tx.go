package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/LimeOnTop/interverse-report/internal/usecase"
)

type ctxKey int

const txCtxKey ctxKey = 1

// DBTX is the common query surface for *sql.DB and *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func ContextWithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey, tx)
}

func DBTXFromContext(ctx context.Context, fallback DBTX) DBTX {
	if tx, ok := ctx.Value(txCtxKey).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return fallback
}

type SQLTxManager struct {
	db *sql.DB
}

func NewSQLTxManager(db *sql.DB) *SQLTxManager {
	return &SQLTxManager{db: db}
}

var _ usecase.TxManager = (*SQLTxManager)(nil)

func (m *SQLTxManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer tx.Rollback()
	txCtx := ContextWithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
