package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
)

type OktaBackend interface {
	// Returns the current set of public signing keys for the okta auth server
	PublicKeyFor(id string) (rsa.PublicKey, bool)

	// Adds an api client application to our Okta organization
	AddClientApplication(string) (string, string, error)

	// Renews client secret for an okta client
	GenerateNewClientSecret(string) (string, error)
}

type OktaAuthPlugin struct {
	backend OktaBackend // interface, not a concrete type, so no *
}

// Create a new plugin using the provided backend. Having the backend passed in facilitates testing with Mockta.
func NewOktaAuthPlugin(backend OktaBackend) OktaAuthPlugin {
	return OktaAuthPlugin{backend}
}

func (o OktaAuthPlugin) RegisterClient(localID string) (Credentials, error) {
	if localID == "" {
		return Credentials{}, errors.New("you must provide a localID")
	}

	id, key, err := o.backend.AddClientApplication(localID)
	return Credentials{
		ClientID:     id,
		ClientSecret: key,
	}, err
}

func (o OktaAuthPlugin) UpdateClient(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (o OktaAuthPlugin) DeleteClient(params []byte) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) GenerateClientCredentials(clientID string, ttl int) (Credentials, error) {
	clientSecret, err := o.backend.GenerateNewClientSecret(clientID)
	if err != nil {
		return Credentials{}, err
	}

	c := Credentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	return c, nil
}

func (o OktaAuthPlugin) RevokeClientCredentials(params []byte) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) RequestAccessToken(params []byte) (Token, error) {
	return Token{}, errors.New("not yet implemented")
}

func (o OktaAuthPlugin) RevokeAccessToken(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) ValidateJWT(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) DecodeJWT(tokenString string) (*jwt.Token, error) {
	keyFinder := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		keyID, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("no key id in token header? %v", token.Header)
		}

		key, ok := o.backend.PublicKeyFor(keyID)
		if !ok {
			return nil, fmt.Errorf("no key found with id %s", keyID)
		}

		return &key, nil
	}

	return jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, keyFinder)
}
