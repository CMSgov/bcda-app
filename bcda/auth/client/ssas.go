package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"

	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/pkg/errors"
)

// SSASClient is a client for interacting with the System-to-System Authentication Service.
type SSASClient struct {
	http.Client
	baseURL string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in,omitempty"`
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
		return errors.Wrap(err, constants.DeleteGroupErr)
	}
	if err := c.setAuthHeader(req); err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, constants.DeleteGroupErr)
	}

	if resp.StatusCode != http.StatusOK {
		rb, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return errors.Wrap(err, constants.DeleteGroupErr)
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
		return nil, errors.Wrap(err, constants.SystemCreateErr)
	}
	br := bytes.NewReader(bb)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/system", c.baseURL), br)
	if err != nil {
		return nil, errors.Wrap(err, constants.SystemCreateErr)
	}
	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, constants.SystemCreateErr)
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
		return nil, errors.Wrap(err, constants.PublicKeyErr)
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, constants.PublicKeyErr)
	}
	defer resp.Body.Close()

	var respMap map[string]string
	if err = json.NewDecoder(resp.Body).Decode(&respMap); err != nil {
		return nil, errors.Wrap(err, constants.PublicKeyErr)
	}

	return []byte(respMap["public_key"]), nil
}

// ResetCredentials PUTs to the SSAS /system/{systemID}/credentials endpoint to reset the system's secret.
func (c *SSASClient) ResetCredentials(systemID string) ([]byte, error) {
	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/system/%s/credentials", c.baseURL, systemID), nil)
	if err != nil {
		return nil, errors.Wrap(err, constants.ResetCredentialsErr)
	}

	if err := c.setAuthHeader(req); err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, constants.ResetCredentialsErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(fmt.Sprintf("failed to reset credentials. status code: %v", resp.StatusCode))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, constants.ResetCredentialsErr)
	}

	return body, nil

}

// DeleteCredentials DELETEs from the SSAS /system/{systemID}/credentials endpoint to deactivate credentials associated with the system.
func (c *SSASClient) DeleteCredentials(systemID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/system/%s/credentials", c.baseURL, systemID), nil)
	if err != nil {
		return errors.Wrap(err, constants.DeleteCredentialsErr)
	}
	if err := c.setAuthHeader(req); err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, constants.DeleteCredentialsErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Wrap(err, constants.DeleteCredentialsErr)
	}

	return nil
}

// RevokeAccessToken DELETEs to the public SSAS /token endpoint to revoke the token
func (c *SSASClient) RevokeAccessToken(tokenID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/token/%s", c.baseURL, tokenID), nil)
	if err != nil {
		return errors.Wrap(err, constants.RequestStructErr)
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
func (c *SSASClient) GetToken(credentials Credentials) (string, error) {
	public := conf.GetEnv("SSAS_PUBLIC_URL")
	tokenUrl := fmt.Sprintf("%s/token", public)
	req, err := http.NewRequest("POST", tokenUrl, nil)
	if err != nil {
		return "", &customErrors.RequestError{Err: err, Msg: constants.RequestStructErr}
	}
	req.SetBasicAuth(credentials.ClientID, credentials.ClientSecret)

	resp, err := c.Do(req)
	if err != nil {
		if urlError, ok := err.(*url.Error); ok && urlError.Timeout() {
			return "", &customErrors.RequestTimeoutError{Err: err, Msg: constants.TokenRequestTimeoutErr}
		}
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)

		if err == nil {
			bodyString := string(bodyBytes)
			bodyStringToLowerCase := strings.ToLower(bodyString)
			isUnauthorized := strings.Contains(bodyStringToLowerCase, "unauthorized")

			defer resp.Body.Close()

			if isUnauthorized {
				return "", &customErrors.UnauthorizedError{Err: err, Msg: constants.TokenRequestUnauthorizedErr}
			}
			return "", &customErrors.UnexpectedSSASError{Err: err, SsasStatusCode: resp.StatusCode, Msg: constants.TokenRequestUnexpectedErr}
		}
	}

	var t = TokenResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", &customErrors.InternalParsingError{Err: err, Msg: "token request failed - error parsing response to json"}
	}

	return fmt.Sprintf(`{"access_token": "%s", "expires_in": "%s", "token_type":"bearer"}`, t.AccessToken, t.ExpiresIn), nil
}

