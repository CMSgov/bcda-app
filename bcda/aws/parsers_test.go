package bcdaaws

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSQSEvent(t *testing.T) {
	jsonFile, err := os.Open("../../shared_files/aws/s3event.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Println(err)
	}

	var s3event events.S3Event
	err = json.Unmarshal([]byte(byteValue), &s3event)
	if err != nil {
		fmt.Println(err)
	}

	val, err := json.Marshal(s3event)

	if err != nil {
		fmt.Println(err)
	}

	body := fmt.Sprintf("{\"Type\" : \"Notification\",\n  \"MessageId\" : \"123456-1234-1234-1234-6e06896db643\",\n  \"TopicArn\" : \"my-topic\",\n  \"Subject\" : \"Amazon S3 Notification\",\n  \"Message\" : %s}", strconv.Quote(string(val[:])))

	event := events.SQSEvent{
		Records: []events.SQSMessage{{Body: body}},
	}

	s3Event, err := ParseSQSEvent(event)
	assert.Nil(t, err)
	assert.NotNil(t, s3Event)
	assert.Equal(t, "demo-bucket", s3Event.Records[0].S3.Bucket.Name)
}

const validSQSBody = `{"Records":[{"eventVersion":"2.1","eventSource":"aws:s3","awsRegion":"us-east-1","eventTime":"2026-04-06T15:53:08.464Z","eventName":"ObjectCreated:Put","userIdentity":{"principalId":"AWS:AROAX3DNHMVW75XU7BKY5:GitHubActions"},"requestParameters":{"sourceIPAddress":"10.128.178.117"},"responseElements":{"x-amz-request-id":"YFYF9S4FBZFRN4AA","x-amz-id-2":"Pj/e0FHrLRIJRsxpNJhCaNEofrZ3K1LgHXRrfNneoce4H6hTewDfzpzGy1hPehawi4rLXSg9cr3yx4k6KDrcysG2hRDth6en"},"s3":{"s3SchemaVersion":"1.0","configurationId":"tf-s3-queue-20260403213957476200000007","bucket":{"name":"bcda-test-attribution-import-file-20260403213903766100000002","ownerIdentity":{"principalId":"A2UKBQ39A9SMAG"},"arn":"arn:aws:s3:::bcda-test-attribution-import-file-20260403213903766100000002"},"object":{"key":"bfdeft01/bcda/in/test/T.BCD.A0001.ZCY26.D260406.T1552571","size":1552,"eTag":"4d1d190e07ecceab7339874a3a333e5b","versionId":"l3.JslQw24diHQP7VIOSx4zcAugyv251","sequencer":"0069D3D6E45CCAE058"}}}]}`

func makeSQSRecord(messageID, body string) events.SQSMessage {
	return events.SQSMessage{
		MessageId:     messageID,
		ReceiptHandle: "AQEBE6YYP2TWOrYuUXMFJ/YaKZUx5+XjF/YmgZMoEWgzcvxFpo3d2zibARSwxps2yHYoH3Jo/pBD5BO4CwVTdqA2hicSlpCDoh7jMEmC5nIKvS9okiL8D3QE0YviWXekPCoAlB7bx7y07hF2G0zU0FbRt1IX9LBEJ1YuaqhJqZFAa/W33Pd6ggO9XVFR74nyZlcdyrkt+bldT4IXaUEqj8pxG7J2e/JFpFV2mhML92QHfbycOf7bXhIrhJcteApc0wnaHYqGJeXkVQV1yknPCkTY9wTS3cva39ozgkWmC+6bnJyGWSRDdY+bfLb5PFtLsy2TGZD6J08Divkrz6WRm+IxdPJA4N/l5byluxCGGraN5qNEaCVIonXTn8cvJFxXQv74qE/kHdEuqq0DoPpiGUibSZT9Ugtt1baDMugvoWxKeMI=",
		Body:          body,
		Md5OfBody:     "adeb1ed2f39eaad9335be62388d46114",
		Attributes: map[string]string{
			"ApproximateFirstReceiveTimestamp": "1775490789142",
			"ApproximateReceiveCount":          "1",
			"SenderId":                         "AROA4R74ZO52XAB5OD7T4:S3-PROD-END",
			"SentTimestamp":                    "1775490789134",
		},
		EventSourceARN: "arn:aws:sqs:us-east-1:539247469933:bcda-test-attribution-import",
		EventSource:    "aws:sqs",
		AWSRegion:      "us-east-1",
	}
}

func TestParseSQSEventFromS3(t *testing.T) {
	t.Run("valid single SQS record returns parsed S3 event", func(t *testing.T) {
		event := events.SQSEvent{
			Records: []events.SQSMessage{
				makeSQSRecord("9deef818-dd81-4808-a9c7-ae5981e8c92c", validSQSBody),
			},
		}

		result, err := ParseSQSEventFromS3(event)

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result.Records, 1)

		rec := result.Records[0]
		assert.Equal(t, "2.1", rec.EventVersion)
		assert.Equal(t, "aws:s3", rec.EventSource)
		assert.Equal(t, "us-east-1", rec.AWSRegion)
		assert.Equal(t, "ObjectCreated:Put", rec.EventName)
	})
}

func TestParseS3Directory(t *testing.T) {
	assert.Equal(t, "my-bucket", ParseS3Directory("my-bucket", "some-file"))
	assert.Equal(t, "my-bucket/my-dir", ParseS3Directory("my-bucket", "my-dir/some-file"))
	assert.Equal(t, "my-bucket/my-dir/nested", ParseS3Directory("my-bucket", "my-dir/nested/some-file"))
}
