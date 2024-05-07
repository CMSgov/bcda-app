package optout

import (
	"bufio"
	"bytes"
	"fmt"

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
	Session       *session.Session
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

	bucket, prefix := ParseS3Uri(path)
	s3Objects, err := handler.ListFiles(bucket, prefix)

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

func (handler *S3FileHandler) ListFiles(bucket, prefix string) (objects []*s3.Object, err error) {
	sess, err := handler.createSession()
	if err != nil {
		handler.Errorf("Failed to create S3 session: %s\n", err)
		return nil, err
	}

	svc := s3.New(sess)

	handler.Infof("Listing objects in bucket %s, prefix %s\n", bucket, prefix)

	resp, err := svc.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		handler.Errorf("Failed to list objects in S3 bucket %s, prefix %s: %s\n", bucket, prefix, err)
		return nil, err
	}

	return resp.Contents, nil
}

func (handler *S3FileHandler) OpenFile(metadata *OptOutFilenameMetadata) (*bufio.Scanner, func(), error) {
	byte_arr, err := handler.OpenFileBytes(metadata.FilePath)

	if err != nil {
		handler.Errorf("Failed to download %s\n", metadata.FilePath)
		return nil, nil, err
	}

	sc := bufio.NewScanner(bytes.NewReader(byte_arr))
	return sc, func() {}, err
}

func (handler *S3FileHandler) OpenFileBytes(filePath string) ([]byte, error) {
	handler.Infof("Opening file %s\n", filePath)
	bucket, file := ParseS3Uri(filePath)

	sess, err := handler.createSession()
	if err != nil {
		return nil, err
	}

	downloader := s3manager.NewDownloader(sess)
	buff := &aws.WriteAtBuffer{}
	numBytes, err := downloader.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file),
	})

	if err != nil {
		return nil, err
	}

	handler.Infof("file downloaded: size=%d\n", numBytes)
	byte_arr := buff.Bytes()
	return byte_arr, err
}

func (handler *S3FileHandler) CleanupOptOutFiles(suppresslist []*OptOutFilenameMetadata) error {
	errCount := 0
	for _, suppressionFile := range suppresslist {
		if !suppressionFile.Imported {
			// Don't do anything. The S3 bucket should have a retention policy that
			// automatically cleans up files after a specified period of time,
			handler.Warningf("File %s was not imported successfully. Skipping cleanup.\n", suppressionFile)
			continue
		}

		handler.Infof("Cleaning up file %s\n", suppressionFile)
		err := handler.Delete(suppressionFile.FilePath)

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

// Creates a new AWS S3 session. If the handler is given a custom S3 endpoint
// and/or IAM role ARN to assume, the new session connects using those parameters.
func (handler *S3FileHandler) createSession() (*session.Session, error) {
	if handler.Session != nil {
		return handler.Session, nil
	}

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

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})

	if err == nil {
		handler.Session = sess
	}

	return sess, err
}

func (handler *S3FileHandler) Delete(filePath string) error {
	sess, err := handler.createSession()

	if err != nil {
		handler.Errorf("File %s failed to clean up properly, error occurred while creating S3 session: %v\n", filePath, err)
		return err
	}

	bucket, path := ParseS3Uri(filePath)

	svc := s3.New(sess)
	_, err = svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(path)})

	if err != nil {
		handler.Errorf("File %s failed to clean up properly, error occurred while deleting object: %v\n", filePath, err)
		return err
	}

	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	})

	if err != nil {
		handler.Errorf("File %s failed to clean up properly, error occurred while waiting for object to be deleted: %v\n", filePath, err)
		return err
	}

	return nil
}