func (c *SSASClient) Ping() error {

	tokenString := "None"
	public := conf.GetEnv("SSAS_PUBLIC_URL")
	url := fmt.Sprintf("%s/introspect", public)
	body, err := json.Marshal(struct {
		Token string `json:"token"`
	}{Token: tokenString})
	if err != nil {
		return errors.Wrap(err, constants.RequestStructErr)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, constants.RequestStructErr)
	}

	clientID := conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	secret := conf.GetEnv("BCDA_SSAS_SECRET")
	if clientID == "" || secret == "" {
		return errors.New(constants.MissingIDSecretErr)
	}
	req.SetBasicAuth(clientID, secret)

	resp, err := c.Do(req)
	if err != nil {
		return errors.Wrap(err, "introspect request failed")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("introspect request failed; %v", resp.StatusCode)
	}

	return nil
}

// CallSSASIntrospect verifies that the tokenString presented was issued by the public server.
// It does so using the introspect endpoint as defined by https://tools.ietf.org/html/rfc7662
func (c *SSASClient) CallSSASIntrospect(tokenString string) ([]byte, error) {

	request, err := constructIntrospectRequest(tokenString)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(request)
	if err != nil {
		if urlError, ok := err.(*url.Error); ok && urlError.Timeout() {
			return nil, &customErrors.RequestTimeoutError{Err: err, Msg: "introspect request failed - the SSAS introspect request timed out"}
		} else {
			return nil, &customErrors.UnexpectedSSASError{Err: err, Msg: "introspect request failed - unexpected error occured while performing SSAS introspect request"}
		}
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &customErrors.UnexpectedSSASError{Err: errors.New("Status code NOT equal to 200 (Status OK)"), SsasStatusCode: resp.StatusCode, Msg: fmt.Sprintf("Status code received in introspect response is %v", resp.StatusCode)}
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, &customErrors.InternalParsingError{Err: err, Msg: "could not read the introspect response body"}
	}

	return bytes, nil
}

//constructIntrospectRequest constructs the necessary request for SSAS introspect call.
//it grabs the necessary environmental variables, creates reqeust body, determines
//the url to hit, and sets the basic authorization header.
func constructIntrospectRequest(tokenString string) (request *http.Request, err error) {
	sSASPublicURL, clientID, clientSecret, err := getEnvVariablesForIntrospectCall()
	if err != nil {
		return nil, err
	}

	body, err := createIntrospectRequestBody(tokenString)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/introspect", sSASPublicURL)
	request, err = http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, &customErrors.InternalParsingError{Err: err, Msg: "unable to construct http.Request for SSAS introspect"}
	}

	request.SetBasicAuth(clientID, clientSecret)
	return request, nil
}

//createIntrospectRequestBody creates the body that will be used in the SSAS introspect call.
func createIntrospectRequestBody(tokenString string) (body []byte, err error) {
	body, err = json.Marshal(struct {
		Token string `json:"token"`
	}{Token: tokenString})

	if err != nil {
		return nil, &customErrors.InternalParsingError{Err: err, Msg: "unable to marshal SSAS introspect request body to json format"}
	}
	return body, nil
}

//getEnvVariablesForIntrospectCall retrieves necessary environmental variables
//for SSAS introspect call and verifies they are not empty.
func getEnvVariablesForIntrospectCall() (sSASPublicURL string, clientID string, clientSecret string, err error) {
	sSASPublicURL = conf.GetEnv("SSAS_PUBLIC_URL")
	clientID = conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	clientSecret = conf.GetEnv("BCDA_SSAS_SECRET")

	missingSSASPublicURL := sSASPublicURL == ""
	missingClientID := clientID == ""
	missingClientSecret := clientSecret == ""

	if missingSSASPublicURL || missingClientID || missingClientSecret {
		error := &customErrors.ConfigError{Err: errors.New("Configuration Error"), Msg: fmt.Sprintf("environmental configuration missing for: sSASPublicURL = %t, clientID = %t, clientSecret = %t", missingSSASPublicURL, missingClientID, missingClientSecret)}
		return "", "", "", error
	}
	return sSASPublicURL, clientID, clientSecret, nil
}

func (c *SSASClient) setAuthHeader(req *http.Request) error {
	clientID := conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	secret := conf.GetEnv("BCDA_SSAS_SECRET")
	if clientID == "" || secret == "" {
		return errors.New(constants.MissingIDSecretErr)
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
