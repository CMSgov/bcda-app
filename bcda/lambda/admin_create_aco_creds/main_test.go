package main

import (
	"context"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/stretchr/testify/assert"
)

func TestHandleCreateACOCreds(t *testing.T) {
	ctx := context.Background()

	data := payload{ACOID: "TEST1234", IPs: []string{"1.2.3.4", "1.2.3.5"}}

	mockProvider := &auth.MockProvider{}
	mockProvider.On("FindAndCreateACOCredentials", data.ACOID, data.IPs).Return("creds\nstring", nil)

	s3Path, err := handleCreateACOCreds(ctx, data, mockProvider, &mockS3{}, "test-bucket")
	assert.Nil(t, err)
	assert.Equal(t, s3Path, "{\n\n}")
}
