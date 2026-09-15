package bcdaaws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type CustomS3Client interface {
	CreateBucket(ctx context.Context, input *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjects(ctx context.Context, input *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error)
	ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)

	// needed for transfer manager S3APIClient interface
	AbortMultipartUpload(context.Context, *s3.AbortMultipartUploadInput, ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error)
	CompleteMultipartUpload(context.Context, *s3.CompleteMultipartUploadInput, ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error)
	CreateMultipartUpload(context.Context, *s3.CreateMultipartUploadInput, ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error)
	UploadPart(context.Context, *s3.UploadPartInput, ...func(*s3.Options)) (*s3.UploadPartOutput, error)
}

type MockS3Client struct{}

func (m *MockS3Client) CreateBucket(ctx context.Context, input *s3.CreateBucketInput, optFns ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	return &s3.CreateBucketOutput{}, nil
}

func (m *MockS3Client) HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	return &s3.HeadObjectOutput{}, nil
}

func (m *MockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return &s3.GetObjectOutput{}, nil
}

func (m *MockS3Client) ListObjects(ctx context.Context, input *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	return &s3.ListObjectsOutput{}, nil
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return &s3.ListObjectsV2Output{}, nil
}

func (m *MockS3Client) PutObject(ctx context.Context, input *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return &s3.PutObjectOutput{}, nil
}

func (m *MockS3Client) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return &s3.DeleteObjectOutput{}, nil
}

func (m *MockS3Client) AbortMultipartUpload(ctx context.Context, input *s3.AbortMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.AbortMultipartUploadOutput, error) {
	return &s3.AbortMultipartUploadOutput{}, nil
}
func (m *MockS3Client) CompleteMultipartUpload(ctx context.Context, input *s3.CompleteMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CompleteMultipartUploadOutput, error) {
	return &s3.CompleteMultipartUploadOutput{}, nil
}
func (m *MockS3Client) CreateMultipartUpload(ctx context.Context, input *s3.CreateMultipartUploadInput, optFns ...func(*s3.Options)) (*s3.CreateMultipartUploadOutput, error) {
	return &s3.CreateMultipartUploadOutput{}, nil
}
func (m *MockS3Client) UploadPart(ctx context.Context, input *s3.UploadPartInput, optFns ...func(*s3.Options)) (*s3.UploadPartOutput, error) {
	return &s3.UploadPartOutput{}, nil
}

type ConfigurableMockS3Client struct {
	MockS3Client
	ListObjectsV2Fn func(ctx context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	HeadObjectFn    func(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	GetObjectFn     func(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	DeleteObjectFn  func(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
}

func (m *ConfigurableMockS3Client) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.ListObjectsV2Fn != nil {
		return m.ListObjectsV2Fn(ctx, input)
	}
	return m.MockS3Client.ListObjectsV2(ctx, input, optFns...)
}

func (m *ConfigurableMockS3Client) HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.HeadObjectFn != nil {
		return m.HeadObjectFn(ctx, input)
	}
	return m.MockS3Client.HeadObject(ctx, input, optFns...)
}

func (m *ConfigurableMockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.GetObjectFn != nil {
		return m.GetObjectFn(ctx, input)
	}
	return m.MockS3Client.GetObject(ctx, input, optFns...)
}

func (m *ConfigurableMockS3Client) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.DeleteObjectFn != nil {
		return m.DeleteObjectFn(ctx, input)
	}
	return m.MockS3Client.DeleteObject(ctx, input, optFns...)
}
