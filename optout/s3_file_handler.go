package optout

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

// S3FileHandler manages files located on AWS S3.
type S3FileHandler struct {
	Client *s3.Client
	Logger logrus.FieldLogger
	// Optional S3 endpoint to use for connection.
	Endpoint string
	// Optional role to assume when connecting to S3.
	AssumeRoleArn string
}

// Define logger functions to ensure that logs get sent to:
// 1. Splunk (Logger.*)
// 2. stdout (Jenkins)
func (handler *S3FileHandler) Infof(format string, rest ...interface{}) {
	handler.Logger.Infof(format, rest...)
}

func (handler *S3FileHandler) Warningf(format string, rest ...interface{}) {
	handler.Logger.Warningf(format, rest...)
}

func (handler *S3FileHandler) Errorf(format string, rest ...interface{}) {
	handler.Logger.Errorf(format, rest...)
}

func (handler *S3FileHandler) LoadOptOutFiles(ctx context.Context, path string) (suppressList *[]*OptOutFilenameMetadata, skipped int, err error) {
	var result []*OptOutFilenameMetadata

	bucket, prefix := ParseS3Uri(path)
	s3Objects, err := handler.ListFiles(ctx, bucket, prefix)
	if err != nil {
		return &result, skipped, err
	}

	for _, obj := range s3Objects {
		metadata, err := ParseMetadata(*obj.Key)
		metadata.FilePath = fmt.Sprintf("s3://%s/%s", bucket, *obj.Key)
		metadata.DeliveryDate = *obj.LastModified

		if err != nil {
			// Skip files with a bad name.  An unknown file in this dir isn't a blocker
			handler.Warningf("Unknown file found: %s. Skipping.\n", metadata)
			skipped = skipped + 1
			continue
		}

		result = append(result, &metadata)
	}

	return &result, skipped, err
}

func (handler *S3FileHandler) ListFiles(ctx context.Context, bucket, prefix string) (objects []s3types.Object, err error) {
	handler.Infof("Listing objects in bucket %s, prefix %s\n", bucket, prefix)

	resp, err := handler.Client.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		handler.Errorf("Failed to list objects in S3 bucket %s, prefix %s: %s\n", bucket, prefix, err)
		return nil, err
	}

	return resp.Contents, nil
}

func (handler *S3FileHandler) OpenFile(ctx context.Context, metadata *OptOutFilenameMetadata) (*bufio.Scanner, func(), error) {
	byte_arr, err := handler.OpenFileBytes(ctx, metadata.FilePath)
	if err != nil {
		handler.Errorf("Failed to download %s\n", metadata.FilePath)
		return nil, nil, err
	}

	sc := bufio.NewScanner(bytes.NewReader(byte_arr))
	return sc, func() {}, err
}

func (handler *S3FileHandler) OpenFileBytes(ctx context.Context, filePath string) ([]byte, error) {
	handler.Infof("Opening file %s\n", filePath)
	bucket, file := ParseS3Uri(filePath)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	}

	output, err := handler.Client.HeadObject(ctx, input)
	if err != nil {
		return nil, err
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

	handler.Logger.WithField("file_size_bytes", numBytes).Infof("file downloaded: size=%d\n", numBytes)

	return buff, err
}

func (handler *S3FileHandler) CleanupOptOutFiles(ctx context.Context, suppresslist []*OptOutFilenameMetadata) error {
	errCount := 0

	for _, suppressionFile := range suppresslist {
		if !suppressionFile.Imported {
			// Don't do anything. The S3 bucket should have a retention policy that
			// automatically cleans up files after a specified period of time,
			handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", suppressionFile)
			continue
		}

		handler.Infof("Cleaning up file %s\n", suppressionFile)
		err := handler.Delete(ctx, suppressionFile.FilePath)

		if err != nil {
			errCount++
			continue
		}

		handler.Infof("File %s successfully ingested and deleted from S3.\n", suppressionFile)
	}

	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up\n", errCount)
	}

	return nil
}

func (handler *S3FileHandler) Delete(ctx context.Context, filePath string) error {
	bucket, path := ParseS3Uri(filePath)

	_, err := handler.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		handler.Errorf("file %s failed to clean up properly, error occurred while deleting object: %v\n", filePath, err)
		return err
	} else {
		err = s3.NewObjectNotExistsWaiter(handler.Client).Wait(
			ctx,
			&s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(path),
			},
			time.Minute,
		)
		if err != nil {
			handler.Errorf("File %s failed to clean up properly, error occurred while waiting for object to be deleted: %v\n", filePath, err)
		}
	}

	return err
}
