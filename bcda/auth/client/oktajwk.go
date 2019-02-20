package client

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/url"
	"time"
)

// Gets the current set of public server signing keys. This code treats all failures as fatal,
// since failures are likely to result in an unrecoverable state for the application.
func getPublicKeys() (map[string]rsa.PublicKey) {
	var (
		body []byte
		err  error
		keys map[string]rsa.PublicKey
	)

	body, err = get(fmt.Sprintf("%s/oauth2/%s/.well-known/oauth-authorization-server", oktaBaseUrl, oktaServerID))
	if err != nil {
		logger.Errorf("can't fetch okta oauth server metadata because %v", err)
		return nil
	}
	md := make(map[string]interface{})
	if err = json.Unmarshal(body, &md); err != nil {
		logger.Errorf("can't unmarshal %s because %v", string(body), err)
		return nil
	}

	jwkUrl := md["jwks_uri"].(string)
	body, err = get(jwkUrl)
	if err != nil {
		logger.Errorf("can't get keys from %s because %v", jwkUrl, err)
		return nil
	}

	keys, err = parseKeys(body)
	if err != nil {
		logger.Errorf("couldn't parse %s because %v", string(body), err)
		return nil
	}
	if len(keys) == 0 {
		logger.Errorf("must have at least 1 public key; have none")
		return nil
	}

	logger.Infof("%d okta public oauth server public keys cached", len(keys))
	return keys
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
		err := fmt.Errorf("a non-empty okta metadata url must be provided; is an env var not set")
		logger.Error(err)
		return nil, err
	}
	u, err := url.Parse(urlString)
	if err != nil {
		logger.Errorf("invalid url %s because %v", urlString, err)
		return nil, err
	}

	var client = &http.Client{Timeout: time.Second * 10,}
	res, err := client.Get(u.String())
	if err != nil {
		logger.Errorf("net failure to %s because %v", u.String(), err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		logger.Errorf("okta responded with %v (expected 200)", res.StatusCode)
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		logger.Errorf("couldn't read response body from okta because %v", err)
		return nil, err
	}

	return body, nil
}

func parseKeys(body []byte) (map[string]rsa.PublicKey, error) {
	data := KeyList{}
	if err := json.Unmarshal(body, &data); err != nil {
		logger.Errorf("can't unmarshal key %v because %s", body, err)
		return nil, err
	}

	pks := make(map[string]rsa.PublicKey)
	for _, v := range data.Keys {
		if v.KeyType != "RSA" || v.Algorithm != "RS256" || v.Use != "sig" {
			err := fmt.Errorf(fmt.Sprintf("malformed key (values %v)", v))
			logger.Error(err)
			return nil, err
		}
		if len(v.N) != 342 {
			logger.Warnf("N value is not 342 chars (%v)", v.N)
		}
		if len(v.E) != 4 {
			logger.Warnf("E value is not 4 chars (%v)", v.N)
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
		logger.Errorf("could not decode n value (%s) from key because %v", n, err)
		return rsa.PublicKey{}, err
	}
	modulus.SetBytes(nbytes)

	ebytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		logger.Errorf("could not decode e value (%s) from key because %v", e, err)
		return rsa.PublicKey{}, err
	}
	exponent.SetBytes(ebytes)

	return rsa.PublicKey{N: &modulus, E: int(exponent.Int64())}, nil
}