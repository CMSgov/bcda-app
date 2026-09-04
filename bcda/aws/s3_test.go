package bcdaaws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type configurableMockS3Client struct {
	MockS3Client
	listObjectsFn   func(ctx context.Context, input *s3.ListObjectsInput) (*s3.ListObjectsOutput, error)
	listObjectsV2Fn func(ctx context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error)
	headObjectFn    func(ctx context.Context, input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error)
	getObjectFn     func(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
	deleteObjectFn  func(ctx context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error)
}

func (m *configurableMockS3Client) ListObjects(ctx context.Context, input *s3.ListObjectsInput, optFns ...func(*s3.Options)) (*s3.ListObjectsOutput, error) {
	if m.listObjectsFn != nil {
		return m.listObjectsFn(ctx, input)
	}
	return m.MockS3Client.ListObjects(ctx, input, optFns...)
}

func (m *configurableMockS3Client) ListObjectsV2(ctx context.Context, input *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.listObjectsV2Fn != nil {
		return m.listObjectsV2Fn(ctx, input)
	}
	return m.MockS3Client.ListObjectsV2(ctx, input, optFns...)
}

func (m *configurableMockS3Client) HeadObject(ctx context.Context, input *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectFn != nil {
		return m.headObjectFn(ctx, input)
	}
	return m.MockS3Client.HeadObject(ctx, input, optFns...)
}

func (m *configurableMockS3Client) GetObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectFn != nil {
		return m.getObjectFn(ctx, input)
	}
	return m.MockS3Client.GetObject(ctx, input, optFns...)
}

func (m *configurableMockS3Client) DeleteObject(ctx context.Context, input *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if m.deleteObjectFn != nil {
		return m.deleteObjectFn(ctx, input)
	}
	return m.MockS3Client.DeleteObject(ctx, input, optFns...)
}

func TestListFiles(t *testing.T) {
	t.Run("success returning objects single page", func(t *testing.T) {
		client := &configurableMockS3Client{
			listObjectsV2Fn: func(_ context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				assert.Equal(t, "test-bucket", *input.Bucket)
				assert.Equal(t, "test-prefix/", *input.Prefix)
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{Key: aws.String("test-prefix/file1.csv")},
					},
					IsTruncated: aws.Bool(false),
				}, nil
			},
		}

		objects, err := ListFiles(context.Background(), client, "test-bucket", "test-prefix/")
		require.NoError(t, err)
		require.Len(t, objects, 1)
		assert.Equal(t, "test-prefix/file1.csv", *objects[0].Key)
	})

	t.Run("success returning objects with pagination", func(t *testing.T) {
		callCount := 0
		client := &configurableMockS3Client{
			listObjectsV2Fn: func(_ context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				callCount++
				if callCount == 1 {
					assert.Nil(t, input.ContinuationToken)
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{Key: aws.String("test-prefix/file1.csv")},
						},
						IsTruncated:           aws.Bool(true),
						NextContinuationToken: aws.String("next-token"),
					}, nil
				}
				assert.Equal(t, "next-token", *input.ContinuationToken)
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{Key: aws.String("test-prefix/file2.csv")},
					},
					IsTruncated: aws.Bool(false),
				}, nil
			},
		}

		objects, err := ListFiles(context.Background(), client, "test-bucket", "test-prefix/")
		require.NoError(t, err)
		require.Len(t, objects, 2)
		assert.Equal(t, "test-prefix/file1.csv", *objects[0].Key)
		assert.Equal(t, "test-prefix/file2.csv", *objects[1].Key)
		assert.Equal(t, 2, callCount)
	})

	t.Run("error listing objects", func(t *testing.T) {
		mockErr := errors.New("s3 connection failed")
		client := &configurableMockS3Client{
			listObjectsV2Fn: func(_ context.Context, _ *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return nil, mockErr
			},
		}

		objects, err := ListFiles(context.Background(), client, "test-bucket", "test-prefix/")
		require.ErrorIs(t, err, mockErr)
		assert.Nil(t, objects)
	})
}

