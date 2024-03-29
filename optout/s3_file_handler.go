package optout

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sirupsen/logrus"
)

// S3FileHandler manages files located on AWS S3.
type S3FileHandler struct {
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
	fmt.Printf(format, rest...)
	handler.Logger.Infof(format, rest...)
}

func (handler *S3FileHandler) Warningf(format string, rest ...interface{}) {
	fmt.Printf(format, rest...)
	handler.Logger.Warningf(format, rest...)
}

func (handler *S3FileHandler) Errorf(format string, rest ...interface{}) {
	fmt.Printf(format, rest...)
	handler.Logger.Errorf(format, rest...)
}

func (handler *S3FileHandler) LoadOptOutFiles(path string) (suppressList *[]*OptOutFilenameMetadata, skipped int, err error) {
	var result []*OptOutFilenameMetadata
	bucket, prefix := parseS3Uri(path)

	sess, err := handler.createSession()
	if err != nil {
		handler.Errorf("Failed to create S3 session: %s\n", err)
		return &result, skipped, err
	}

	svc := s3.New(sess)

	handler.Infof("Listing objects in bucket %s, prefix %s\n", bucket, prefix)

	resp, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		handler.Errorf("Failed to list objects in S3 bucket %s, prefix %s: %s\n", bucket, prefix, err)
		return &result, skipped, err
	}

	for _, obj := range resp.Contents {
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

func (handler *S3FileHandler) OpenFile(metadata *OptOutFilenameMetadata) (*bufio.Scanner, func(), error) {
	handler.Infof("Opening file %s\n", metadata.FilePath)
	bucket, file := parseS3Uri(metadata.FilePath)

	sess, err := handler.createSession()
	if err != nil {
		return nil, nil, err
	}

	downloader := s3manager.NewDownloader(sess)
	buff := &aws.WriteAtBuffer{}
	numBytes, err := downloader.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})

	if err != nil {
		handler.Errorf("Failed to download bucket %s, key %s\n", bucket, file)
		return nil, nil, err
	}

	handler.Infof("file downloaded: size=%d\n", numBytes)

	byte_arr := buff.Bytes()
	sc := bufio.NewScanner(bytes.NewReader(byte_arr))
	return sc, func() {}, err
}

func (handler *S3FileHandler) CleanupOptOutFiles(suppresslist []*OptOutFilenameMetadata) error {
	sess, err := handler.createSession()
	if err != nil {
		return err
	}

	errCount := 0
	for _, suppressionFile := range suppresslist {
		if !suppressionFile.Imported {
			// Don't do anything. The S3 bucket should have a retention policy that
			// automatically cleans up files after a specified period of time,
			handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", suppressionFile)
			continue
		}

		handler.Infof("Cleaning up file %s\n", suppressionFile)
		bucket, file := parseS3Uri(suppressionFile.FilePath)

		svc := s3.New(sess)
		_, err = svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(file)})

		if err != nil {
			handler.Errorf("File %s failed to clean up properly, error occurred while deleting object: %v\n", suppressionFile, err)
			errCount++
			continue
		}

		err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(file),
		})

		if err != nil {
			handler.Errorf("File %s failed to clean up properly, error occurred while waiting for object to be deleted: %v\n", suppressionFile, err)
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

// Creates a new AWS S3 session. If the handler is given a custom S3 endpoint
// and/or IAM role ARN to assume, the new session connects using those parameters.
func (handler *S3FileHandler) createSession() (*session.Session, error) {
	sess := session.Must(session.NewSession())

	config := aws.Config{
		Region: aws.String("us-east-1"),
	}

	if handler.Endpoint != "" {
		config.S3ForcePathStyle = aws.Bool(true)
		config.Endpoint = &handler.Endpoint
	}

	if handler.AssumeRoleArn != "" {
		config.Credentials = stscreds.NewCredentials(
			sess,
			handler.AssumeRoleArn,
		)
	}

	return session.NewSessionWithOptions(session.Options{
		Config: config,
	})
}

// Parses an S3 URI and returns the bucket and key.
//
// @example:
//   input: s3://my-bucket/path/to/file
//   output: "my-bucket", "path/to/file"
//
// @example
//   input: s3://my-bucket
//   output: "my-bucket", ""
//
func parseS3Uri(str string) (bucket string, key string) {
	workingString := strings.TrimPrefix(str, "s3://")
	resultArr := strings.SplitN(workingString, "/", 2)

	if len(resultArr) == 1 {
		return resultArr[0], ""
	}

	return resultArr[0], resultArr[1]
}
