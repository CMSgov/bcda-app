package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HandleCreateACOTestSuite struct {
	suite.Suite
	tx   pgx.Tx
	conn pgx.Conn
	ctx  context.Context
}

func (c *HandleCreateACOTestSuite) SetupTest() {
	c.ctx = context.Background()

	conn, err := pgx.Connect(c.ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		assert.FailNow(c.T(), "Failed to setup pgx connection")
	}

	c.tx, err = conn.Begin(c.ctx)
	if err != nil {
		assert.FailNow(c.T(), "Failed to begin transaction")
	}
}

func (c *HandleCreateACOTestSuite) TeardownTest() {
	// cleanup
	err := c.tx.Rollback(c.ctx)
	if err != nil {
		assert.FailNow(c.T(), "Failed to rollback transaction")
	}

	c.conn.Close(context.Background())
}

func TestHandleCreateACOTestSuite(t *testing.T) {
	suite.Run(t, new(HandleCreateACOTestSuite))
}

type mockString struct{}

func (ms mockString) Match(v any) bool {
	_, ok := v.(string)
	return ok
}

type mockUuid struct{}

func (muuid mockUuid) Match(v any) bool {
	_, ok := v.(uuid.UUID)
	return ok
}

func TestHandleCreateACOSuccess(t *testing.T) {
	ctx := context.Background()

	mockConn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockConn.Close(ctx)

	mockConn.ExpectExec(literalRegex).
		WithArgs(
			mockUuid{},
			testACO.CMSID,
			mockString{}, // mocking random clientid
			testACO.Name,
			testACO.TerminationDetails,
		).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	data := payload{"TESTACO", "TEST002", nil}
	id := uuid.NewRandom()

	err = handleCreateACO(ctx, mockConn, data, id)
	assert.Nil(t, err)
}

func TestHandleCreateACOFailure(t *testing.T) {
	ctx := context.Background()

	mockConn, err := pgxmock.NewConn()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockConn.Close(ctx)

	mockConn.ExpectExec(literalRegex).
		WithArgs(
			mockUuid{},
			testACO.CMSID,
			mockString{},
			testACO.Name,
			testACO.TerminationDetails,
		).
		WillReturnError(errors.New("test error"))

	data := payload{"TESTACO", "TEST002", nil}
	id := uuid.NewRandom()

	err = handleCreateACO(ctx, mockConn, data, id)
	assert.ErrorContains(t, err, "test error")
}

func (c *HandleCreateACOTestSuite) TestHandleCreateACOInvalidCMSID() {
	data := payload{"TESTACO", "12345678", nil}
	id := uuid.NewRandom()

	err := handleCreateACO(c.ctx, c.tx, data, id)
	assert.ErrorContains(c.T(), err, "invalid")
}

func (c *HandleCreateACOTestSuite) TestHandleCreateACOMissingName() {
	data := payload{"", "TEST510", nil}
	id := uuid.NewRandom()

	err := handleCreateACO(c.ctx, c.tx, data, id)
	assert.ErrorContains(c.T(), err, "ACO name must be provided")
}

func (c *HandleCreateACOTestSuite) TestHandleCreateACOMissingCMSID() {
	data := payload{"Test ACO 5", "", nil}
	id := uuid.NewRandom()

	err := handleCreateACO(c.ctx, c.tx, data, id)
	assert.ErrorContains(c.T(), err, "CMSID must be provided")
}

func TestGetAWSParams(t *testing.T) {
	env := conf.GetEnv("ENV")

	cleanupParam1 := testUtils.SetParameter(t, fmt.Sprintf("/bcda/%s/sensitive/api/DATABASE_URL", env), "test-db-url")
	t.Cleanup(func() { cleanupParam1() })
	cleanupParam2 := testUtils.SetParameter(t, "/slack/token/workflow-alerts", "test-slack-token")
	t.Cleanup(func() { cleanupParam2() })

	params, err := getAWSParams(context.Background())

	assert.Nil(t, err)
	assert.Equal(t, "test-db-url", params.dbURL)
	assert.Equal(t, "test-slack-token", params.slackToken)
}
