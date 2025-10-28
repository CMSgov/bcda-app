package main

import (
	"context"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestHandleCreateACOCreds(t *testing.T) {
	ctx := context.Background()

	data := payload{ACOID: "TEST1234", IPs: []string{"1.2.3.4", "1.2.3.5"}}

	mockProvider := &auth.MockProvider{}
	mockProvider.On("FindAndCreateACOCredentials", data.ACOID, data.IPs).Return("creds\nstring", nil)

	client := testUtils.TestS3Client(t, testUtils.TestAWSConfig(t))

	_, err := client.CreateBucket(t.Context(), &s3.CreateBucketInput{
		Bucket: aws.String("test-bucket"),
	})
	assert.Nil(t, err)

	s3Path, err := handleCreateACOCreds(ctx, data, mockProvider, client, "test-bucket")
	assert.Nil(t, err)
	assert.Equal(t, s3Path, "test-bucket/TEST1234-creds")
}
