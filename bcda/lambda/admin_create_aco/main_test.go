package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/pborman/uuid"
	"github.com/slack-go/slack"
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

	params, err := getAWSParams()
	if err != nil {
		assert.FailNow(c.T(), "Failed to get AWS Params")
	}

	conn, err := pgx.Connect(c.ctx, params.dbURL)
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

type mockNotifier struct {
	Notifier
}

func (m *mockNotifier) PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
	return channelID, time.Now().String(), nil
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

	err = handleCreateACO(ctx, mockConn, data, id, &mockNotifier{})
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

	err = handleCreateACO(ctx, mockConn, data, id, &mockNotifier{})
	assert.ErrorContains(t, err, "test error")
}

func (c *HandleCreateACOTestSuite) TestHandleCreateACOInvalidCMSID() {
	data := payload{"TESTACO", "12345678", nil}
	id := uuid.NewRandom()

	err := handleCreateACO(c.ctx, c.tx, data, id, &mockNotifier{})
	assert.ErrorContains(c.T(), err, "invalid")
}

func (c *HandleCreateACOTestSuite) TestHandleCreateACOMissingName() {
	data := payload{"", "TEST510", nil}
	id := uuid.NewRandom()

	err := handleCreateACO(c.ctx, c.tx, data, id, &mockNotifier{})
	assert.ErrorContains(c.T(), err, "ACO name must be provided")
}

func (c *HandleCreateACOTestSuite) TestHandleCreateACOMissingCMSID() {
	data := payload{"Test ACO 5", "", nil}
	id := uuid.NewRandom()

	err := handleCreateACO(c.ctx, c.tx, data, id, &mockNotifier{})
	assert.ErrorContains(c.T(), err, "CMSID must be provided")
}
