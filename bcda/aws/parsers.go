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

		s3Records, err := extractS3Records(sqsRecord.Body)
		if err != nil {
			log.Warnf("Skipping SQS record %s: %v", sqsRecord.MessageId, err)
			continue
		}
		allRecords = append(allRecords, s3Records...)
	}

	return &events.S3Event{Records: allRecords}, nil
}

func extractS3Records(body string) ([]events.S3EventRecord, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse SQS body as JSON: %w", err)
	}

	if msgType, ok := raw["Type"]; ok {
		var t string
		if err := json.Unmarshal(msgType, &t); err == nil && t == "Notification" {
			return extractFromSNS(raw)
		}
	}

	if _, ok := raw["Records"]; ok {
		return extractFromS3Body(body)
	}

	return nil, fmt.Errorf("unrecognized SQS body shape (neither SNS notification nor S3 event)")
}

func extractFromSNS(raw map[string]json.RawMessage) ([]events.S3EventRecord, error) {
	msgRaw, ok := raw["Message"]
	if !ok {
		return nil, fmt.Errorf("SNS notification missing 'Message' field")
	}

	var msg string
	if err := json.Unmarshal(msgRaw, &msg); err != nil {
		return nil, fmt.Errorf("failed to parse SNS Message field: %w", err)
	}
	if msg == "" {
		return nil, fmt.Errorf("SNS Message field is empty")
	}

	return extractFromS3Body(msg)
}

func extractFromS3Body(body string) ([]events.S3EventRecord, error) {
	var s3Event events.S3Event
	if err := json.Unmarshal([]byte(body), &s3Event); err != nil {
		return nil, fmt.Errorf("failed to parse S3 event: %w", err)
	}
	if len(s3Event.Records) == 0 {
		return nil, fmt.Errorf("S3 event contained no records")
	}

	var s3Records []events.S3EventRecord
	for _, r := range s3Event.Records {
		if r.EventSource != "aws:s3" {
			log.Warnf("Skipping non-S3 record with eventSource %q", r.EventSource)
			continue
		}
		s3Records = append(s3Records, r)
	}

	if len(s3Records) == 0 {
		return nil, fmt.Errorf("no valid S3 records found after filtering")
	}

	return s3Records, nil
}

func ParseS3Directory(bucket, key string) string {
	lastSeparatorIdx := strings.LastIndex(key, "/")

	if lastSeparatorIdx == -1 {
		return bucket
	} else {
		return fmt.Sprintf("%s/%s", bucket, key[:lastSeparatorIdx])
	}
}
