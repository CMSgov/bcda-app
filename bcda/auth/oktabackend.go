package auth

import (
	"bytes"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var publicKeys map[string]rsa.PublicKey
var once sync.Once

var oktaBaseUrl string
var authString string
var oktaServerID string

func config() error {
	log.SetFormatter(&log.JSONFormatter{})

	// required
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

	// manufactured
	authString = fmt.Sprintf("SSWS %s", oktaToken)

	return nil
}

type OB struct{}

func NewOB() *OB {
	var err error
	once.Do(func() {
		err = config()
		if err == nil {
			publicKeys, err = getPublicKeys()
		}
		go refreshKeys()
	})
	if err != nil {
		log.Warnf("no public keys available for server because %s", err)
		// our practice is to not stop the app, even when it's in a state where it can do nothing but emit errors
		// methods called on this ob value will result in errors until the publicKeys map is successfully updated
	}
	return &OB{}
}

func (ob *OB) PublicKeyFor(id string) (rsa.PublicKey, bool) {
	key, ok := publicKeys[id]
	log.Warnf("invalid key id %s presented", id)
	return key, ok
}

func (ob *OB) AddClientApplication(localID string) (string, string, error) {
	body := fmt.Sprintf(`{ "client_name": "%s", "client_uri": null, "logo_uri": null, "application_type": "service", "redirect_uris": [], "response_types": [ "token" ], "grant_types": [ "client_credentials" ], "token_endpoint_auth_method": "client_secret_basic" }`, localID)
	req, err := http.NewRequest("POST", oktaBaseUrl+"/oauth2/v1/clients", bytes.NewBufferString(body))
	if err != nil {
		return "", "", err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", authString)

	var client = &http.Client{Timeout: time.Second * 10,}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode != 201 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to create client %s because %v", localID, err)
			return "", "", err
		}
		return "", "", fmt.Errorf("unexpected result: %s", body)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", "", err
	}

	clientID := result["client_id"].(string)
	clientSecret := result["client_secret"].(string)

	err = addClientToPolicy(clientID)
	if err != nil {
		return "", "", fmt.Errorf("failed to add client %s to server policy because %v", localID, err)
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
func addClientToPolicy(clientID string) error {
	// get the current list
	policyUrl := fmt.Sprintf("%s/api/v1/authorizationServers/%s/policies", oktaBaseUrl, oktaServerID)
	req, err := http.NewRequest("GET", policyUrl, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", authString)

	var client = &http.Client{Timeout: time.Second * 10,}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	var result []Policy
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return err
	}

	if len(result) > 1 {
		return fmt.Errorf("more than one policy entry for server; can't continue safely")
	}

	// add the new client to the inclusion list
	incl := result[0].Conditions.Clients.Include
	incl = append(incl, clientID)
	result[0].Conditions.Clients.Include = incl

	// put the list back to the server
	body, err := json.Marshal(result[0])
	if err != nil {
		return err
	}

	req, err = http.NewRequest("PUT", fmt.Sprintf("%s/%s", policyUrl, result[0].ID), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", authString)

	client = &http.Client{Timeout: time.Second * 10,}
	resp, err = client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to update policy with %s because %v", clientID, err)
			return err
		}
		return fmt.Errorf("unexpected result: %s", body)
	}

	return nil
}

func refreshKeys() {
	for range time.Tick(time.Hour * 1) {
		log.Info("refreshing okta public keys")
		var err error
		publicKeys, err = getPublicKeys()
		if err != nil {
			log.Warnf("No public keys for ")
		}
	}
}

// Gets the current set of public server signing keys. This code treats all failures as fatal,
// since failures are likely to result in an unrecoverable state for the application.
func getPublicKeys() (map[string]rsa.PublicKey, error) {
	var (
		body []byte
		err  error
		keys map[string]rsa.PublicKey
	)

	body, err = get(fmt.Sprintf("%s/oauth2/%s/.well-known/oauth-authorization-server", oktaBaseUrl, oktaServerID))
	if err != nil {
		log.Errorf("can't fetch okta oauth server metadata because %v", err)
		return nil, err
	}
	md := make(map[string]interface{})
	if err = json.Unmarshal(body, &md); err != nil {
		log.Errorf("can't unmarshal %s because %v", string(body), err)
		return nil, err
	}

	jwkUrl := md["jwks_uri"].(string)
	body, err = get(jwkUrl)
	if err != nil {
		log.Errorf("can't get keys from %s because %v", jwkUrl, err)
		return nil, err
	}

	keys, err = parseKeys(body)
	if err != nil {
		log.Errorf("couldn't parse %s because %v", string(body), err)
		return nil, err
	}
	if len(keys) == 0 {
		err = fmt.Errorf("must have at least 1 public key; have none")
		log.Error(err)
		return nil, err
	}

	log.Infof("%d okta public oauth server public keys cached", len(keys))
	return keys, nil
}

type RsaJWK struct {
	KeyType   string `json:"kty"`
	Algorithm string `json:"alg"`
	ID        string `json:"kid"`
	Use       string `json:"use"`
	E         string `json:"e"`
	N         string `json:"n"`
}

type KeyList struct {
	Keys []*RsaJWK `json:"keys"`
}

func get(urlString string) ([]byte, error) {
	if urlString == "" {
		err := fmt.Errorf("a non-empty okta metadata url must be provided; is an env var not set?")
		log.Error(err)
		return nil, err
	}
	u, err := url.Parse(urlString)
	if err != nil {
		log.Errorf("invalid url %s because %v", urlString, err)
		return nil, err
	}

	var client = &http.Client{Timeout: time.Second * 10,}
	res, err := client.Get(u.String())
	if err != nil {
		log.Errorf("net failure to %s because %v", u.String(), err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		log.Errorf("okta responded with %v (expected 200)", res.StatusCode)
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		log.Errorf("couldn't read response body from okta because %v", err)
		return nil, err
	}

	return body, nil
}

func parseKeys(body []byte) (map[string]rsa.PublicKey, error) {
	data := KeyList{}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Errorf("can't unmarshal key %v because %s", body, err)
		return nil, err
	}

	pks := make(map[string]rsa.PublicKey)
	for _, v := range data.Keys {
		if v.KeyType != "RSA" || v.Algorithm != "RS256" || v.Use != "sig" {
			err := fmt.Errorf(fmt.Sprintf("malformed key (values %v)", v))
			log.Error(err)
			return nil, err
		}
		if len(v.N) != 342 {
			log.Warnf("N value is not 342 chars (%v)", v.N)
		}
		if len(v.E) != 4 {
			log.Warnf("E value is not 4 chars (%v)", v.N)
		}

		var err error
		pks[v.ID], err = toPublicKey(v.N, v.E)
		if err != nil {
			return nil, err
		}
	}

	return pks, nil
}

func toPublicKey(n string, e string) (rsa.PublicKey, error) {
	var (
		modulus  big.Int
		exponent big.Int
	)

	nbytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		log.Errorf("could not decode n value (%s) from key because %v", n, err)
		return rsa.PublicKey{}, err
	}
	modulus.SetBytes(nbytes)

	ebytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		log.Errorf("could not decode e value (%s) from key because %v", e, err)
		return rsa.PublicKey{}, err
	}
	exponent.SetBytes(ebytes)

	return rsa.PublicKey{N: &modulus, E: int(exponent.Int64())}, nil
}
