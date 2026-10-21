package bcdaaws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListFiles(t *testing.T) {
	t.Run("success returning objects single page", func(t *testing.T) {
		client := &ConfigurableMockS3Client{
			ListObjectsV2Fn: func(_ context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
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
		client := &ConfigurableMockS3Client{
			ListObjectsV2Fn: func(_ context.Context, input *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
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
		client := &ConfigurableMockS3Client{
			ListObjectsV2Fn: func(_ context.Context, _ *s3.ListObjectsV2Input) (*s3.ListObjectsV2Output, error) {
				return nil, mockErr
			},
		}

		objects, err := ListFiles(context.Background(), client, "test-bucket", "test-prefix/")
		require.ErrorIs(t, err, mockErr)
		assert.Nil(t, objects)
	})
}

func TestOpenFileAsScanner(t *testing.T) {
	content := "hello world"
	client := &ConfigurableMockS3Client{
		HeadObjectFn: func(_ context.Context, _ *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
			return &s3.HeadObjectOutput{
				ContentLength: aws.Int64(int64(len(content))),
			}, nil
		},
		GetObjectFn: func(_ context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
			return &s3.GetObjectOutput{
				Body:          io.NopCloser(strings.NewReader(content)),
				ContentLength: aws.Int64(int64(len(content))),
				ContentRange:  aws.String(fmt.Sprintf("bytes 0-%d/%d", len(content)-1, len(content))),
			}, nil
		},
	}

	fileBytes, _, err := OpenFileAsScanner(t.Context(), client, "mock-file-path")
	assert.NoError(t, err)
	assert.NotNil(t, fileBytes)
}

func TestOpenFileAsScanner_BadFileError(t *testing.T) {
	cfg, ctx := testUtils.TestAWSConfig(t)
	client := testUtils.TestS3Client(t, cfg)
	fileBytes, f, err := OpenFileAsScanner(ctx, client, "bad-file/bad-name.txt")
	assert.ErrorContains(t, err, "failed to download file bad-file/bad-name.txt")
	assert.Nil(t, fileBytes)
	assert.Nil(t, f)
}

func TestOpenFileAsBytes(t *testing.T) {
	path := "s3://test-bucket/test-prefix/test-file.txt"

	t.Run("success reading bytes", func(t *testing.T) {
		path := "../../shared_files/csv"
		tmpPath, cleanup := testUtils.CopyToTemporaryDirectory(t, path)
		defer cleanup()
		cfg, ctx := testUtils.TestAWSConfig(t)
		client := testUtils.TestS3Client(t, cfg)

		bucketName, cleanup := testUtils.CopyToS3(t, tmpPath)
		defer cleanup()

		bytes, err := OpenFileAsBytes(ctx, client, filepath.Join(bucketName, tmpPath, "valid.csv"))
		require.NoError(t, err)
		assert.NotEmpty(t, bytes)
	})

	t.Run("head object error", func(t *testing.T) {
		mockErr := errors.New("s3 head object error")
		client := &ConfigurableMockS3Client{
			HeadObjectFn: func(_ context.Context, _ *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				return nil, mockErr
			},
		}

		bytes, err := OpenFileAsBytes(context.Background(), client, path)
		require.ErrorIs(t, err, mockErr)
		assert.Nil(t, bytes)
	})

	t.Run("download error", func(t *testing.T) {
		mockErr := errors.New("s3 download body error")
		client := &ConfigurableMockS3Client{
			HeadObjectFn: func(_ context.Context, _ *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
				return &s3.HeadObjectOutput{
					ContentLength: aws.Int64(10),
				}, nil
			},
			GetObjectFn: func(_ context.Context, _ *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
				return nil, mockErr
			},
		}

		bytes, err := OpenFileAsBytes(context.Background(), client, path)
		require.ErrorIs(t, err, mockErr)
		assert.Nil(t, bytes)
	})
}

func TestDelete(t *testing.T) {
	path := "../../shared_files/csv"
	tmpPath, cleanup := testUtils.CopyToTemporaryDirectory(t, path)
	defer cleanup()

	t.Run("success deleting object", func(t *testing.T) {
		bucketName, cleanup := testUtils.CopyToS3(t, tmpPath)
		defer cleanup()

		cfg, ctx := testUtils.TestAWSConfig(t)
		client := testUtils.TestS3Client(t, cfg)

		err := Delete(ctx, client, filepath.Join(bucketName, tmpPath, "valid.csv"))
		require.NoError(t, err)
	})

	t.Run("delete object error", func(t *testing.T) {
		mockErr := errors.New("delete object permission denied")
		client := &ConfigurableMockS3Client{
			DeleteObjectFn: func(_ context.Context, _ *s3.DeleteObjectInput) (*s3.DeleteObjectOutput, error) {
				return nil, mockErr
			},
		}

		err := Delete(context.Background(), client, path)
		require.ErrorIs(t, err, mockErr)
	})
}
