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

func TestParseS3Directory(t *testing.T) {
	assert.Equal(t, "my-bucket", ParseS3Directory("my-bucket", "some-file"))
	assert.Equal(t, "my-bucket/my-dir", ParseS3Directory("my-bucket", "my-dir/some-file"))
	assert.Equal(t, "my-bucket/my-dir/nested", ParseS3Directory("my-bucket", "my-dir/nested/some-file"))
}
