package database

import (
	"context"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/stretchr/testify/assert"
)

func TestDBOperations(t *testing.T) {
	c := Connect()
	var q Queryable = &DB{c}
	var e Executable = &DB{c}
	rows, err := q.QueryContext(context.Background(), constants.TestSelectNowSQL)
	assert.NoError(t, err)

	var result string
	assert.True(t, rows.Next())
	assert.NoError(t, rows.Scan(&result))
	assert.NotEmpty(t, result)
	assert.NoError(t, rows.Close())

	assert.NoError(t, q.QueryRowContext(context.Background(), constants.TestSelectNowSQL).Scan(&result))
	assert.NotEmpty(t, result)

	res, err := e.ExecContext(context.Background(), constants.TestSelectNowSQL)
	assert.NoError(t, err)
	affected, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, affected)
}

func TestTxOperations(t *testing.T) {
	c := Connect()
	tx, err := c.Begin()
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, tx.Rollback())
	}()

	var q Queryable = &Tx{tx}
	var e Executable = &Tx{tx}
	rows, err := q.QueryContext(context.Background(), constants.TestSelectNowSQL)
	assert.NoError(t, err)

	var result string
	assert.True(t, rows.Next())
	assert.NoError(t, rows.Scan(&result))
	assert.NotEmpty(t, result)
	assert.NoError(t, rows.Close())

	assert.NoError(t, q.QueryRowContext(context.Background(), constants.TestSelectNowSQL).Scan(&result))
	assert.NotEmpty(t, result)

	res, err := e.ExecContext(context.Background(), constants.TestSelectNowSQL)
	assert.NoError(t, err)
	affected, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, affected)
}
