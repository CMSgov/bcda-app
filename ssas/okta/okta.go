package okta

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"net"
	"net/http"
	"os"
	"time"
)

var OktaBaseUrl string
var OktaAuthString string
var OktaCACertFingerprint []byte

type OktaError struct {
	ErrorCode 		string	`json:"errorCode"`
	ErrorSummary	string	`json:"errorSummary"`
}

type Dialer func(network, addr string) (net.Conn, error)

func init() {
	err := config()
	if err != nil {
		initEvent := ssas.Event{Op: "OktaInitialization", Help: "unable to complete Okta config: " + err.Error()}
		ssas.OperationFailed(initEvent)
	}
}

// separate from init for testing
func config() error {
	oktaToken := os.Getenv("OKTA_CLIENT_TOKEN")
	at := oktaToken
	if at != "" {
		at = "[Redacted]"
	}
	OktaAuthString = fmt.Sprintf("SSWS %s", oktaToken)
	OktaBaseUrl = os.Getenv("OKTA_CLIENT_ORGURL")
	fingerprintString := os.Getenv("OKTA_CA_CERT_FINGERPRINT")

	if OktaBaseUrl == "" || oktaToken == "" || fingerprintString == "" {
		return fmt.Errorf(fmt.Sprintf("missing env vars: OKTA_CLIENT_ORGURL=%s, OKTA_CA_CERT_FINGERPRINT=%s, OKTA_CLIENT_TOKEN=%s",
			OktaBaseUrl, fingerprintString, at))
	}

	var err error
	OktaCACertFingerprint, err = hex.DecodeString(fingerprintString)
	if err != nil {
		return fmt.Errorf("unable to parse OKTA_CA_CERT_FINGERPRINT: " + err.Error())
	}

	return nil
}

/*
	Client returns an http.Client set with appropriate defaults, including an extra layer of certificate validation
 */
func Client() *http.Client {
	client := http.Client{Timeout: time.Second * 10}
	client.Transport = &http.Transport{
		DialTLS: makeDialer(OktaCACertFingerprint, false),
	}
	return &client
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

// Modified from https://medium.com/@zmanian/server-public-key-pinning-in-go-7a57bbe39438
func makeDialer(fingerprint []byte, skipCAVerification bool) Dialer {
	return func(network, addr string) (net.Conn, error) {
		var errMessage string
		c, err := tls.Dial(network, addr, &tls.Config{InsecureSkipVerify: skipCAVerification})
		if err != nil {
			return c, err
		}
		connstate := c.ConnectionState()
		keyPinValid := false
		for _, peercert := range connstate.PeerCertificates {
			hash := sha1.Sum(peercert.Raw)

			// We're not pinning the certificate itself, just the CA that issued it
			if peercert.IsCA {
				if bytes.Compare(hash[0:], fingerprint) != 0 {
					errMessage = fmt.Sprintf("pinned CA key changed; issuer of presented key: %s, DNSNames: %s, IsCA: %t, Subject: %s, fingerprint: %#v, stored fingerprint: %#v",
						peercert.Issuer, peercert.DNSNames, peercert.IsCA, peercert.Subject, hash, OktaCACertFingerprint)
				} else {
					keyPinValid = true
				}
			}
		}
		if keyPinValid == false {
			return nil, fmt.Errorf(errMessage)
		}
		return c, nil
	}
}
