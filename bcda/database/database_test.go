package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDBOperations(t *testing.T) {
	var q Queryable = &DB{Connection}
	var e Executable = &DB{Connection}
	rows, err := q.QueryContext(context.Background(), "SELECT NOW()")
	assert.NoError(t, err)

	var result string
	assert.True(t, rows.Next())
	assert.NoError(t, rows.Scan(&result))
	assert.NotEmpty(t, result)
	assert.NoError(t, rows.Close())

	assert.NoError(t, q.QueryRowContext(context.Background(), "SELECT NOW()").Scan(&result))
	assert.NotEmpty(t, result)

	res, err := e.ExecContext(context.Background(), "SELECT NOW()")
	assert.NoError(t, err)
	affected, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, affected)
}

func TestTxOperations(t *testing.T) {
	tx, err := Connection.Begin()
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, tx.Rollback())
	}()

	var q Queryable = &Tx{tx}
	var e Executable = &Tx{tx}
	rows, err := q.QueryContext(context.Background(), "SELECT NOW()")
	assert.NoError(t, err)

	var result string
	assert.True(t, rows.Next())
	assert.NoError(t, rows.Scan(&result))
	assert.NotEmpty(t, result)
	assert.NoError(t, rows.Close())

	assert.NoError(t, q.QueryRowContext(context.Background(), "SELECT NOW()").Scan(&result))
	assert.NotEmpty(t, result)

	res, err := e.ExecContext(context.Background(), "SELECT NOW()")
	assert.NoError(t, err)
	affected, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, affected)
}
