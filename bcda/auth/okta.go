package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	jwt "github.com/dgrijalva/jwt-go"
)

type OktaBackend interface {
	// Returns the current set of public signing keys for the okta auth server
	PublicKeyFor(id string) (rsa.PublicKey, bool)

	// Returns the id of the authorization server
	ServerID() string

	// Adds an api client application to our Okta organization
	AddClientApplication(string) (clientID string, secret string, clientName string, err error)

	// Gets a session token from Okta
	RequestAccessToken(creds client.Credentials) (client.OktaToken, error)

	// Renews client secret for an okta client
	GenerateNewClientSecret(string) (string, error)

	// Deactivates a client application so it cannot be used
	DeactivateApplication(clientID string) error
}

type OktaAuthPlugin struct {
	backend OktaBackend // interface, not a concrete type, so no *
}

// Create a new plugin using the provided backend. Having the backend passed in facilitates testing with Mockta.
func NewOktaAuthPlugin(backend OktaBackend) OktaAuthPlugin {
	return OktaAuthPlugin{backend}
}

func (o OktaAuthPlugin) RegisterSystem(localID, publicKey string) (Credentials, error) {
	if localID == "" {
		return Credentials{}, errors.New("you must provide a localID")
	}

	id, secret, name, err := o.backend.AddClientApplication(localID)

	return Credentials{
		ClientID:     id,
		ClientSecret: secret,
		ClientName:   name,
	}, err
}

func (o OktaAuthPlugin) UpdateSystem(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (o OktaAuthPlugin) DeleteSystem(clientID string) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) ResetSecret(clientID string) (Credentials, error) {
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

func (o OktaAuthPlugin) RevokeSystemCredentials(clientID string) error {
	err := o.backend.DeactivateApplication(clientID)
	if err != nil {
		return err
	}

	return nil
}

// Manufactures an access token for the given credentials
func (o OktaAuthPlugin) MakeAccessToken(creds Credentials) (string, error) {
	clientID := creds.ClientID
	// Also accept clientID via creds.UserID to match alpha auth implementation
	if clientID == "" {
		clientID = creds.UserID
	}

	if clientID == "" {
		return "", fmt.Errorf("client ID required")
	}

	if creds.ClientSecret == "" {
		return "", fmt.Errorf("client secret required")
	}

	clientCreds := client.Credentials{ClientID: clientID, ClientSecret: creds.ClientSecret}
	ot, err := o.backend.RequestAccessToken(clientCreds)

	if err != nil {
		return "", err
	}

	return ot.AccessToken, nil
}

func (o OktaAuthPlugin) RevokeAccessToken(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) AuthorizeAccess(tokenString string) error {
	t, err := o.VerifyToken(tokenString)
	if err != nil {
		return err
	}

	c := t.Claims.(*CommonClaims)

	ok := c.Issuer == o.backend.ServerID()
	if !ok {
		return fmt.Errorf("invalid iss claim; %s <> %s", c.Issuer, o.backend.ServerID())
	}

	err = c.Valid()
	if err != nil {
		return err
	}

	// need to check revocation here, which is not yet implemented
	// options:
	// keep an in-memory cache of tokens we have revoked and check that
	// use the introspection endpoint okta provides (expensive network call)

	_, err = GetACOByClientID(c.ClientID)
	if err != nil {
		return fmt.Errorf("invalid cid claim; %s", err)
	}

	return nil
}

func (o OktaAuthPlugin) VerifyToken(tokenString string) (*jwt.Token, error) {
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

	return jwt.ParseWithClaims(tokenString, &CommonClaims{}, keyFinder)
}
