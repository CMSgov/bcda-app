package main

import (
	"context"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

type mockNotifier struct {
	Notifier
}

func (m *mockNotifier) PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
	return channelID, time.Now().String(), nil
}

func TestHandleCreateACO(t *testing.T) {
	ctx := context.Background()

	data := payload{"TestACO", "T0006"}

	err := handleCreateACO(ctx, data, &mockNotifier{})
	assert.Nil(t, err)
}
