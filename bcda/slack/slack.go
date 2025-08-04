package slack_utils

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

const (
	OperationsChannel = "C0992DK6Y01" // #bcda-operations
	AlertsChannel     = "C034CFU945C" // #bcda-alerts
	SuccessIcon       = "white_check_mark"
	FailureIcon       = "rotating_light"
	SuccessMsg        = "SUCCESS"
	FailureMsg        = "FAILURE"
)

func SendSlackMessage(sc *slack.Client, channel string, msg string, icon string) {
	_, _, err := sc.PostMessageContext(context.Background(), channel, slack.MsgOptionText(fmt.Sprint(msg), false), slack.MsgOptionIconEmoji(icon))
	if err != nil {
		log.Errorf("Failed to send slack message: %+v", err)
	}
}
