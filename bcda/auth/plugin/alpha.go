package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
)

type AlphaAuthPlugin struct{}

type CustomClaims struct {
	Aco string `json:"aco"`
	ID  string `json:"id"`
	jwt.StandardClaims
}

// This implementation expects one value in params, an id the API knows this client by in string form
// It returns a single string as well, being the clientID this implementation knows this client by
// NB: Other implementations will probably expect more input, and will certainly return more data
func (p *AlphaAuthPlugin) RegisterClient(params []byte) ([]byte, error) {
	var empty []byte

	acoUUID, err := GetParamString(params, "clientID")
	if err != nil {
		return empty, err
	}

	// We'll check carefully in this method, because we're returning something to be used as an id
	// Normally, a plugin would treat this value as a black box external key, but this implementation is
	// intimate with the API. So, we're going to protect against accidental bad things
	if len(acoUUID) != 36 {
		return empty, errors.New("you must provide a non-empty string 36 characters in length")
	}

	if matched, err := regexp.MatchString("^[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$", acoUUID); !matched || err != nil {
		return empty, errors.New("expected a valid UUID string")
	}

	if _, err := getACOFromDB(acoUUID); err != nil {
		return empty, err
	}

	// return the aco UUID as our auth client id. why? because we have to return something that the API / CLI will
	// use as our clientId for all the methods below. We could come up with yet another numbering scheme, or generate
	// more UUIDs, but I can't see a benefit in that. Plus, we will know just looking at the DB that any aco
	// whose client_id matches their UUID was created by this plugin.
	return params, nil
}

func (p *AlphaAuthPlugin) UpdateClient(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (p *AlphaAuthPlugin) DeleteClient(params []byte) error {
	return errors.New("not yet implemented")
}

// can treat as a no-op or call RequestAccessToken
func (p *AlphaAuthPlugin) GenerateClientCredentials(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

// look up the active access token associated with id, and call RevokeAccessToken
func (p *AlphaAuthPlugin) RevokeClientCredentials(params []byte) error {
	return errors.New("not yet implemented")
}

// generate a token for the id (which user? just have a single "user" (alpha2, alpha3, ...) per test cycle?)
// params are currently acoId and ttl; not going to introduce user until we have clear use cases
func (p *AlphaAuthPlugin) RequestAccessToken(params []byte) (jwt.Token, error) {
	backend := auth.InitAuthBackend()
	db := database.GetGORMDbConnection()
	jwtToken := jwt.Token{}

	acoUUID, err := GetParamString(params, "clientID")
	if err != nil {
		return jwtToken, err
	}

	aco, err := getACOFromDB(acoUUID)
	if err != nil {
		return jwtToken, err
	}

	// I arbitrarily decided to use the first user. An alternative would be to make a specific user
	// that represents the client. I have no strong opinion here other than not creating stuff in the db
	// unless we're willing to live with it forever.
	var user models.User
	if err = db.First(&user, "aco_id = ?", aco.UUID).Error; err != nil {
		return jwtToken, errors.New("no user found for " + aco.UUID.String())
	}

	ttl, err := GetParamInt(params, "ttl")
	if err != nil {
		return jwtToken, errors.New("no valid ttl found because " + err.Error())
	}

	tokenUUID := uuid.NewRandom()
	jwtToken = *jwt.New(jwt.SigningMethodRS512)
	jwtToken.Claims = jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * time.Duration(ttl)).Unix(),
		"iat": time.Now().Unix(),
		"sub": user.UUID.String(),
		"aco": aco.UUID.String(),
		"id":  tokenUUID.String(),
	}

	tokenString, err := backend.SignJwtToken(jwtToken)
	if err != nil {
		return jwtToken, err
	}

	token := auth.Token{
		UUID:        tokenUUID,
		UserID:      user.UUID,
		Value:       tokenString,	// replaced with hash when saved to db
		Active:      true,
		Token:       jwtToken,
		TokenString: tokenString,
	}

	if err = db.Create(&token).Error; err != nil {
		return jwtToken, err
	}

	return jwtToken, err // really want to return auth.Token here, but first let's get this all working
}

// lookup token and set active flag to false
func (p *AlphaAuthPlugin) RevokeAccessToken(token string) error {
	return errors.New("not yet implemented")
}

func (p *AlphaAuthPlugin) ValidateAccessToken(token string) error {
	return errors.New("not yet implemented")
}

func (p *AlphaAuthPlugin) DecodeAccessToken(token string) (jwt.Token, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return auth.InitAuthBackend().PublicKey, nil
	}
	t, err := jwt.ParseWithClaims(token, &CustomClaims{}, keyFunc)
	if err != nil {
		return jwt.Token{}, err
	}
	return *t, nil
}

func getACOFromDB(acoUUID string) (models.ACO, error) {
	var (
		db  = database.GetGORMDbConnection()
		aco models.ACO
		err error
	)

	if db.Find(&aco, "UUID = ?", acoUUID).RecordNotFound() {
		err = errors.New("no ACO record found for " + acoUUID)
	}
	return aco, err
}

func GetParamString(params []byte, name string) (string, error) {
	var (
		j   interface{}
		err error
	)

	if err = json.Unmarshal(params, &j); err != nil {
		return "", err
	}
	paramsMap := j.(map[string]interface{})

	stringForName, ok := paramsMap[name].(string)
	if !ok {
		return "", errors.New("missing or otherwise invalid string value for " + name)
	}

	return stringForName, err
}

func GetParamInt(params []byte, name string) (int, error) {
	var (
		j   interface{}
		err error
	)

	if err = json.Unmarshal(params, &j); err != nil {
		return -1, err
	}
	paramsMap := j.(map[string]interface{})

	valueForName, ok := paramsMap[name].(float64)
	if !ok {
		return -1, errors.New("missing or otherwise invalid int value for " + name)
	}

	return int(valueForName), err
}
