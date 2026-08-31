package filehandler

import (
	"context"
	"fmt"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

// S3FileHandler manages generic file operations on AWS S3.
type S3FileHandler struct {
	Client bcdaaws.CustomS3Client
	Logger logrus.FieldLogger
}

func (handler *S3FileHandler) Infof(format string, rest ...interface{}) {
	if handler.Logger != nil {
		handler.Logger.Infof(format, rest...)
	}
}

func (handler *S3FileHandler) Warningf(format string, rest ...interface{}) {
	if handler.Logger != nil {
		handler.Logger.Warningf(format, rest...)
	}
}

func (handler *S3FileHandler) Errorf(format string, rest ...interface{}) {
	if handler.Logger != nil {
		handler.Logger.Errorf(format, rest...)
	}
}

func (handler *S3FileHandler) ListFiles(ctx context.Context, bucket, prefix string) ([]s3types.Object, error) {
	handler.Infof("Listing objects in bucket %s, prefix %s", bucket, prefix)

	var objects []s3types.Object
	var continuationToken *string

	for {
		resp, err := handler.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: continuationToken,
		})
		if err != nil {
			handler.Errorf("Failed to list objects in S3 bucket %s, prefix %s: %s", bucket, prefix, err)
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

func (handler *S3FileHandler) OpenFileBytes(ctx context.Context, filePath string) ([]byte, error) {
	handler.Infof("Opening file %s", filePath)
	bucket, file := bcdaaws.ParseS3Uri(filePath)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	}

	output, err := handler.Client.HeadObject(ctx, input)
	if err != nil {
		return nil, err
	}
	if output == nil || output.ContentLength == nil || *output.ContentLength <= 0 {
		return []byte{}, fmt.Errorf("file %s is empty", filePath)
	}

	buff := make([]byte, int(*output.ContentLength))
	w := manager.NewWriteAtBuffer(buff)

	downloader := manager.NewDownloader(handler.Client)
	numBytes, err := downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})
	if err != nil {
		return nil, err
	}

	if handler.Logger != nil {
		handler.Logger.WithField("file_size_bytes", numBytes).Infof("file downloaded: size=%d", numBytes)
	}

	return w.Bytes(), nil
}

func (handler *S3FileHandler) Delete(ctx context.Context, filePath string) error {
	bucket, path := bcdaaws.ParseS3Uri(filePath)

	_, err := handler.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		handler.Errorf("file %s failed to clean up properly, error occurred while deleting object: %v", filePath, err)
		return err
	}

	return nil
}
