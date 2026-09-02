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
	"github.com/sirupsen/logrus"
)

// S3Helper manages generic file operations on AWS S3
type S3Helper struct {
	Client CustomS3Client
	Logger logrus.FieldLogger
}

func (h *S3Helper) ListFiles(ctx context.Context, bucket, prefix string) ([]types.Object, error) {
	h.Logger.Infof("Listing objects in bucket %s, prefix %s", bucket, prefix)

	var objects []types.Object
	var continuationToken *string

	for {
		resp, err := h.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			h.Logger.Errorf("Failed to list objects in S3 bucket %s, prefix %s: %s", bucket, prefix, err)
			return nil, err
		}

		objects = append(objects, resp.Contents...)

		if resp.IsTruncated == nil || !*resp.IsTruncated {
			break
		}
		continuationToken = resp.NextContinuationToken
	}

	return objects, nil
}

func (h *S3Helper) OpenFileAsScanner(ctx context.Context, filePath string) (*bufio.Scanner, func(), error) {
	byte_arr, err := h.OpenFileAsBytes(ctx, filePath)
	if err != nil {
		h.Logger.Errorf("Failed to download %s\n", filePath)
		return nil, nil, err
	}

	sc := bufio.NewScanner(bytes.NewReader(byte_arr))
	return sc, func() {}, err
}

func (h *S3Helper) OpenFileAsBytes(ctx context.Context, filePath string) ([]byte, error) {
	h.Logger.Infof("Opening file %s", filePath)
	bucket, file := ParseS3Uri(filePath)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	}

	output, err := h.Client.HeadObject(ctx, input)
	if err != nil {
		return nil, err
	}
	if output == nil || output.ContentLength == nil || *output.ContentLength <= 0 {
		return []byte{}, fmt.Errorf("file %s is empty", filePath)
	}

	buff := make([]byte, int(*output.ContentLength))
	w := manager.NewWriteAtBuffer(buff)

	downloader := manager.NewDownloader(h.Client)
	numBytes, err := downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})
	if err != nil {
		return nil, err
	}

	if h.Logger != nil {
		h.Logger.WithField("file_size_bytes", numBytes).Infof("file downloaded: size=%d", numBytes)
	}

	return w.Bytes(), nil
}

func (h *S3Helper) Delete(ctx context.Context, filePath string) error {
	bucket, path := ParseS3Uri(filePath)
	timeoutDuration := time.Duration(utils.GetEnvInt("S3_DELETE_TIMEOUT", 60)) * time.Second

	_, err := h.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		h.Logger.Errorf("file %s failed to clean up properly, error occurred while deleting object: %v", filePath, err)
		return err
	} else {
		err = s3.NewObjectNotExistsWaiter(h.Client).Wait(
			ctx,
			&s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(path),
			},
			timeoutDuration,
		)
		if err != nil {
			h.Logger.Errorf("File %s failed to clean up properly, error occurred while waiting for object to be deleted: %v\n", filePath, err)
		}
	}

	return nil
}
