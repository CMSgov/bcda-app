package main

import (
	"context"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

type mockNotifier struct {
	Notifier
}

func (m *mockNotifier) PostMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
	return channelID, time.Now().String(), nil
}

func TestHandleCreateACOCreds(t *testing.T) {
	ctx := context.Background()

	data := payload{ACOID: "TEST1234", IPs: []string{"1.2.3.4", "1.2.3.5"}}

	mockProvider := &auth.MockProvider{}
	mockProvider.On("FindAndCreateACOCredentials", data.ACOID, data.IPs).Return("creds\nstring", nil)

	s3Path, err := handleCreateACOCreds(ctx, data, mockProvider, &mockS3{}, &mockNotifier{}, "test-bucket")
	assert.Nil(t, err)
	assert.Equal(t, s3Path, "{\n\n}")
}
