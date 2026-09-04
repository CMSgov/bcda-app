package slackmessenger

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const (
	OperationsChannel = "C0992DK6Y01" // #bcda-operations
	AlertsChannel     = "C034CFU945C" // #bcda-alerts
	SuccessMsg        = "SUCCESS"
	FailureMsg        = "FAILURE"
	Danger            = "danger"
	Good              = "good"
)

func SendSlackMessage(sc *slack.Client, channel string, msg string, color string) {
	a := slack.Attachment{
		Color: color,
		Text:  msg,
	}
	_, _, err := sc.PostMessageContext(context.Background(), channel, slack.MsgOptionAttachments(a))
	if err != nil {
		log.Errorf("Failed to send slack message: %+v", err)
	}
}

func SendSuccessToOperations(sc *slack.Client, msg string) {
	formattedMsg := fmt.Sprintf("%s: [%s environment]: %s ", SuccessMsg, os.Getenv("ENV"), msg)
	color := Good
	channel := OperationsChannel
	SendSlackMessage(sc, channel, formattedMsg, color)
}

func SendFailureToAlerts(sc *slack.Client, msg string) {
	formattedMsg := fmt.Sprintf("%s: [%s environment] %s ", FailureMsg, os.Getenv("ENV"), msg)
	color := Danger
	channel := AlertsChannel
	SendSlackMessage(sc, channel, formattedMsg, color)
}
