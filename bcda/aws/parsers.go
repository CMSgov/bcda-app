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
