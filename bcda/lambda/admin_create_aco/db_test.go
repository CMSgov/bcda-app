package main

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/pborman/uuid"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

var (
	cms_id       = "TEST002"
	testACO      = &models.ACO{UUID: uuid.NewRandom(), CMSID: &cms_id, Name: "TESTACO", ClientID: "", TerminationDetails: &models.Termination{}}
	literalRegex = regexp.QuoteMeta(insertACOQuery)
)

func TestCreateACOSuccess(t *testing.T) {
	ctx := context.Background()
	mockConn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer mockConn.Close(ctx)

	mockConn.ExpectExec(literalRegex).
		WithArgs(testACO.UUID, testACO.CMSID, testACO.ClientID, testACO.Name, testACO.TerminationDetails).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = createACO(ctx, mockConn, *testACO)
	assert.Nil(t, err)
}

func TestCreateACOQueryFailure(t *testing.T) {
	ctx := context.Background()
	mockConn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatal(err)
	}
	defer mockConn.Close(ctx)

	mockConn.ExpectExec(literalRegex).
		WithArgs(testACO.UUID, testACO.CMSID, testACO.ClientID, testACO.Name, testACO.TerminationDetails).
		WillReturnError(errors.New("test error"))

	err = createACO(ctx, mockConn, *testACO)
	assert.ErrorContains(t, err, "test error")
}
