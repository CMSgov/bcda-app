package database

import (
	"context"

	"github.com/jackc/pgx"
)

type PgxTx struct {
	*pgx.Tx
}

func (tx *PgxTx) QueryContext(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := tx.QueryEx(ctx, query, nil, args...)
	return &pgxRows{rows}, err
}

func (tx *PgxTx) QueryRowContext(ctx context.Context, query string, args ...interface{}) Row {
	return tx.QueryRowEx(ctx, query, nil, args...)
}

func (tx *PgxTx) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := tx.ExecEx(ctx, query, nil, args...)
	if err != nil {
		return nil, err
	}
	return &pgxResult{result}, nil
}

type pgxRows struct {
	*pgx.Rows
}

var _ Rows = &pgxRows{}

func (r *pgxRows) Close() error {
	r.Rows.Close()
	return nil
}

func (r *pgxRows) Next() bool {
	return r.Rows.Next()
}

func (r *pgxRows) Scan(dest ...interface{}) error {
	return r.Rows.Scan(dest...)
}

type pgxResult struct {
	pgx.CommandTag
}

var _ Result = &pgxResult{}

func (r *pgxResult) RowsAffected() (int64, error) {
	return r.CommandTag.RowsAffected(), nil
}
