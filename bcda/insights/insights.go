package insights

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

type Event struct {
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	Result    string    `json:"json_result"`
}

func PutEvent(name string, event string) {

	if utils.GetEnvBool("BCDA_ENABLE_INSIGHTS_EVENTS", true) {

		targetEnv := conf.GetEnv("DEPLOYMENT_TARGET")
		streamName := "bfd-insights-bcda-" + targetEnv + "-" + name

		sess := session.Must(session.NewSession())
		firehoseService := firehose.New(sess, aws.NewConfig().WithRegion("us-east-1"))

		recordInput := &firehose.PutRecordInput{}
		recordInput = recordInput.SetDeliveryStreamName(streamName)

		data := Event{
			Name:      name,
			Timestamp: time.Now(),
			Result:    event,
		}

		b, err := json.Marshal(data)

		if err != nil {
			log.API.Error(err)
		}

		record := &firehose.Record{Data: b}
		recordInput = recordInput.SetRecord(record)

		_, err := firehoseService.PutRecord(recordInput)

		if err != nil {
			log.API.Error(err)
		}
	}
}
