package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/stdlib"
	"github.com/stretchr/testify/assert"
)

func TestPgxTxOperations(t *testing.T) {
	conn, err := stdlib.AcquireConn(Connection)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, conn.Close())
	}()

	tx, err := conn.Begin()
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, tx.Rollback())
	}()

	var q Queryable = &PgxTx{tx}
	var e Executable = &PgxTx{tx}
	rows, err := q.QueryContext(context.Background(), "SELECT NOW()")
	assert.NoError(t, err)

	var result pgtype.Timestamptz
	assert.True(t, rows.Next())
	assert.NoError(t, rows.Scan(&result))
	assert.False(t, result.Time.IsZero(), "Time should be set")
	assert.NoError(t, rows.Close())

	assert.NoError(t, q.QueryRowContext(context.Background(), "SELECT NOW()").Scan(&result))
	assert.False(t, result.Time.IsZero(), "Time should be set")

	res, err := e.ExecContext(context.Background(), "SELECT NOW()")
	assert.NoError(t, err)
	affected, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, affected)
}
