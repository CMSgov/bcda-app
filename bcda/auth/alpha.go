package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

type AlphaAuthPlugin struct{}

func (p AlphaAuthPlugin) RegisterClient(localID string) (Credentials, error) {
	regEvent := event{op: "RegisterClient", trackingID: localID}
	operationStarted(regEvent)
	if localID == "" {
		// do we want to report on usage errors?
		regEvent.help = "provide a non-empty string"
		operationFailed(regEvent)
		return Credentials{}, errors.New(regEvent.help)
	}

	aco, err := getACOFromDB(localID)
	if err != nil {
		regEvent.help = err.Error()
		operationFailed(regEvent)
		return Credentials{}, err
	}

	if aco.AlphaSecret != "" {
		regEvent.help = fmt.Sprintf("aco %s has a secret", localID)
		operationFailed(regEvent)
		return Credentials{}, errors.New(regEvent.help)
	}

	s, err := generateClientSecret()
	if err != nil {
		regEvent.help = err.Error()
		operationFailed(regEvent)
		return Credentials{}, err
	}
	secretCreated(regEvent)

	hashedSecret, err := NewHash(s)
	if err != nil {
		regEvent.help = err.Error()
		operationFailed(regEvent)
		return Credentials{}, err
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	aco.ClientID = localID
	aco.AlphaSecret = hashedSecret.String()
	err = db.Save(&aco).Error
	if err != nil {
		regEvent.help = err.Error()
		operationFailed(regEvent)
		return Credentials{}, err
	}

	regEvent.clientID = aco.ClientID
	operationSucceeded(regEvent)
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
	delEvent := event{op: "DeleteClient", trackingID: clientID}
	operationStarted(delEvent)
	aco, err := GetACOByClientID(clientID)
	if err != nil {
		delEvent.help = err.Error()
		operationFailed(delEvent)
		return err
	}

	aco.ClientID = ""
	aco.AlphaSecret = ""

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err = db.Save(&aco).Error
	if err != nil {
		delEvent.help = err.Error()
		operationFailed(delEvent)
		return err
	}

	operationSucceeded(delEvent)
	return nil
}

func (p AlphaAuthPlugin) GenerateClientCredentials(clientID string) (Credentials, error) {
	genEvent := event{op: "GenerateClientCredentials", trackingID: clientID}
	operationStarted(genEvent)

	if clientID == "" {
		genEvent.help = "provide a non-empty string"
		operationFailed(genEvent)
		return Credentials{}, errors.New("provide a non-empty string")
	}

	aco, err := getACOFromDB(clientID)
	if err != nil {
		genEvent.help = err.Error()
		operationFailed(genEvent)
		return Credentials{}, err
	}

	s, err := generateClientSecret()
	if err != nil {
		genEvent.help = err.Error()
		operationFailed(genEvent)
		return Credentials{}, err
	}

	hashedSecret, err := NewHash(s)
	if err != nil {
		genEvent.help = err.Error()
		operationFailed(genEvent)
		return Credentials{}, err
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	aco.AlphaSecret = hashedSecret.String()
	err = db.Save(&aco).Error
	if err != nil {
		genEvent.help = err.Error()
		operationFailed(genEvent)
		return Credentials{}, err
	}

	operationSucceeded(genEvent)
	return Credentials{ClientName: aco.Name, ClientID: clientID, ClientSecret: s}, nil
}

func (p AlphaAuthPlugin) RevokeClientCredentials(clientID string) error {
	return fmt.Errorf("RevokeClientCredentials is not implemented for alpha auth")
}

// MakeAccessToken manufactures an access token for the given credentials
func (p AlphaAuthPlugin) MakeAccessToken(credentials Credentials) (string, error) {
	tknEvent := event{op: "MakeAccessToken", trackingID: credentials.ClientID}
	if credentials.ClientSecret == "" || credentials.ClientID == "" {
		tknEvent.help = "missing or incomplete credentials"
		operationFailed(tknEvent)
		return "", fmt.Errorf("missing or incomplete credentials")
	}

	if uuid.Parse(credentials.ClientID) == nil {
		tknEvent.help = "missing or incomplete credentials"
		operationFailed(tknEvent)
		return "", fmt.Errorf("ClientID must be a valid UUID")
	}

	aco, err := GetACOByClientID(credentials.ClientID)
	if err != nil {
		tknEvent.help = err.Error()
		operationFailed(tknEvent)
		return "", fmt.Errorf("invalid credentials; %s", err)
	}
	if !Hash(aco.AlphaSecret).IsHashOf(credentials.ClientSecret) {
		tknEvent.help = "IsHashOf failed"
		operationFailed(tknEvent)
		return "", fmt.Errorf("invalid credentials")
	}
	issuedAt := time.Now().Unix()
	expiresAt := time.Now().Add(TokenTTL).Unix()
	uuid := uuid.NewRandom().String()
	tknEvent.tokenID = uuid
	operationSucceeded(tknEvent)
	accessTokenIssued(tknEvent)
	return GenerateTokenString(uuid, aco.UUID.String(), issuedAt, expiresAt)
}

// RequestAccessToken generates a token for the ACO (Deprecated, use MakeAccessToken()
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
	tknEvent := event{op: "ValidateJWT"}
	operationStarted(tknEvent)
	t, err := p.DecodeJWT(tokenString)
	if err != nil {
		tknEvent.help = err.Error()
		operationFailed(tknEvent)
		// can we log the fail token here
		return err
	}

	c := t.Claims.(*CommonClaims)

	err = checkRequiredClaims(c)
	if err != nil {
		tknEvent.help = err.Error()
		operationFailed(tknEvent)
		return err
	}

	err = c.Valid()
	if err != nil {
		tknEvent.help = err.Error()
		operationFailed(tknEvent)
		return err
	}

	_, err = getACOFromDB(c.ACOID)
	if err != nil {
		tknEvent.help = err.Error()
		operationFailed(tknEvent)
		return err
	}

	operationSucceeded(tknEvent)
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
