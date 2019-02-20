package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/okta/okta-sdk-golang/okta"
	"github.com/okta/okta-sdk-golang/okta/query"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
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

func HealthCheck() error {
	reqId := uuid.NewRandom()
	client := NewOktaClient()

	// The email in OKTA_EMAIL should represent a test user present in the Okta sandbox environment
	oktaTestUser, success := os.LookupEnv("OKTA_EMAIL")
	if !success {
		logRequest(reqId).Warn("Unable to perform Okta HealthCheck.  Please set OKTA_EMAIL to match a test user account.")
		return errors.New("cannot perform Okta health check without OKTA_EMAIL configured ")
	}

	logRequest(reqId).Info("Okta ping request")
	users, resp, err := client.User.ListUsers(query.NewQueryParams(query.WithQ(oktaTestUser)))
	if err != nil {
		logError(err, reqId).Error("Okta ping request error")
		return err
	}

	if len(users) >= 1 {
		logResponse(resp.StatusCode, reqId).Info("Okta ping request successful")
		return nil
	}

	logResponse(resp.StatusCode, reqId).Info("Okta ping request unsuccessful")
	return errors.New("ping unsuccessful ")
}

func DeleteUser(userId string) (bool, error) {
	reqId := uuid.NewRandom()
	client := NewOktaClient()
	userField := logrus.Fields{"user_id": userId}

	logRequest(reqId).WithFields(userField).Info("Okta delete user request")
	resp, err := client.User.DeactivateUser(userId, nil)
	if err != nil {
		logError(err, reqId).WithFields(userField).Error("Okta delete user error")
		return false, err
	}

	logResponse(resp.StatusCode, reqId).WithFields(userField).Info("Okta delete user success")
	return true, err
}

func FindUser(email string) (string, error) {
	reqId := uuid.NewRandom()
	client := NewOktaClient()

	logRequest(reqId).Info("Okta find request")
	filter := query.NewQueryParams(query.WithQ(email))
	users, resp, err := client.User.ListUsers(filter)
	if err != nil {
		logError(err, reqId).Error("Okta find error")
		return "", err
	}

	l := logResponse(resp.StatusCode, reqId)
	switch len(users) {
	case 0:
		l.Info("Okta user find request unsuccessful")
		return "", err
	case 1:
		userId := users[0].Id
		userField := logrus.Fields{"user_id": userId}
		l.WithFields(userField).Info("Okta user find request successful")
		return userId, err
	default:
		l.Info("Okta user find request returned more than one user")
		return "", err
	}
}

func NewOktaClient() *okta.Client {
	// Reads OKTA_CLIENT_TOKEN and OKTA_CLIENT_ORGURL for configuration
	config := okta.NewConfig()
	httpClient := &http.Client{}
	return okta.NewClient(config, httpClient, nil)
}

func GenerateNewClientSecret(clientID string) (string, error) {
	url := os.Getenv("OKTA_CLIENT_ORGURL") + "/oauth2/v1/clients/" + clientID + "/lifecycle/newSecret"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}

	addRequestHeaders(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", errors.New(resp.Status)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	cs := result["client_secret"].(string)

	return cs, nil
}

func addRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "SSWS "+os.Getenv("OKTA_CLIENT_TOKEN"))
}

func logRequest(requestId uuid.UUID) *logrus.Entry {
	return logger.WithField("request_id", requestId)
}

func logResponse(httpStatus int, requestId uuid.UUID) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"http_status": httpStatus, "request_id": requestId})
}

func logError(err error, requestId uuid.UUID) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"error": err, "request_id": requestId})
}
