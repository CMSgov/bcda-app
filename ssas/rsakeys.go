package ssas

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
)

const RSAKEYMINBITS = 2048

// ReadPublicKey reads a string containing a PEM-formatted public key and returns a pointer to an rsa.PublicKey type
// or an error. The key must have a length of at least 2048 bits, and it must be an rsa key.
func ReadPublicKey (publicKey string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return nil, fmt.Errorf("not able to decode PEM-formatted public key")
	}

	publicKeyImported, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse public key: %s", err.Error())
	}

	rsaPub, ok := publicKeyImported.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not able to cast key as *rsa.PublicKey")
	}

	if rsaPub.Size() < RSAKEYMINBITS / 8 {
		return nil, fmt.Errorf("insecure key length (%d bytes)", rsaPub.Size())
	}

	return rsaPub, nil
}

// ConvertJWKToPEM extracts the (hopefully single) public key contained in a jwks.
// Modified from source at: https://play.golang.org/p/mLpOxS-5Fy
func ConvertJWKToPEM(jwks string) (string, error) {
	j := map[string]string{}
	err := json.Unmarshal([]byte(jwks), &j)
	if err != nil {
		return "", errors.New("unable to parse JSON for jwk: " + err.Error())
	}

	if j["kty"] != "RSA" {
		return "", errors.New("invalid key type: " + j["kty"] + "; only 'RSA' accepted")
	}

	if j["use"] != "" && j["use"] != "enc" {
		return "", errors.New("invalid use type: " + j["use"] + "; only 'enc' accepted")
	}

	nb, err := base64.RawURLEncoding.DecodeString(j["n"])
	if err != nil {
		return "", errors.New("base64 error in key n value: " + err.Error())
	}
	nv := new(big.Int).SetBytes(nb)

	eb, err := base64.RawURLEncoding.DecodeString(j["e"])
	if err != nil {
		return "", errors.New("base64 error in key exponent: " + err.Error())
	}

	bigE := new(big.Int).SetBytes(eb)
	if !bigE.IsInt64() {
		return "", errors.New("key exponent too large: " + bigE.String())
	}
	ev := int(bigE.Int64())

	pk := &rsa.PublicKey{
		N: nv,
		E: ev,
	}

	der, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return "", errors.New("unable to marshal public key: " + err.Error())
	}

	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: der,
	}

	var out bytes.Buffer
	err = pem.Encode(&out, block)
	if err != nil {
		return "", errors.New("unable to encode key in PEM format: " + err.Error())
	}

	return out.String(), nil
}