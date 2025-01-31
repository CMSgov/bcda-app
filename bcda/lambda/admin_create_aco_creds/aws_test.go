package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
)

type mockKMS struct {
	kmsiface.KMSAPI
}

func (m *mockKMS) ListAliases(input *kms.ListAliasesInput) (*kms.ListAliasesOutput, error) {
	return &kms.ListAliasesOutput{
		Aliases: []*kms.AliasListEntry{
			{
				AliasName:   aws.String(kmsAliasName),
				TargetKeyId: aws.String("test-id"),
			},
		},
	}, nil
}

type mockS3 struct {
	s3iface.S3API
}

func (m *mockS3) PutObject(*s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func TestGetKMSID(t *testing.T) {
	mock := &mockKMS{}

	result, err := getKMSID(mock)
	assert.Nil(t, err)
	assert.Equal(t, result, "test-id")
}

func TestPutObject(t *testing.T) {
	mock := &mockS3{}

	result, err := putObject(mock, "test-creds", "kms-id")
	assert.Nil(t, err)
	assert.Equal(t, result, "{\n\n}")
}
