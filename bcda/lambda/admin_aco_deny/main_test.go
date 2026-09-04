package main

import (
	"context"
	"testing"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
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

func TestGetAWSParams(t *testing.T) {
	dbURLName := "/bcda/local/sensitive/api/DATABASE_URL"
	slackParamName := "/slack/token/workflow-alerts"
	storedParams := map[string]string{
		dbURLName:      "db://url",
		slackParamName: "test-slack-token",
	}
	params, err := getAWSParams(context.Background(), &bcdaaws.MockSSMClient{Params: storedParams})
	assert.Nil(t, err)
	assert.Equal(t, "test-slack-token", params.SlackToken)
	assert.Equal(t, "db://url", params.DBURL)
	print(dbURLName)
}
