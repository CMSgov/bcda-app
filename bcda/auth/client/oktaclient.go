package client

import (
	"net/http"
	"os"

	"github.com/pborman/uuid"

	"github.com/okta/okta-sdk-golang/okta"
	"github.com/okta/okta-sdk-golang/okta/query"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	filePath, success := os.LookupEnv("BCDA_OKTA_LOG")
	if success {
		/* #nosec -- 0640 permissions required for Splunk ingestion */
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

		if err == nil {
			logger.SetOutput(file)
		} else {
			logger.Info("Failed to open Okta log file; using default stderr")
		}
	} else {
		logger.Info("No Okta log location provided; using default stderr")
	}

	oktaToken := os.Getenv("OKTA_CLIENT_TOKEN")
	if oktaToken == "" {
		logger.Fatal("No Okta token found; please set OKTA_CLIENT_TOKEN")
	}

	oktaURL := os.Getenv("OKTA_CLIENT_ORGURL")
	if oktaURL == "" {
		logger.Fatal("No Okta URL found; please set OKTA_CLIENT_ORGURL")
	}
}

func HealthCheck() (bool, error) {
	reqId := uuid.NewRandom()
	client := NewOktaClient()
	oktaTestUser := "shawn@bcda.aco-group.us"

	l := reqLog(reqId)
	l.Info("Okta ping request")
	users, resp, err := client.User.ListUsers(query.NewQueryParams(query.WithQ(oktaTestUser)))
	if err != nil {
		respErrLog(err, reqId).Error("Okta ping request error")
		return false, err
	}

	if len(users) >= 1 {
		respLog(resp.StatusCode, reqId).Info("Okta ping request successful")
		return true, err
	}

	respLog(resp.StatusCode, reqId).Info("Okta ping request unsuccessful")
	return false, nil
}

func DeleteUser(userId string) (bool, error) {
	reqId := uuid.NewRandom()
	client := NewOktaClient()
	userField := log.Fields{"user_id": userId}

	reqLog(reqId).WithFields(userField).Info("Okta delete user request")
	resp, err := client.User.DeactivateUser(userId, nil)
	if err != nil {
		respErrLog(err, reqId).WithFields(userField).Error("Okta delete user error")
		return false, err
	}

	respLog(resp.StatusCode, reqId).WithFields(userField).Info("Okta delete user success")
	return true, err
}

func FindUser(email string) (string, error) {
	reqId := uuid.NewRandom()
	client := NewOktaClient()

	reqLog(reqId).Info("Okta find request")
	filter := query.NewQueryParams(query.WithQ(email))
	users, resp, err := client.User.ListUsers(filter)
	if err != nil {
		respErrLog(err, reqId).Error("Okta find error")
		return "", err
	}

	respLog := respLog(resp.StatusCode, reqId)
	switch len(users) {
	case 0:
		respLog.Info("Okta user find request unsuccessful")
		return "", err
	case 1:
		userId := users[0].Id
		userField := log.Fields{"user_id": userId}
		respLog.WithFields(userField).Info("Okta user find request successful")
		return userId, err
	default:
		respLog.Info("Okta user find request returned more than one user")
		return "", err
	}
}

func NewOktaClient() *okta.Client {
	// Reads OKTA_CLIENT_TOKEN and OKTA_CLIENT_ORGURL for configuration
	config := okta.NewConfig()
	httpClient := &http.Client{}
	client := okta.NewClient(config, httpClient, nil)
	return client
}

func reqLog(requestId uuid.UUID) *log.Entry {
	return logger.WithField("request_id", requestId)
}

func respLog(httpStatus int, requestId uuid.UUID) *log.Entry {
	return logger.WithFields(log.Fields{"http_status": httpStatus, "request_id": requestId})
}

func respErrLog(err error, requestId uuid.UUID) *log.Entry {
	return logger.WithFields(log.Fields{"error": err, "request_id": requestId})
}
