package client

import (
	"crypto/rsa"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pborman/uuid"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

var oktaBaseUrl string
var oktaAuthString string
var oktaServerID string

var publicKeys map[string]rsa.PublicKey
var once sync.Once

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
	config()
}

// separate from init for testing
func config() error {
	// required env vars
	oktaBaseUrl = os.Getenv("OKTA_CLIENT_ORGURL")
	oktaServerID = os.Getenv("OKTA_OAUTH_SERVER_ID")
	oktaToken := os.Getenv("OKTA_CLIENT_TOKEN")

	// report missing env vars
	at := oktaToken
	if at != "" {
		at =  "[Redacted]"
	}

	if oktaBaseUrl == "" || oktaServerID == "" || oktaToken == "" {
		return fmt.Errorf(fmt.Sprintf("missing env vars: OKTA_CLIENT_ORGURL=%s, OKTA_OAUTH_SERVER_ID=%s, OKTA_CLIENT_TOKEN=%s", oktaBaseUrl, oktaServerID, at))
	}

	// manufactured from env var
	oktaAuthString = fmt.Sprintf("SSWS %s", oktaToken)

	return nil
}

type OktaClient struct {}

// Returns an OktaClient. An OktaClient is always created, whether or not it is currently able to converse with Okta.
func NewOktaClient() *OktaClient {
	once.Do(func() {
		publicKeys = getPublicKeys()
		go refreshKeys()
	})

	return &OktaClient{}
}

func (oc *OktaClient) PublicKeyFor(id string) (rsa.PublicKey, bool) {
	key, ok := publicKeys[id]
	logger.Warnf("invalid key id %s presented", id)
	return key, ok
}


func refreshKeys() {
	for range time.Tick(time.Hour * 1) {
		logger.Info("refreshing okta public keys")
		publicKeys = getPublicKeys()
	}
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
