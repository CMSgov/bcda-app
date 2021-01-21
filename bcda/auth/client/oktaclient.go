package client

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

    configuration "github.com/CMSgov/bcda-app/config"

	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

var oktaBaseUrl string
var oktaAuthString string
var oktaServerID string

var publicKeys map[string]rsa.PublicKey
var once sync.Once

type OktaToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

type Credentials struct {
	ClientID     string
	ClientSecret string
	ClientName   string
}

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
	oktaBaseUrl = configuration.GetEnv("OKTA_CLIENT_ORGURL")
	oktaServerID = configuration.GetEnv("OKTA_OAUTH_SERVER_ID")
	oktaToken := configuration.GetEnv("OKTA_CLIENT_TOKEN")

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
		logEmergency(err).Print("No public keys available for server")
		// our practice is to not stop the app, even when it's in a state where it can do nothing but emit errors
		// methods called on this ob value will result in errors until the publicKeys map is successfully updated
	}
	return &OktaClient{}
}

func (oc *OktaClient) ServerID() string {
	return oktaServerID
}

func (oc *OktaClient) PublicKeyFor(id string) (rsa.PublicKey, bool) {
	key, ok := publicKeys[id]
	if !ok {
		logger.WithFields(logrus.Fields{"signing_key_id": id}).Warn("invalid signing key id presented")
	}
	return key, ok
}

func (oc *OktaClient) AddClientApplication(localID string) (clientID string, clientSecret string, clientName string, err error) {
	requestID := uuid.NewRandom()
	clientName = fmt.Sprintf("BCDA %s", localID)

	body := fmt.Sprintf(`{ "client_name": "%s", "client_uri": null, "logo_uri": null, "application_type": "service", "redirect_uris": [], "response_types": [ "token" ], "grant_types": [ "client_credentials" ], "token_endpoint_auth_method": "client_secret_basic" }`, clientName)
	req, err := http.NewRequest("POST", oktaBaseUrl+"/oauth2/v1/clients", bytes.NewBufferString(body))
	if err != nil {
		return
	}

	addRequestHeaders(req)

	logRequest(requestID).WithField("client_name", clientName).Print("creating client in okta")

	resp, err := client().Do(req)

	if err != nil {
		return
	}

	logResponse(resp.StatusCode, requestID).Print()

	if resp.StatusCode != 201 {
		var body []byte
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			logError(err, requestID).WithField("local_id", localID).Print()
			return
		}
		err = fmt.Errorf("unexpected result: %s", body)
		logError(err, requestID).Print()
		return
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return
	}

	clientID = result["client_id"].(string)
	clientSecret = result["client_secret"].(string)

	err = addClientToPolicy(clientID, requestID)
	if err != nil {
		logError(err, requestID).WithField("local_id", localID).Info("client can't access server")
		return
		// client will not be able to use server until it is added to the policy
	}

	return
}

func (oc *OktaClient) RequestAccessToken(creds Credentials) (OktaToken, error) {
	requestID := uuid.NewRandom()

	tokenURL := fmt.Sprintf("%s/oauth2/%s/v1/token", oktaBaseUrl, oktaServerID)
	req, err := http.NewRequest("POST", tokenURL, nil)
	if err != nil {
		return OktaToken{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Cache-Control", "no-cache")

	params := url.Values{}
	params.Set("client_id", creds.ClientID)
	params.Set("client_secret", creds.ClientSecret)
	params.Set("grant_type", "client_credentials")
	params.Set("scope", "bcda_api")
	req.URL.RawQuery = params.Encode()

	logRequest(requestID).Print("requesting Okta access token")
	resp, err := client().Do(req)
	if err != nil {
		return OktaToken{}, err
	}

	defer resp.Body.Close()
	logResponse(resp.StatusCode, requestID).Print()

	if resp.StatusCode >= 400 {
		err = errors.New(resp.Status)
		logError(err, requestID).WithField("client_id", creds.ClientID).Info("unable to get access token")
		return OktaToken{}, err
	}

	var ot = OktaToken{}

	if err = json.NewDecoder(resp.Body).Decode(&ot); err != nil {
		message := "unexpected token response format from Okta"
		logError(err, requestID).WithField("client_id", creds.ClientID).Info(message)
		return OktaToken{}, errors.New(message)
	}

	return ot, nil
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

	addRequestHeaders(req)

	// not calling logRequest() because this is a step of AddClientApplication

	resp, err := client().Do(req)

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

	addRequestHeaders(req)

	// not calling logRequest() because this is a step of AddClientApplication

	resp, err = client().Do(req)

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

func (oc *OktaClient) GenerateNewClientSecret(clientID string) (string, error) {
	url := oktaBaseUrl + "/oauth2/v1/clients/" + clientID + "/lifecycle/newSecret"

	reqID := uuid.NewRandom()
	logRequest(reqID).WithFields(logrus.Fields{"url": url, "clientID": clientID}).Print()
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logError(err, reqID)
		return "", err
	}

	addRequestHeaders(req)

	resp, err := client().Do(req)
	if err != nil {
		logError(err, reqID).Print()
		return "", err
	}
	if resp.StatusCode >= 400 {
		logError(errors.New(resp.Status), reqID).Print()
		return "", errors.New(resp.Status)
	}

	logResponse(resp.StatusCode, reqID).Print()
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	cs := result["client_secret"].(string)

	return cs, nil
}

func (oc *OktaClient) DeactivateApplication(clientID string) error {
	url := oktaBaseUrl + "/api/v1/apps/" + clientID + "/lifecycle/deactivate"

	reqID := uuid.NewRandom()
	logRequest(reqID).WithFields(logrus.Fields{"url": url, "clientID": clientID}).Print()
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logError(err, reqID)
		return err
	}

	addRequestHeaders(req)

	resp, err := client().Do(req)
	if err != nil {
		logError(err, reqID).Print()
		return err
	}
	if resp.StatusCode >= 400 {
		logError(errors.New(resp.Status), reqID).Print()
		return errors.New(resp.Status)
	}
	logResponse(resp.StatusCode, reqID).Print()

	return nil
}

func (oc *OktaClient) RemoveClientApplication(clientID string) error {
	url := oktaBaseUrl + "/oauth2/v1/clients/" + clientID

	reqID := uuid.NewRandom()
	logRequest(reqID).WithFields(logrus.Fields{"url": url, "clientID": clientID}).Print()
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		logError(err, reqID)
		return err
	}

	addRequestHeaders(req)

	resp, err := client().Do(req)
	if err != nil {
		logError(err, reqID).Print()
		return err
	}
	if resp.StatusCode >= 400 {
		logError(errors.New(resp.Status), reqID).Print()
		return errors.New(resp.Status)
	}
	logResponse(resp.StatusCode, reqID).Print()

	return nil
}

func client() *http.Client {
	return &http.Client{Timeout: time.Second * 10}
}

func addRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", oktaAuthString)
}

func logRequest(requestID uuid.UUID) *logrus.Entry {
	return logger.WithField("request_id", requestID)
}

func logResponse(httpStatus int, requestID uuid.UUID) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"http_status": httpStatus, "request_id": requestID})
}

func logError(err error, requestID uuid.UUID) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"error": err, "request_id": requestID})
}

func logEmergency(err error) *logrus.Entry {
	return logger.WithFields(logrus.Fields{"error": err, "emergency": "invalid system state"})
}
