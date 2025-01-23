package main

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

type mockNotifier struct {
	Notifier
}

func (m *mockNotifier) PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
	return channelID, time.Now().String(), nil
}

func TestHandleACODenies(t *testing.T) {
	ctx := context.Background()

	mockConn, err := pgxmock.NewConn()
	assert.Nil(t, err)
	defer mockConn.Close(ctx)

	mockConn.ExpectExec("^UPDATE acos SET termination_details = (.+)").
		WithArgs(mockTermination{}, testACODenies).
		WillReturnResult(pgxmock.NewResult("UPDATE", 3))

	err = handleACODenies(ctx, mockConn, payload{testACODenies}, &mockNotifier{})
	assert.Nil(t, err)
}
