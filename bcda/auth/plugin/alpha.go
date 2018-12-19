package auth

import (
	"errors"
	"regexp"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	jwt "github.com/dgrijalva/jwt-go"
)

type AlphaAuthPlugin struct{}

// This implementation expects one value in params, an id the API knows this client by in string form
// It returns a single string as well, being the clientID this implementation knows this client by
// NB: Other implementations will probably expect more input, and will certainly return more data
func (p *AlphaAuthPlugin) RegisterClient(params []byte) ([]byte, error) {
	var alphaID []byte
	acoUUID := string(params)

	// We'll check carefully in this method, because we're returning something to be used as an id
	// Normally, a plugin would treat this value as a black box external key, but this implementation is
	// intimate with the API. So, we're going to protect against accidental bad things
	if acoUUID == "" || len(acoUUID) > 36 {
		return alphaID, errors.New("you must provide a non-empty string 36 characters in length")
	}

	if matched, err := regexp.MatchString("^[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$", acoUUID); !matched || err != nil {
		return alphaID, errors.New("expected a valid UUID string")
	}

	if aco, err := getFromDB(acoUUID); err != nil {
		return alphaID, err
	} else {
		alphaID = []byte(strings.ToUpper(aco.UUID.String()))
	}

	// return the aco UUID as our auth client id. why? because we have to return something that the API / CLI will
	// use as our clientId for all the methods below. We could come up with yet another numbering scheme, or generate
	// more UUIDs, but I can't see a benefit in that. Plus, we will know just looking at the DB that any aco
	// whose client_id matches their UUID was created by this plugin.
	return alphaID, nil
}

func (p *AlphaAuthPlugin) UpdateClient(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) DeleteClient(params []byte) error {
	return errors.New("Not yet implemented")
}

// can treat as a no-op or call RequestAccessToken
func (p *AlphaAuthPlugin) GenerateClientCredentials(params []byte) ([]byte, error) {
	return nil, errors.New("Not yet implemented")
}

// look up the active access token associated with id, and call RevokeAccessToken
func (p *AlphaAuthPlugin) RevokeClientCredentials(params []byte) error {
	return errors.New("Not yet implemented")
}

// generate a token for the id (which user? just have a single "user" (alpha2, alpha3, ...) per test cycle?)
func (p *AlphaAuthPlugin) RequestAccessToken(params []byte) (jwt.Token, error) {
	return jwt.Token{}, errors.New("Not yet implemented")
}

// lookup token and set active flag to false
func (p *AlphaAuthPlugin) RevokeAccessToken(token string) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) ValidateAccessToken(token string) error {
	return errors.New("Not yet implemented")
}

func (p *AlphaAuthPlugin) DecodeAccessToken(token string) (jwt.Token, error) {
	return jwt.Token{}, errors.New("Not yet implemented")
}

func getFromDB(acoUUID string) (models.ACO, error) {
	var db = database.GetGORMDbConnection()
	var aco models.ACO
	db.Find(&aco, "UUID = ?", acoUUID)
	return aco, db.Error
}
