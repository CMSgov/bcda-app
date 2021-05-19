package insights

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/service/firehose/firehoseiface"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
)

var instance *firehose.Firehose
var once sync.Once

func GetFirehose() *firehose.Firehose {
	once.Do(func() {
		sess := session.Must(session.NewSession())
		instance = firehose.New(sess, aws.NewConfig().WithRegion("us-east-1"))
	})
	return instance
}

type JsonResult struct {
	EventMsg string `json:"event"`
}

type Event struct {
	Name      string     `json:"name"`
	Timestamp time.Time  `json:"timestamp"`
	Result    JsonResult `json:"json_result"`
}

func PutEvent(svc firehoseiface.FirehoseAPI, name string, event string) {

	if utils.GetEnvBool("BCDA_ENABLE_INSIGHTS_EVENTS", true) {

		targetEnv := conf.GetEnv("DEPLOYMENT_TARGET")
		streamName := "bfd-insights-bcda-" + targetEnv + "-event_processor"

		recordInput := &firehose.PutRecordInput{}
		recordInput = recordInput.SetDeliveryStreamName(streamName)

		eventMsg := JsonResult{
			EventMsg: event,
		}

		data := Event{
			Name:      name,
			Timestamp: time.Now(),
			Result:    eventMsg,
		}

		b, err := json.Marshal(data)

		if err != nil {
			log.API.Error(err)
		}

		record := &firehose.Record{Data: b}
		recordInput = recordInput.SetRecord(record)

		_, err = svc.PutRecord(recordInput)

		if err != nil {
			log.API.Error(err)
		}
	} else {
		log.API.Info("Insights is not enabled for the application.  No data was sent to BFD.")
	}
}
