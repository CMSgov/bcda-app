package auth

import (
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	jwt "github.com/dgrijalva/jwt-go"
)

type OktaBackend interface {
	// Returns the current set of public signing keys for the okta auth server
	PublicKeyFor(id string) (rsa.PublicKey, bool)

	// Returns the id of the authorization server
	ServerID() string

	// Adds an api client application to our Okta organization
	AddClientApplication(string) (string, string, error)

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

func (o OktaAuthPlugin) RevokeClientCredentials(clientID string) error {
	err := o.backend.DeactivateApplication(clientID)
	if err != nil {
		return err
	}

	return nil
}

func (o OktaAuthPlugin) RequestAccessToken(creds Credentials, ttl int) (Token, error) {
	if creds.ClientID == "" {
		return Token{}, fmt.Errorf("client ID required")
	}

	if creds.ClientSecret == "" {
		return Token{}, fmt.Errorf("client secret required")
	}

	clientCreds := client.Credentials{ClientID: creds.ClientID, ClientSecret: creds.ClientSecret}
	ot, err := o.backend.RequestAccessToken(clientCreds)

	if err != nil {
		return Token{}, err
	}

	return Token{TokenString: ot.AccessToken}, nil
}

func (o OktaAuthPlugin) RevokeAccessToken(tokenString string) error {
	return errors.New("not yet implemented")
}

func (o OktaAuthPlugin) ValidateJWT(tokenString string) error {
	t, err := o.DecodeJWT(tokenString)
	if err != nil {
		return err
	}

	c := t.Claims.(jwt.MapClaims)

	ok := c["iss"].(string) == o.backend.ServerID()
	if !ok {
		return fmt.Errorf("invalid iss claim; %s <> %s", c["iss"].(string), o.backend.ServerID())
	}

	err = c.Valid()
	if err != nil {
		return err
	}

	// need to check revocation here, which is not yet implemented
	// options:
	// keep an in-memory cache of tokens we have revoked and check that
	// use the introspection endpoint okta provides (expensive network call)

	_, err = getACOByClientID(c["cid"].(string))
	if err != nil {
		return fmt.Errorf("invalid cid claim; %s", err)
	}

	return nil
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

func getACOByClientID(clientID string) (models.ACO, error) {
	var (
		db  = database.GetGORMDbConnection()
		aco models.ACO
		err error
	)
	defer database.Close(db)

	if db.Find(&aco, "client_id = ?", clientID).RecordNotFound() {
		err = errors.New("no ACO record found for " + clientID)
	}
	return aco, err
}
