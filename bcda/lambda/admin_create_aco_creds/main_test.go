package main

import (
	"context"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

func TestHandleCreateACOCreds(t *testing.T) {
	ctx := context.Background()

	data := payload{ACOID: "TEST1234", IPs: []string{"1.2.3.4", "1.2.3.5"}}

	mockProvider := &auth.MockProvider{}
	mockProvider.On("FindAndCreateACOCredentials", data.ACOID, data.IPs).Return("creds\nstring", nil)

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	assert.Nil(t, err)
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	bucketInput := &s3.CreateBucketInput{
		Bucket: aws.String("test-bucket"),
	}
	_, err = client.CreateBucket(t.Context(), bucketInput)
	assert.Nil(t, err)

	s3Path, err := handleCreateACOCreds(ctx, data, mockProvider, client, "test-bucket")
	assert.Nil(t, err)
	assert.Equal(t, s3Path, "test-bucket/TEST1234-creds")
}
