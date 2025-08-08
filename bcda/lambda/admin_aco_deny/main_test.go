package main

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestHandleACODenies(t *testing.T) {
	ctx := context.Background()

	mockConn, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mockConn.Close(ctx)

	mockConn.ExpectExec("^UPDATE acos SET termination_details = (.+)").
		WithArgs(mockTermination{}, testACODenies).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	err = handleACODenies(ctx, mockConn, payload{testACODenies})
	assert.Nil(t, err)
}
