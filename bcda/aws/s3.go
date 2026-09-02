package bcdaaws

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
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

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	}

	output, err := client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get head object for %s, err: %w", filePath, err)
	}
	if output == nil || output.ContentLength == nil || *output.ContentLength <= 0 {
		return []byte{}, fmt.Errorf("file %s is empty", filePath)
	}

	buff := make([]byte, int(*output.ContentLength))
	w := manager.NewWriteAtBuffer(buff)

	downloader := manager.NewDownloader(client)
	_, err = downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file %s, err: %w", filePath, err)
	}

	return w.Bytes(), nil
}

func Delete(ctx context.Context, client CustomS3Client, filePath string) error {
	bucket, path := ParseS3Uri(filePath)
	timeoutDuration := time.Duration(utils.GetEnvInt("S3_DELETE_TIMEOUT", 60)) * time.Second

	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("file %s failed to clean up properly, error occurred while deleting object: %w", filePath, err)
	} else {
		err = s3.NewObjectNotExistsWaiter(client).Wait(
			ctx,
			&s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(path),
			},
			timeoutDuration,
		)
		if err != nil {
			return fmt.Errorf("file %s failed to clean up properly, error occurred while waiting for object deletion: %w", filePath, err)
		}
	}

	return nil
}
