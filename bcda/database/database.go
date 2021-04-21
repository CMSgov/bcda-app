package database

import (
	"context"
	"database/sql"
)

// Row is an interface around https://golang.org/pkg/database/sql/#Row.
// It can be implemented by other database libraries (like pgx)
type Row interface {
	Scan(dest ...interface{}) error
}

// Rows is an interface around https://golang.org/pkg/database/sql/#Rows.
// It can be implemented by other database libraries (like pgx)
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...interface{}) error
}

// Result is an interface around https://golang.org/pkg/database/sql/#Result.
// It can be implemented by other database libraries (like pgx)
type Result interface {
	RowsAffected() (int64, error)
}

type Queryable interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) Row
}

type Executable interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error)
}

// DB is a wrapper around *sql.DB to allow us to implement our internal interfaces
type DB struct {
	*sql.DB
}

var (
	_ Queryable  = &DB{}
	_ Executable = &DB{}
)

func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return db.DB.QueryContext(ctx, query, args...)
}
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) Row {
	return db.DB.QueryRowContext(ctx, query, args...)
}
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return db.DB.ExecContext(ctx, query, args...)
}

// Tx is a wrapper around *sql.DB to allow us to implement our internal interfaces
type Tx struct {
	*sql.Tx
}

var (
	_ Queryable  = &Tx{}
	_ Executable = &Tx{}
)

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	return tx.Tx.QueryContext(ctx, query, args...)
}
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) Row {
	return tx.Tx.QueryRowContext(ctx, query, args...)
}
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return tx.Tx.ExecContext(ctx, query, args...)
}
