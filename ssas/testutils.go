package ssas

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

func GeneratePublicKey(bits int) (string, error) {
	keyPair, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", fmt.Errorf("unable to generate keyPair: %s", err.Error())
	}

	publicKeyPKIX, err := x509.MarshalPKIXPublicKey(&keyPair.PublicKey)
	if err != nil {
		return "", fmt.Errorf("unable to marshal public key: %s", err.Error())
	}

	publicKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyPKIX,
	})

	return string(publicKeyBytes), nil
}
