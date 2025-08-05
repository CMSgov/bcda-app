package slack_utils

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const (
	OperationsChannel = "C0992DK6Y01" // #bcda-operations
	AlertsChannel     = "C034CFU945C" // #bcda-alerts
	SuccessMsg        = "SUCCESS"
	FailureMsg        = "FAILURE"
)

func SendSlackMessage(sc *slack.Client, channel string, msg string, status bool) {
	color := "danger"
	if status {
		color = "good"
	}

	a := slack.Attachment{
		Color: color,
		Text:  msg,
	}
	_, _, err := sc.PostMessageContext(context.Background(), channel, slack.MsgOptionAttachments(a))
	if err != nil {
		log.Errorf("Failed to send slack message: %+v", err)
	}
}
