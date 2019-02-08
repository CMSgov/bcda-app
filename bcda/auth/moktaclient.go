package auth

import (
	"crypto/rand"
	"encoding/base64"
)

// adds a client application, returning the client_id and secret_key
func addClientApplication(localId string) (string, string, error) {
	id, err := someRandomBytes(16)
	if err != nil {
		return "", "", nil
	}
	key, err := someRandomBytes(32)
	if err != nil {
		return "", "", nil
	}

	return base64.URLEncoding.EncodeToString(id), base64.URLEncoding.EncodeToString(key), err
}

func someRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}