func TestOpenFileAsScanner(t *testing.T) {
	client := &configurableMockS3Client{}
	fileBytes, f, err := OpenFileAsScanner(t.Context(), client, "bad-file")
	assert.ErrorContains(t, err, "file bad-file is empty")
	assert.Nil(t, fileBytes)
	assert.Nil(t, f)
}

func TestOpenFileAsBytes(t *testing.T) {
	path := "s3://test-bucket/test-prefix/test-file.txt"

	t.Run("success reading bytes", func(t *testing.T) {
		content := "hello world"
		client := &configurableMockS3Client{
			headObjectFn: func(_ context.Context, _ *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{
					ContentLength: aws.Int64(int64(len(content))),
				}, nil
			},
			getObjectFn: func(_ context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body:          io.NopCloser(strings.NewReader(content)),
					ContentLength: aws.Int64(int64(len(content))),
					ContentRange:  aws.String(fmt.Sprintf("bytes 0-%d/%d", len(content)-1, len(content))),
				}, nil
			},
		}

		bytes, err := OpenFileAsBytes(context.Background(), client, path)
		require.NoError(t, err)
		assert.Equal(t, content, string(bytes))
	})

	t.Run("head object error", func(t *testing.T) {
		mockErr := errors.New("s3 head object error")
		client := &configurableMockS3Client{
			headObjectFn: func(_ context.Context, _ *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				return nil, mockErr
			},
		}

		bytes, err := OpenFileAsBytes(context.Background(), client, path)
		require.ErrorIs(t, err, mockErr)
		assert.Nil(t, bytes)
	})

	t.Run("file empty or zero content length", func(t *testing.T) {
		client := &configurableMockS3Client{
			headObjectFn: func(_ context.Context, _ *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{
					ContentLength: aws.Int64(0),
				}, nil
			},
		}

		bytes, err := OpenFileAsBytes(context.Background(), client, path)
		require.ErrorContains(t, err, "is empty")
		assert.Empty(t, bytes)
	})

	t.Run("download error", func(t *testing.T) {
		mockErr := errors.New("s3 download body error")
		client := &configurableMockS3Client{
			headObjectFn: func(_ context.Context, _ *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{
					ContentLength: aws.Int64(10),
				}, nil
			},
			getObjectFn: func(_ context.Context, _ *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return nil, mockErr
			},
		}

		bytes, err := OpenFileAsBytes(context.Background(), client, path)
		require.ErrorIs(t, err, mockErr)
		assert.Nil(t, bytes)
	})
}

func TestDelete(t *testing.T) {
	path := "s3://test-bucket/test-prefix/test-file.txt"

	// t.Run("success deleting object", func(t *testing.T) {
	// 	t.Setenv("S3_DELETE_TIMEOUT", "1")
	// 	client := &configurableMockS3Client{
	// 		deleteObjectFn: func(_ context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
	// 			assert.Equal(t, "test-bucket", *input.Bucket)
	// 			assert.Equal(t, "test-prefix/test-file.txt", *input.Key)
	// 			return &s3.DeleteObjectOutput{}, nil
	// 		},
	// 	}

	// 	err := Delete(context.Background(), client, path)
	// 	require.NoError(t, err)
	// })

	t.Run("error on timing out on delete", func(t *testing.T) {
		t.Setenv("S3_DELETE_TIMEOUT", "1")
		client := &configurableMockS3Client{
			deleteObjectFn: func(_ context.Context, input *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				assert.Equal(t, "test-bucket", *input.Bucket)
				assert.Equal(t, "test-prefix/test-file.txt", *input.Key)
				return &s3.DeleteObjectOutput{}, nil
			},
		}

		err := Delete(context.Background(), client, path)
		require.ErrorContains(t, err, "file s3://test-bucket/test-prefix/test-file.txt failed to clean up properly, error occurred while waiting for object deletion: exceeded max wait time for ObjectNotExists waiter")
	})

	t.Run("delete object error", func(t *testing.T) {
		t.Setenv("S3_DELETE_TIMEOUT", "1")
		mockErr := errors.New("delete object permission denied")
		client := &configurableMockS3Client{
			deleteObjectFn: func(_ context.Context, _ *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				return nil, mockErr
			},
		}

		err := Delete(context.Background(), client, path)
		require.ErrorIs(t, err, mockErr)
	})
}
