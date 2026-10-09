package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
)

type txState struct{ commits, rollbacks int }
type txDriver struct{ state *txState }
type txConnection struct{ state *txState }
type testTransaction struct{ state *txState }

var txDriverSequence atomic.Int64

func (d txDriver) Open(string) (driver.Conn, error) { return &txConnection{d.state}, nil }
func (c *txConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}
func (c *txConnection) Close() error              { return nil }
func (c *txConnection) Begin() (driver.Tx, error) { return &testTransaction{c.state}, nil }
func (c *txConnection) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}
func (t *testTransaction) Commit() error   { t.state.commits++; return nil }
func (t *testTransaction) Rollback() error { t.state.rollbacks++; return nil }
func TestTransactionFinishesOnSuccessErrorAndPanic(t *testing.T) {
	for _, mode := range []string{"success", "error", "panic"} {
		t.Run(mode, func(t *testing.T) {
			state := &txState{}
			name := fmt.Sprintf("report-tx-%d", txDriverSequence.Add(1))
			sql.Register(name, txDriver{state})
			db, err := sql.Open(name, "")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			manager := NewSQLTxManager(db)
			sentinel := errors.New("callback failure")
			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
						if r != "callback panic" {
							t.Fatalf("unexpected panic: %v", r)
						}
					}
				}()
				err = manager.WithinTransaction(context.Background(), func(ctx context.Context) error {
					if DBTXFromContext(ctx, db) == db {
						t.Fatal("transaction missing from context")
					}
					if mode == "panic" {
						panic("callback panic")
					}
					if mode == "error" {
						return sentinel
					}
					return nil
				})
			}()
			if mode == "success" {
				if err != nil || state.commits != 1 || state.rollbacks != 0 {
					t.Fatalf("success: err=%v state=%+v", err, state)
				}
			} else {
				if state.rollbacks != 1 || state.commits != 0 {
					t.Fatalf("transaction leaked: %+v", state)
				}
			}
			if mode == "error" && !errors.Is(err, sentinel) {
				t.Fatalf("err=%v", err)
			}
			if mode == "panic" && !panicked {
				t.Fatal("callback panic swallowed")
			}
		})
	}
}
