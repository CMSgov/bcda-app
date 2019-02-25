package client

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
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
	// there is no possibility of recovery if we put the call to config() here
	// because init() is called once when main is being started
}

// separate from init for testing
func config() error {
	oktaBaseUrl = os.Getenv("OKTA_CLIENT_ORGURL")
	oktaServerID = os.Getenv("OKTA_OAUTH_SERVER_ID")
	oktaToken := os.Getenv("OKTA_CLIENT_TOKEN")

	at := oktaToken
	if at != "" {
		at = "[Redacted]"
	}

	if oktaBaseUrl == "" || oktaServerID == "" || oktaToken == "" {
		return fmt.Errorf(fmt.Sprintf("missing env vars: OKTA_CLIENT_ORGURL=%s, OKTA_OAUTH_SERVER_ID=%s, OKTA_CLIENT_TOKEN=%s", oktaBaseUrl, oktaServerID, at))
	}

	oktaAuthString = fmt.Sprintf("SSWS %s", oktaToken)

	return nil
}

type OktaClient struct{}

// Returns an OktaClient. An OktaClient is always created, whether or not it is currently able to converse with Okta.
func NewOktaClient() *OktaClient {
	var err error
	once.Do(func() {
		err = config()
		if err == nil {
			publicKeys = getPublicKeys()
		}
		// called even if there's been an error so we might recover
		go refreshKeys()
	})
	if err != nil {
		logEmergency(err, nil).Print("No public keys available for server")
		// our practice is to not stop the app, even when it's in a state where it can do nothing but emit errors
		// methods called on this ob value will result in errors until the publicKeys map is successfully updated
	}
	return &OktaClient{}
}

func (oc *OktaClient) PublicKeyFor(id string) (rsa.PublicKey, bool) {
	key, ok := publicKeys[id]
	if !ok {
		logger.WithFields(logrus.Fields{"signing_key_id": id}).Warn("invalid signing key id presented")
	}
	return key, ok
}

func (oc *OktaClient) AddClientApplication(localID string) (string, string, error) {
	requestID := uuid.NewRandom()

	body := fmt.Sprintf(`{ "client_name": "BCDA %s", "client_uri": null, "logo_uri": null, "application_type": "service", "redirect_uris": [], "response_types": [ "token" ], "grant_types": [ "client_credentials" ], "token_endpoint_auth_method": "client_secret_basic" }`, localID)
	req, err := http.NewRequest("POST", oktaBaseUrl+"/oauth2/v1/clients", bytes.NewBufferString(body))
	if err != nil {
		return "", "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", oktaAuthString)

	logRequest(requestID).Print("creating client in okta")

	var client = &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}

	logResponse(resp.StatusCode, requestID).Print()

	if resp.StatusCode != 201 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logError(err, requestID).WithField("local_id", localID).Print()
			return "", "", err
		}
		err = fmt.Errorf("unexpected result: %s", body)
		logError(err, requestID).Print()
		return "", "", err
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", "", err
	}

	clientID := result["client_id"].(string)
	clientSecret := result["client_secret"].(string)

	err = addClientToPolicy(clientID, requestID)
	if err != nil {
		logError(err, requestID).WithField("local_id", localID).Info("client can't access server")
		return "", "", err
		// client will not be able to use server until it is added to the policy
	}

	return clientID, clientSecret, nil
}

type Policy struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	System      bool   `json:"system"`
	Conditions  Cond   `json:"conditions"`
}

type Cond struct {
	Clients Cli `json:"clients"`
}

type Cli struct {
	Include []string `json:"include"`
}

// Update the Auth Server's access policy to include our new client application. Otherwise, that application
// will not be able to use the server. To do this, we first get the current list of clients, add the new
// server to the inclusion list, and put it back to the server
func addClientToPolicy(clientID string, requestID uuid.UUID) error {
	policyUrl := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies", oktaBaseUrl, oktaServerID)

	req, err := http.NewRequest("GET", policyUrl, nil)
	if err != nil {
		logError(err, requestID).Print()
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", oktaAuthString)

	// not calling logRequest() because this is a step of AddClientApplication

	var client = &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(req)
	if err != nil {
		logError(err, requestID).Print()
		return err
	}

	var result []Policy
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		logError(err, requestID).Print()
		return err
	}

	if len(result) > 1 {
		logError(err, requestID).Print("more than one policy entry for server; can't continue safely")
		return err
	}

	incl := result[0].Conditions.Clients.Include
	incl = append(incl, clientID)
	result[0].Conditions.Clients.Include = incl

	body, err := json.Marshal(result[0])
	if err != nil {
		logError(err, requestID).Print()
		return err
	}

	req, err = http.NewRequest("PUT", fmt.Sprintf("%s/%s", policyUrl, result[0].ID), bytes.NewBuffer(body))
	if err != nil {
		logError(err, requestID).Print()
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", oktaAuthString)

	// not calling logRequest() because this is a step of AddClientApplication

	client = &http.Client{Timeout: time.Second * 10}
	resp, err = client.Do(req)
	if err != nil {
		logError(err, requestID).WithField("policy_id", result[0].ID).Print()
		return err
	}

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logError(err, requestID).WithFields(logrus.Fields{"client_id": clientID, "http_status": resp.StatusCode}).Info("failed to update policy")
			return err
		}
		err = fmt.Errorf("unexpected result: %s", body)
		logError(err, requestID).WithFields(logrus.Fields{"client_id": clientID, "http_status": resp.StatusCode}).Print()
		return err
	}

	return nil
}

func refreshKeys() {
	for range time.Tick(time.Hour * 1) {
		logger.Info("Refreshing okta public keys")
		publicKeys = getPublicKeys()
	}
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

func logEmergency(err error, requestId uuid.UUID) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"error": err, "emergency": "invalid system state"})
}
