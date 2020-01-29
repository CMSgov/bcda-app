package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
)

type Mokta struct {
	publicKey   rsa.PublicKey
	privateKey  *rsa.PrivateKey
	publicKeyID string
	serverID    string
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

	return &Mokta{publicKey, privateKey, "mokta", "mokta.fake.backend"}
}

func (m *Mokta) PublicKeyFor(id string) (rsa.PublicKey, bool) {
	if id != m.publicKeyID {
		return rsa.PublicKey{}, false
	}
	return m.publicKey, true
}

func (m *Mokta) ServerID() string {
	return m.serverID
}

func (m *Mokta) AddClientApplication(localId string) (clientID string, clientSecret string, clientName string, err error) {
	id, err := someRandomBytes(16)
	if err != nil {
		return
	}
	key, err := someRandomBytes(32)
	if err != nil {
		return
	}

	clientID = base64.URLEncoding.EncodeToString(id)
	clientSecret = base64.URLEncoding.EncodeToString(key)
	clientName = fmt.Sprintf("BCDA %s", clientID)
	return
}

func (m *Mokta) RequestAccessToken(creds client.Credentials) (client.OktaToken, error) {
	if creds.ClientID == "" {
		return client.OktaToken{}, fmt.Errorf("client ID required")
	}

	if creds.ClientSecret == "" {
		return client.OktaToken{}, fmt.Errorf("client secret required")
	}

	mt, err := m.NewToken(creds.ClientID)

	if err != nil {
		fmt.Printf("mokta RequestAccessToken() error: %v\n", err.Error())
		return client.OktaToken{}, err
	}

	return client.OktaToken{
		AccessToken: mt,
		TokenType:   "mokta_token",
		ExpiresIn:   300,
		Scope:       "bcda_api",
	}, nil
}

func (m *Mokta) GenerateNewClientSecret(clientID string) (string, error) {
	if len(clientID) != 20 {
		return "", errors.New("404 Not Found")
	}

	fakeClientSecret := "thisClientSecretIsFakeButIsCorrectLength"
	return fakeClientSecret, nil
}

func (m *Mokta) DeactivateApplication(clientID string) error {
	return nil
}

func randomClientID() string {
	b, err := someRandomBytes(4)
	if err != nil {
		return "not_a_random_client_id"
	}
	return fmt.Sprintf("%x", b)
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
func (m *Mokta) NewToken(clientID string) (string, error) {
	return m.NewCustomToken(OktaToken{
		m.publicKeyID,
		m.serverID,
		500,
		clientID,
		[]string{"bcda_api"},
		clientID,
	})
}

type OktaToken struct {
	KeyID     string   `json:"kid,omitempty"`
	Issuer    string   `json:"iss,omitempty"`
	ExpiresIn int64    `json:"exp,omitempty"`
	ClientID  string   `json:"cid,omitempty"`
	Scopes    []string `json:"scp,omitempty"`
	Subject   string   `json:"sub,omitempty"`
}

func (m *Mokta) NewCustomToken(overrides OktaToken) (string, error) {
	values := m.valuesWithOverrides(overrides)
	tid, err := someRandomBytes(32)
	if err != nil {
		return "", err
	}

	token := jwt.New(jwt.SigningMethodRS256)
	token.Header["kid"] = values.KeyID
	token.Claims = jwt.MapClaims{
		"ver": 1,
		"jti": base64.URLEncoding.EncodeToString(tid),
		"iss": values.Issuer,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour * time.Duration(values.ExpiresIn)).Unix(),
		"cid": values.ClientID,
		"scp": values.Scopes,
	}

	return token.SignedString(m.privateKey)
}

func (m *Mokta) valuesWithOverrides(or OktaToken) OktaToken {
	cid := randomClientID()
	v := OktaToken{
		m.publicKeyID,
		m.serverID,
		500,
		cid,
		[]string{"bcda_api"},
		cid,
	}

	if or.ClientID != "" {
		v.ClientID = or.ClientID
	}

	if or.ExpiresIn != 0 {
		v.ExpiresIn = or.ExpiresIn
	}

	if or.Issuer != "" {
		v.Issuer = or.Issuer
	}

	if or.KeyID != "" {
		v.KeyID = or.KeyID
	}

	if len(or.Scopes) != 0 {
		v.Scopes = make([]string, len(or.Scopes))
		copy(v.Scopes, or.Scopes)
	}

	if or.Subject != "" {
		v.Subject = or.Subject
	}

	return v
}
