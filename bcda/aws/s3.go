package bcdaaws

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func ListFiles(ctx context.Context, client CustomS3Client, bucket, prefix string) ([]types.Object, error) {
	var objects []types.Object
	var continuationToken *string

	for {
		resp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in S3 bucket %s, prefix %s: %w", bucket, prefix, err)
		}

		objects = append(objects, resp.Contents...)

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		continuationToken = resp.NextContinuationToken
	}

	return objects, nil
}

// OpenFileAsScanner opens a file from S3 and returns a bufio.Scanner for reading its contents line by line.
// It also returns a cleanup function and an error if any occurred.
// Be warned this is not memory efficient for large files as it reads the entire file into memory.
func OpenFileAsScanner(ctx context.Context, client CustomS3Client, filePath string) (*bufio.Scanner, func(), error) {
	byte_arr, err := OpenFileAsBytes(ctx, client, filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download %s, err: %w", filePath, err)
	}

	sc := bufio.NewScanner(bytes.NewReader(byte_arr))
	return sc, func() {}, err
}

func OpenFileAsBytes(ctx context.Context, client CustomS3Client, filePath string) ([]byte, error) {
	bucket, file := ParseS3Uri(filePath)

	manager := transfermanager.New(client)
	output, err := manager.GetObject(ctx, &transfermanager.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file %s, err: %w", filePath, err)
	}
	defer func() {
		if closer, ok := output.Body.(io.Closer); ok {
			closer.Close()
		}
	}()

	bytes, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s, err: %w", filePath, err)
	}
	if len(bytes) == 0 {
		return nil, fmt.Errorf("file %s is empty", filePath)
	}

	return bytes, nil
}

func Delete(ctx context.Context, client CustomS3Client, filePath string) error {
	bucket, path := ParseS3Uri(filePath)

	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("file %s failed to clean up properly, error occurred while deleting object: %w", filePath, err)
	}

	return nil
}
