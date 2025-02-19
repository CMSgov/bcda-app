package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/pborman/uuid"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

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

	data := payload{"TESTACO", "TEST002"}
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

	data := payload{"TESTACO", "TEST002"}
	id := uuid.NewRandom()

	err = handleCreateACO(ctx, mockConn, data, id, &mockNotifier{})
	assert.ErrorContains(t, err, "test error")
}
