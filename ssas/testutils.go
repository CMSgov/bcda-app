package ssas

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
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

func RandomHexID() string {
	b, err := someRandomBytes(4)
	if err != nil {
		return "not_a_random_client_id"
	}
	return fmt.Sprintf("%x", b)
}

func RandomBase64(n int) string {
	b, err := someRandomBytes(20)
	if err != nil {
		return "not_a_random_base_64_string"
	}
	return base64.StdEncoding.EncodeToString(b)
}

func someRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

