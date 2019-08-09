package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var ssasLogger *logrus.Logger

// SSASClient is a client for interacting with the System-to-System Authentication Service.
type SSASClient struct {
	http.Client
	baseURL string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func init() {
	ssasLogger = logrus.New()
	ssasLogger.Formatter = &logrus.JSONFormatter{}
	filePath := os.Getenv("BCDA_SSAS_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filepath.Clean(filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		ssasLogger.SetOutput(file)
	} else {
		ssasLogger.Info("Failed to open SSAS log file; using default stderr")
	}
}

// NewSSASClient creates and returns an SSASClient.
func NewSSASClient() (*SSASClient, error) {
	var (
		transport = &http.Transport{}
		err       error
	)
	if os.Getenv("SSAS_USE_TLS") == "true" {
		transport, err = tlsTransport()
		if err != nil {
			return nil, errors.Wrap(err, "SSAS client could not be created")
		}
	}

	var timeout int
	if timeout, err = strconv.Atoi(os.Getenv("SSAS_TIMEOUT_MS")); err != nil {
		ssasLogger.Info("Could not get SSAS timeout from environment variable; using default value of 500.")
		timeout = 500
	}

	ssasURL := os.Getenv("SSAS_URL")
	if ssasURL == "" {
		return nil, errors.New("SSAS client could not be created: no URL provided")
	}

	c := http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Millisecond}

	return &SSASClient{c, ssasURL}, nil
}

func tlsTransport() (*http.Transport, error) {
	certFile := os.Getenv("SSAS_CLIENT_CERT_FILE")
	keyFile := os.Getenv("SSAS_CLIENT_KEY_FILE")
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, errors.Wrap(err, "could not load SSAS keypair")
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}

	caFile := os.Getenv("SSAS_CLIENT_CA_FILE")
	caCert, err := ioutil.ReadFile(filepath.Clean(caFile))
	if err != nil {
		return nil, errors.Wrap(err, "could not read CA file")
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, errors.New("could not append CA certificate(s)")
	}

	tlsConfig.RootCAs = caCertPool
	tlsConfig.BuildNameToCertificate()

	return &http.Transport{TLSClientConfig: tlsConfig}, nil
}

// CreateSystem POSTs to the SSAS /system endpoint to create a system.
func (c *SSASClient) CreateSystem() ([]byte, error) {
	return nil, nil
}

// GetPublicKey GETs the SSAS /system/{systemID}/key endpoint to retrieve a system's public key.
func (c *SSASClient) GetPublicKey(systemID int) ([]byte, error) {
	resp, err := c.Get(fmt.Sprintf("%s/system/%v/key", c.baseURL, systemID))
	if err != nil {
		return nil, errors.Wrap(err, "could not get public key")
	}

	defer resp.Body.Close()

	var respMap map[string]string
	if err = json.NewDecoder(resp.Body).Decode(&respMap); err != nil {
		return nil, errors.Wrap(err, "could not get public key")
	}

	return []byte(respMap["public_key"]), nil
}

// ResetCredentials PUTs to the SSAS /system/{systemID}/credentials endpoint to reset the system's credentials.
func (c *SSASClient) ResetCredentials(systemID int) ([]byte, error) {
	return nil, nil
}

// DeleteCredentials DELETEs from the SSAS /system/{systemID}/credentials endpoint to deactivate credentials associated with the system.
func (c *SSASClient) DeleteCredentials(systemID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/system/%s/credentials", c.baseURL, systemID), nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete credentials")
	}

	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to delete credentials")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Wrap(err, "failed to delete credentials")
	}

	return nil
}

// GetToken POSTs to the public SSAS /token endpoint to get an access token for a BCDA client
func (c *SSASClient) GetToken(credentials Credentials) ([]byte, error) {
	public := os.Getenv("SSAS_PUBLIC_URL")
	url := fmt.Sprintf("%s/token", public)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "bad request structure")
	}

	req.SetBasicAuth(credentials.ClientID, credentials.ClientSecret)
	req.Header.Add("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "token request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed; %v", resp.StatusCode)
	}

	var t = TokenResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, errors.Wrap(err, "could not decode token response")
	}

	return []byte(t.AccessToken), nil
}

// VerifyPublicToken verifies that the tokenString presented was issued by the public server. It does so using
// the introspect endpoint as defined by https://tools.ietf.org/html/rfc7662
func (c *SSASClient) VerifyPublicToken(tokenString string) ([]byte, error) {
	public := os.Getenv("SSAS_PUBLIC_URL")
	url := fmt.Sprintf("%s/introspect", public)
	body := strings.NewReader("token="+tokenString)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, errors.Wrap(err, "bad request structure")
	}

	// TODO assuming auth is by self-management
	req.SetBasicAuth(os.Getenv("BCDA_SSAS_CLIENT_ID"), os.Getenv("BCDA_SSAS_SECRET"))
	req.Header.Add("Accept", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "introspect request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspect request failed; %v", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "could not read introspect response")
	}

	return b, nil
}
