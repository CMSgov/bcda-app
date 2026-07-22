package bcdaaws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TODO: Iterate over records
func ParseSQSEvent(event events.SQSEvent) (*events.S3Event, error) {
	var snsEntity events.SNSEntity
	err := json.Unmarshal([]byte(event.Records[0].Body), &snsEntity)

	unmarshalTypeErr := new(json.UnmarshalTypeError)
	if errors.As(err, &unmarshalTypeErr) {
		log.Warn("Skipping event due to unrecognized format for SNS")
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var s3Event events.S3Event
	err = json.Unmarshal([]byte(snsEntity.Message), &s3Event)

	unmarshalTypeErr = new(json.UnmarshalTypeError)
	if errors.As(err, &unmarshalTypeErr) {
		log.Warn("Skipping event due to unrecognized format for S3")
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &s3Event, nil
}

func ParseS3Directory(bucket, key string) string {
	lastSeparatorIdx := strings.LastIndex(key, "/")

	if lastSeparatorIdx == -1 {
		return bucket
	} else {
		return fmt.Sprintf("%s/%s", bucket, key[:lastSeparatorIdx])
	}
}

// Parses an S3 URI and returns the bucket and key.
//
// @example:
//
//	input: s3://my-bucket/path/to/file
//	output: "my-bucket", "path/to/file"
//
// @example
//
//	input: s3://my-bucket
//	output: "my-bucket", ""
func ParseS3Uri(str string) (bucket string, key string) {
	workingString := strings.TrimPrefix(str, "s3://")
	resultArr := strings.SplitN(workingString, "/", 2)

	if len(resultArr) == 1 {
		return resultArr[0], ""
	}

	return resultArr[0], resultArr[1]
}
