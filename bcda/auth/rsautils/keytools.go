package rsautils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
)

const RSAKEYMINBITS = 2048

func ReadPublicKey (publicKey string) (*rsa.PublicKey, error) {
	if publicKey == "" {
		return nil, fmt.Errorf("empty string provided")
	}

	emptyPEMRegex := "-----BEGIN RSA PUBLIC KEY-----(\\W*)-----END RSA PUBLIC KEY-----"
	emptyPEM, err := regexp.MatchString(emptyPEMRegex, publicKey)
	if err != nil {
		return nil, fmt.Errorf("regex error searching for empty keys")
	}
	if emptyPEM {
		return nil, fmt.Errorf("empty key")
	}

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