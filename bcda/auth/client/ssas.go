package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"

	"github.com/pkg/errors"
)

// SSASClient is a client for interacting with the System-to-System Authentication Service.
type SSASClient struct {
	http.Client
	baseURL string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type Credentials struct {
	ClientID     string
	ClientSecret string
	ClientName   string
}

// NewSSASClient creates and returns an SSASClient.
func NewSSASClient() (*SSASClient, error) {
	var (
		transport = &http.Transport{}
		err       error
	)
	if conf.GetEnv("SSAS_USE_TLS") == "true" {
		transport, err = tlsTransport()
		if err != nil {
			return nil, errors.Wrap(err, "SSAS client could not be created")
		}
	}

	var timeout int
	if timeout, err = strconv.Atoi(conf.GetEnv("SSAS_TIMEOUT_MS")); err != nil {
		log.SSAS.Info("Could not get SSAS timeout from environment variable; using default value of 500.")
		timeout = 500
	}

	ssasURL := conf.GetEnv("SSAS_URL")
	if ssasURL == "" {
		return nil, errors.New("SSAS client could not be created: no URL provided")
	}

	c := http.Client{Transport: transport, Timeout: time.Duration(timeout) * time.Millisecond}

	return &SSASClient{c, ssasURL}, nil
}

func tlsTransport() (*http.Transport, error) {
	caFile := conf.GetEnv("BCDA_CA_FILE")
	caCert, err := ioutil.ReadFile(filepath.Clean(caFile))
	if err != nil {
		return nil, errors.Wrap(err, "could not read CA file")
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, errors.New("could not append CA certificate(s)")
	}

	log.SSAS.Println("Using ca cert sourced from ", filepath.Clean(caFile))
	tlsConfig := &tls.Config{RootCAs: caCertPool, MinVersion: tls.VersionTLS12}

	return &http.Transport{TLSClientConfig: tlsConfig}, nil
}

// CreateGroup POSTs to the SSAS /group endpoint to create a system.
func (c *SSASClient) CreateGroup(id, name, acoCMSID string) ([]byte, error) {
	b := fmt.Sprintf(`{"group_id": "%s", "name": "%s", "scopes": ["bcda-api"], "xdata": "{\"cms_ids\": [\"%s\"]}"}`, id, name, acoCMSID)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/group", c.baseURL), strings.NewReader(b))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("invalid input for new group_id %s", id))
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not create group")
	}

	rb, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.Wrap(err, "could not create group")
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.Errorf("could not create group: %s", rb)
	}

	return rb, nil
}

// DeleteGroup DELETEs to the SSAS /group/{id} endpoint to delete a group.
func (c *SSASClient) DeleteGroup(id int) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/group/%d", c.baseURL, id), nil)
	if err != nil {
		return errors.Wrap(err, "could not delete group")
	}
	if err := c.setAuthHeader(req); err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not delete group")
	}

	if resp.StatusCode != http.StatusOK {
		rb, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return errors.Wrap(err, "could not delete group")
		}
		return errors.Errorf("could not delete group: %s", rb)
	}

	return nil
}

// CreateSystem POSTs to the SSAS /system endpoint to create a system.
func (c *SSASClient) CreateSystem(clientName, groupID, scope, publicKey, trackingID string, ips []string) ([]byte, error) {
	type system struct {
		ClientName string   `json:"client_name"`
		GroupID    string   `json:"group_id"`
		Scope      string   `json:"scope"`
		PublicKey  string   `json:"public_key"`
		TrackingID string   `json:"tracking_id"`
		IPs        []string `json:"ips,omitempty"`
	}

	sys := system{
		ClientName: clientName,
		GroupID:    groupID,
		Scope:      scope,
		PublicKey:  publicKey,
		TrackingID: trackingID,
		IPs:        ips,
	}

	bb, err := json.Marshal(sys)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create system")
	}
	br := bytes.NewReader(bb)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/system", c.baseURL), br)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create system")
	}
	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create system")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(fmt.Sprintf("failed to create system. status code: %v", resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return body, nil
}

// GetPublicKey GETs the SSAS /system/{systemID}/key endpoint to retrieve a system's public key.
func (c *SSASClient) GetPublicKey(systemID int) ([]byte, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/system/%v/key", c.baseURL, systemID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "could not get public key")
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
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

// ResetCredentials PUTs to the SSAS /system/{systemID}/credentials endpoint to reset the system's secret.
func (c *SSASClient) ResetCredentials(systemID string) ([]byte, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/system/%s/credentials", c.baseURL, systemID), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to reset credentials")
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to reset credentials")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(fmt.Sprintf("failed to reset credentials. status code: %v", resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to reset credentials")
	}

	return body, nil

}

// DeleteCredentials DELETEs from the SSAS /system/{systemID}/credentials endpoint to deactivate credentials associated with the system.
func (c *SSASClient) DeleteCredentials(systemID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/system/%s/credentials", c.baseURL, systemID), nil)
	if err != nil {
		return errors.Wrap(err, "failed to delete credentials")
	}
	if err := c.setAuthHeader(req); err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to delete credentials")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, "failed to delete credentials")
	}

	return nil
}

// RevokeAccessToken DELETEs to the public SSAS /token endpoint to revoke the token
func (c *SSASClient) RevokeAccessToken(tokenID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/token/%s", c.baseURL, tokenID), nil)
	if err != nil {
		return errors.Wrap(err, "bad request structure")
	}

	if err := c.setAuthHeader(req); err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to revoke token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to revoke token; %v", resp.StatusCode)
	}

	return nil
}

// GetToken POSTs to the public SSAS /token endpoint to get an access token for a BCDA client
func (c *SSASClient) GetToken(credentials Credentials) ([]byte, error) {
	public := conf.GetEnv("SSAS_PUBLIC_URL")
	url := fmt.Sprintf("%s/token", public)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "bad request structure")
	}
	req.SetBasicAuth(credentials.ClientID, credentials.ClientSecret)

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
	public := conf.GetEnv("SSAS_PUBLIC_URL")
	url := fmt.Sprintf("%s/introspect", public)
	body, err := json.Marshal(struct {
		Token string `json:"token"`
	}{Token: tokenString})
	if err != nil {
		return nil, errors.Wrap(err, "bad request structure")
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "bad request structure")
	}

	clientID := conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	secret := conf.GetEnv("BCDA_SSAS_SECRET")
	if clientID == "" || secret == "" {
		return nil, errors.New("missing clientID or secret")
	}
	req.SetBasicAuth(clientID, secret)

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

func (c *SSASClient) setAuthHeader(req *http.Request) error {
	clientID := conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	secret := conf.GetEnv("BCDA_SSAS_SECRET")
	if clientID == "" || secret == "" {
		return errors.New("missing clientID or secret")
	}
	req.SetBasicAuth(clientID, secret)
	req.Header.Add("Accept", "application/json")
	return nil
}

// GetVersion Gets the version of the SSAS client
func (c *SSASClient) GetVersion() (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/_version", c.baseURL), nil)
	if err != nil {
		return "", err
	}

	if err := c.setAuthHeader(req); err != nil {
		return "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("SSAS server failed to return version ")
	}

	type ssasVersion struct {
		Version string `json:"version"`
	}
	var versionInfo = ssasVersion{}
	if err = json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return "", err
	}
	if versionInfo.Version == "" {
		return "", errors.New("Unable to parse version from response")
	}
	return versionInfo.Version, nil
}
