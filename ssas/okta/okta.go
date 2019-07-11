package okta

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"net/http"
	"os"
	"time"
)

var OktaBaseUrl string
var OktaAuthString string
var OktaServerID string

type OktaError struct {
	ErrorCode 		string	`json:"errorCode"`
	ErrorSummary	string	`json:"errorSummary"`
}

func init() {
	err := config()
	if err != nil {
		haltEvent := ssas.Event{Op: "OktaInitialization", Help: "unable to complete Okta config: " + err.Error()}
		ssas.ServiceHalted(haltEvent)
		os.Exit(-1)
	}
}

// separate from init for testing
func config() error {
	OktaBaseUrl = os.Getenv("OKTA_CLIENT_ORGURL")
	OktaServerID = os.Getenv("OKTA_OAUTH_SERVER_ID")
	oktaToken := os.Getenv("OKTA_CLIENT_TOKEN")

	at := oktaToken
	if at != "" {
		at = "[Redacted]"
	}

	if OktaBaseUrl == "" || OktaServerID == "" || oktaToken == "" {
		return fmt.Errorf(fmt.Sprintf("missing env vars: OKTA_CLIENT_ORGURL=%s, OKTA_OAUTH_SERVER_ID=%s, OKTA_CLIENT_TOKEN=%s", OktaBaseUrl, OktaServerID, at))
	}

	OktaAuthString = fmt.Sprintf("SSWS %s", oktaToken)

	return nil
}

/*
	Client returns an http.Client set with appropriate defaults
 */
func Client() *http.Client {
	return &http.Client{Timeout: time.Second * 10}
}

/*
	AddRequestHeaders sets common headers needed for all Okta requests
 */
func AddRequestHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", OktaAuthString)
}

func ParseOktaError(body []byte) (OktaError, error) {
	oktaError := OktaError{}
	if err := json.Unmarshal(body, &oktaError); err != nil {
		return oktaError, errors.New("unexpected response format; not a standard Okta error")
	}
	return oktaError, nil
}

type RoundTripFunc func(req *http.Request) *http.Response

/*
	RoundTrip allows control of an http.Client's response for testing purposes.  This code is taken
	from https://hassansin.github.io/Unit-Testing-http-client-in-Go
 */
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

/*
	NewTestClient returns *http.Client with Transport replaced to avoid making real calls
 */
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}
