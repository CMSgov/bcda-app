package database

import (
	"context"
	"database/sql"
)

type PgxTx struct {
	*sql.Tx
}

func (tx *PgxTx) QueryContext(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := tx.Tx.QueryContext(ctx, query, args...)
	return &pgxRows{rows}, err
}

func (tx *PgxTx) QueryRowContext(ctx context.Context, query string, args ...interface{}) Row {
	return tx.Tx.QueryRowContext(ctx, query, args...)
}

func (tx *PgxTx) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := tx.Tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{result}, nil
}

type pgxRows struct {
	*sql.Rows
}

var _ Rows = &pgxRows{}

func (r *pgxRows) Close() error {
	return r.Rows.Close()
}

func (r *pgxRows) Next() bool {
	return r.Rows.Next()
}

func (r *pgxRows) Scan(dest ...interface{}) error {
	return r.Rows.Scan(dest...)
}

type pgxResult struct {
	sql.Result
}

var _ Result = &pgxResult{}

func (r *pgxResult) RowsAffected() (int64, error) {
	return r.Result.RowsAffected()
}
