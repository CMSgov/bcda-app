package database

import (
	"context"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/stretchr/testify/assert"
)

func TestPgxTxOperations(t *testing.T) {
	db := Connect()
	defer db.Close()

	conn, err := db.Conn(context.Background())
	assert.NoError(t, err)
	defer conn.Close()

	tx, err := conn.BeginTx(context.Background(), nil)
	assert.NoError(t, err)
	defer func() {
		assert.NoError(t, tx.Rollback())
	}()

	var q Queryable = &PgxTx{tx}
	var e Executable = &PgxTx{tx}
	rows, err := q.QueryContext(context.Background(), constants.TestSelectNowSQL)
	assert.NoError(t, err)

	var result time.Time
	assert.True(t, rows.Next())
	assert.NoError(t, rows.Scan(&result))
	assert.False(t, result.IsZero(), "Time should be set")
	assert.NoError(t, rows.Close())

	assert.NoError(t, q.QueryRowContext(context.Background(), constants.TestSelectNowSQL).Scan(&result))
	assert.False(t, result.IsZero(), "Time should be set")

	res, err := e.ExecContext(context.Background(), constants.TestSelectNowSQL)
	assert.NoError(t, err)
	affected, err := res.RowsAffected()
	assert.NoError(t, err)
	assert.EqualValues(t, 1, affected)
}
