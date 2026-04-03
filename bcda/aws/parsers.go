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

func ParseSQSEventFromS3(event events.SQSEvent) (*events.S3Event, error) {
    var allRecords []events.S3EventRecord

    for _, sqsRecord := range event.Records {
        if len(sqsRecord.Body) == 0 {
            log.Warn("Skipping empty SQS record body")
            continue
        }

        var snsEntity events.SNSEntity
        if err := json.Unmarshal([]byte(sqsRecord.Body), &snsEntity); err != nil {
            var unmarshalTypeErr *json.UnmarshalTypeError
            if errors.As(err, &unmarshalTypeErr) {
                log.Warn("Skipping record due to unrecognized format for SNS")
                continue
            }
            return nil, fmt.Errorf("failed to parse SNS entity: %w", err)
        }

        if snsEntity.Message == "" {
            log.Warn("Skipping SNS entity with empty message")
            continue
        }

        var s3Event events.S3Event
        if err := json.Unmarshal([]byte(snsEntity.Message), &s3Event); err != nil {
            var unmarshalTypeErr *json.UnmarshalTypeError
            if errors.As(err, &unmarshalTypeErr) {
                log.Warn("Skipping record due to unrecognized format for S3")
                continue
            }
            return nil, fmt.Errorf("failed to parse S3 event: %w", err)
        }

        allRecords = append(allRecords, s3Event.Records...)
    }

    return &events.S3Event{Records: allRecords}, nil
}

func ParseS3Directory(bucket, key string) string {
	lastSeparatorIdx := strings.LastIndex(key, "/")

	if lastSeparatorIdx == -1 {
		return bucket
	} else {
		return fmt.Sprintf("%s/%s", bucket, key[:lastSeparatorIdx])
	}
}
