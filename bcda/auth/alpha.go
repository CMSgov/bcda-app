package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
)

type AlphaAuthPlugin struct{}

func (p AlphaAuthPlugin) RegisterClient(localID string) (Credentials, error) {

	if localID == "" {
		return Credentials{}, errors.New("provide a non-empty string")
	}

	aco, err := getACOFromDB(localID)
	if err != nil {
		return Credentials{}, err
	}

	if aco.AlphaSecret != "" {
		return Credentials{}, fmt.Errorf("aco %s has a secret", localID)
	}

	s, err := generateClientSecret()
	if err != nil {
		return Credentials{}, err
	}

	hashedSecret, err := NewHash(s)
	if err != nil {
		return Credentials{}, err
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	aco.ClientID = localID
	aco.AlphaSecret = hashedSecret.String()
	err = db.Save(&aco).Error
	if err != nil {
		return Credentials{}, err
	}

	return Credentials{ClientName: aco.Name, ClientID: localID, ClientSecret: s}, nil
}

func generateClientSecret() (string, error) {
	b := make([]byte, 40)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", b), nil
}

func (p AlphaAuthPlugin) UpdateClient(params []byte) ([]byte, error) {
	return nil, errors.New("not yet implemented")
}

func (p AlphaAuthPlugin) DeleteClient(clientID string) error {
	aco, err := GetACOByClientID(clientID)
	if err != nil {
		return err
	}

	aco.ClientID = ""
	aco.AlphaSecret = ""

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err = db.Save(&aco).Error
	if err != nil {
		return err
	}

	return nil
}

func (p AlphaAuthPlugin) GenerateClientCredentials(clientID string) (Credentials, error) {

	if clientID == "" {
		return Credentials{}, errors.New("provide a non-empty string")
	}

	aco, err := getACOFromDB(clientID)
	if err != nil {
		return Credentials{}, err
	}

	s, err := generateClientSecret()
	if err != nil {
		return Credentials{}, err
	}

	hashedSecret, err := NewHash(s)
	if err != nil {
		return Credentials{}, err
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	aco.AlphaSecret = hashedSecret.String()
	err = db.Save(&aco).Error
	if err != nil {
		return Credentials{}, err
	}

	return Credentials{ClientName: aco.Name, ClientID: clientID, ClientSecret: s}, nil
}

func (p AlphaAuthPlugin) RevokeClientCredentials(clientID string) error {
	return fmt.Errorf("RevokeClientCredentials is not implemented for alpha auth")
}

// MakeAccessToken manufactures an access token for the given credentials
func (p AlphaAuthPlugin) MakeAccessToken(credentials Credentials) (string, error) {
	if credentials.ClientSecret == "" || credentials.ClientID == "" {
		return "", fmt.Errorf("missing or incomplete credentials")
	}

	if uuid.Parse(credentials.ClientID) == nil {
		return "", fmt.Errorf("ClientID must be a valid UUID")
	}

	aco, err := GetACOByClientID(credentials.ClientID)
	if err != nil {
		return "", fmt.Errorf("invalid credentials; %s", err)
	}
	if !Hash(aco.AlphaSecret).IsHashOf(credentials.ClientSecret) {
		return "", fmt.Errorf("invalid credentials")
	}
	issuedAt := time.Now().Unix()
	expiresAt := time.Now().Add(time.Hour * time.Duration(TokenTTL)).Unix()
	return GenerateTokenString(uuid.NewRandom().String(), aco.UUID.String(), issuedAt, expiresAt)
}

// RequestAccessToken generates a token for the ACO
func (p AlphaAuthPlugin) RequestAccessToken(creds Credentials, ttl int) (Token, error) {
	var err error
	token := Token{}

	if creds.ClientID == "" {
		return token, fmt.Errorf("must provide ClientID")
	}

	if uuid.Parse(creds.ClientID) == nil {
		return token, fmt.Errorf("ClientID must be a valid UUID")
	}

	if ttl < 0 {
		return token, fmt.Errorf("invalid TTL: %d", ttl)
	}

	var aco models.ACO
	aco, err = getACOFromDB(creds.ClientID)
	if err != nil {
		return token, err
	}

	token.UUID = uuid.NewRandom()
	token.ACOID = aco.UUID
	token.IssuedAt = time.Now().Unix()
	token.ExpiresOn = time.Now().Add(time.Hour * time.Duration(ttl)).Unix()
	token.Active = true

	token.TokenString, err = GenerateTokenString(token.UUID.String(), token.ACOID.String(), token.IssuedAt, token.ExpiresOn)
	if err != nil {
		return Token{}, err
	}

	return token, nil
}

func (p AlphaAuthPlugin) RevokeAccessToken(tokenString string) error {
	return fmt.Errorf("RevokeAccessToken is not implemented for alpha auth")
}

func (p AlphaAuthPlugin) ValidateJWT(tokenString string) error {
	t, err := p.DecodeJWT(tokenString)
	if err != nil {
		log.Errorf("could not decode token %s because %s", tokenString, err)
		return err
	}

	c := t.Claims.(*CommonClaims)

	err = checkRequiredClaims(c)
	if err != nil {
		return err
	}

	err = c.Valid()
	if err != nil {
		return err
	}

	_, err = getACOFromDB(c.ACOID)
	if err != nil {
		return err
	}

	return nil
}

func checkRequiredClaims(claims *CommonClaims) error {
	if claims.ExpiresAt == 0 ||
		claims.IssuedAt == 0 ||
		claims.ACOID == "" ||
		claims.UUID == "" {
		return fmt.Errorf("missing one or more required claims")
	}
	return nil
}

func (p AlphaAuthPlugin) DecodeJWT(tokenString string) (*jwt.Token, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return InitAlphaBackend().PublicKey, nil
	}

	return jwt.ParseWithClaims(tokenString, &CommonClaims{}, keyFunc)
}

func getACOFromDB(acoUUID string) (models.ACO, error) {
	var (
		db  = database.GetGORMDbConnection()
		aco models.ACO
		err error
	)
	defer database.Close(db)

	if db.Find(&aco, "UUID = ?", uuid.Parse(acoUUID)).RecordNotFound() {
		err = errors.New("no ACO record found for " + acoUUID)
	}
	return aco, err
}