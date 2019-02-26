package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

type Mokta struct {
	publicKey  rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func NewMokta() *Mokta {
	reader := rand.Reader
	bitSize := 1024

	privateKey, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		log.Fatal(err)
	}
	publicKey := privateKey.PublicKey

	keys := make(map[string]rsa.PublicKey)
	keys["mokta"] = publicKey

	return &Mokta{publicKey, privateKey}
}

func (m *Mokta) PublicKeyFor(id string) (rsa.PublicKey, bool) {
	if id != "mokta" {
		return rsa.PublicKey{}, false
	}
	return m.publicKey, true
}

func (m *Mokta) AddClientApplication(localId string) (string, string, error) {
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

func (m *Mokta) GenerateNewClientSecret(clientID string) (string, error) {
	if len(clientID) != 20 {
		return "", errors.New("404 Not Found")
	}

	fakeClientSecret := "thisClientSecretIsFakeButIsCorrectLength"
	return fakeClientSecret, nil
}

func someRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Returns a new access token
func (m *Mokta) NewToken(clientID, acoID string, expiresIn int) (string, error) {
	tid, err := someRandomBytes(32)
	if err != nil {
		return "", err
	}
	token := jwt.New(jwt.SigningMethodRS256)
	token.Header["kid"] = "mokta"
	token.Claims = jwt.MapClaims{
		"ver": 1,
		"jti": base64.URLEncoding.EncodeToString(tid),
		"iss": "mokta.fake.backend",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * time.Duration(expiresIn)).Unix(),
		"cid": clientID,
		"scp": []string{"bcda_api"},
		"sub": clientID,
	}
	tokenString, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
