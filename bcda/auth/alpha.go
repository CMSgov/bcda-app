package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
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

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	aco.ClientID = localID
	aco.AlphaSecret = string(NewHash(s))
	db.Save(&aco)

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

func (p AlphaAuthPlugin) DeleteClient(params []byte) error {
	return errors.New("not yet implemented")
}

func (p AlphaAuthPlugin) GenerateClientCredentials(clientID string, ttl int) (Credentials, error) {
	aco, err := getACOFromDB(clientID)
	if err != nil {
		return Credentials{}, fmt.Errorf(`no ACO found for client ID %s because %s`, clientID, err)
	}

	if aco.ClientID == "" {
		return Credentials{}, fmt.Errorf("ACO %s does not have a registered client", clientID)
	}

	err = p.RevokeClientCredentials(clientID)
	if err != nil {
		return Credentials{}, fmt.Errorf("unable to revoke existing credentials for ACO %s because %s", clientID, err)
	}

	if ttl < 0 {
		return Credentials{}, errors.New("invalid TTL")
	}

	token, err := p.RequestAccessToken(Credentials{ClientID: clientID}, ttl)
	if err != nil {
		return Credentials{}, fmt.Errorf("unable to generate new credentials for ACO %s because %s", clientID, err)
	}

	return Credentials{Token: token}, err
}

// look up the active access token associated with id, and call RevokeAccessToken
func (p AlphaAuthPlugin) RevokeClientCredentials(clientID string) error {
	if clientID == "" {
		return errors.New("missing clientID argument")
	}

	db := database.GetGORMDbConnection()
	defer func() {
		if err := db.Close(); err != nil {
			log.Infof("error closing db connection in %s because %s", "alpha plugin", err)
		}
	}()

	var aco models.ACO
	if err := db.First(&aco, "client_id = ?", clientID).Error; err != nil {
		return fmt.Errorf("no ACO found for client ID because %s", err)
	}

	var users []models.User
	if err := db.Find(&users, "aco_id = ?", aco.UUID).Error; err != nil || len(users) == 0 {
		return fmt.Errorf("no users found in client's ACO because %s", err)
	}

	var (
		userIDs []uuid.UUID
		tokens  []Token
	)
	for _, u := range users {
		userIDs = append(userIDs, u.UUID)
	}

	db.Find(&tokens, "user_id in (?) and active = true", userIDs)
	if len(tokens) == 0 {
		log.Info("No tokens found to revoke for users in client's ACO.")
		return nil
	}

	var errs []string
	revokedCount := 0
	for _, t := range tokens {
		err := revokeAccessTokenByID(t.UUID)
		if err != nil {
			log.Error(err)
			errs = append(errs, err.Error())
		} else {
			revokedCount = revokedCount + 1
		}
	}
	log.Infof("%d token(s) revoked.", revokedCount)
	if len(errs) > 0 {
		return fmt.Errorf("%d of %d token(s) could not be revoked due to errors", len(errs), len(tokens))
	}

	return nil
}

// AcessToken manufactures an access token for the given credentials
func (p AlphaAuthPlugin) AccessToken(credentials Credentials) (string, error) {
	if credentials.ClientSecret == "" || credentials.ClientID == "" {
		return "", fmt.Errorf("missing or incomplete credentials")
	}
	aco, err := getACOByClientID(credentials.ClientID)
	if err != nil {
		return "", fmt.Errorf("invalid credentials; %s",err)
	}
	// when we have ClientSecret in ACO, adjust following line
	Hash(/*aco.ClientSecret*/"hashed db value").IsHashOf(credentials.ClientSecret)
	var user models.User
	if database.GetGORMDbConnection().First(&user, "aco_id = ?", aco.UUID).RecordNotFound() {
		return "", fmt.Errorf("invalid credentials; unable to locate User for ACO with id of %s", aco.UUID)
	}
	issuedAt := time.Now().Unix()
	expiresAt := time.Now().Add(time.Hour * time.Duration(tokenTTL)).Unix()
	return GenerateTokenString(uuid.NewRandom().String(), user.UUID.String(), aco.UUID.String(), issuedAt, expiresAt)
}

// RequestAccessToken generate a token for the ACO, either for a specified UserID or (if not provided) any user in the ACO
func (p AlphaAuthPlugin) RequestAccessToken(creds Credentials, ttl int) (Token, error) {
	var userUUID, acoUUID uuid.UUID
	var user models.User
	var err error
	token := Token{}

	if creds.UserID == "" && creds.ClientID == "" {
		return token, fmt.Errorf("must provide either UserID or ClientID")
	}

	if ttl < 0 {
		return token, fmt.Errorf("invalid TTL: %d", ttl)
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	if creds.UserID != "" {
		userUUID = uuid.Parse(creds.UserID)
		if userUUID == nil {
			return token, fmt.Errorf("user ID must be a UUID")
		}

		if db.First(&user, "UUID = ?", creds.UserID).RecordNotFound() {
			return token, fmt.Errorf("unable to locate User with id of %s", creds.UserID)
		}

		userUUID = user.UUID
		acoUUID = user.ACOID
	} else {
		var aco models.ACO
		aco, err = getACOFromDB(creds.ClientID)
		if err != nil {
			return token, err
		}

		if err = db.First(&user, "aco_id = ?", aco.UUID).Error; err != nil {
			return token, errors.New("no user found for " + aco.UUID.String())
		}

		userUUID = user.UUID
		acoUUID = aco.UUID
	}

	token.UUID = uuid.NewRandom()
	token.UserID = userUUID
	token.ACOID = acoUUID
	token.IssuedAt = time.Now().Unix()
	token.ExpiresOn = time.Now().Add(time.Hour * time.Duration(ttl)).Unix()
	token.Active = true

	if err = db.Create(&token).Error; err != nil {
		return Token{}, err
	}

	token.TokenString, err = GenerateTokenString(token.UUID.String(), token.UserID.String(), token.ACOID.String(), token.IssuedAt, token.ExpiresOn)
	if err != nil {
		return Token{}, err
	}

	return token, nil
}

func (p AlphaAuthPlugin) RevokeAccessToken(tokenString string) error {
	t, err := p.DecodeJWT(tokenString)
	if err != nil {
		return err
	}

	if c, ok := t.Claims.(*CommonClaims); ok {
		return revokeAccessTokenByID(uuid.Parse(c.UUID))
	}

	return errors.New("could not read token claims")
}

func revokeAccessTokenByID(tokenID uuid.UUID) error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var token Token
	if db.First(&token, "UUID = ? and active = true", tokenID).RecordNotFound() {
		return gorm.ErrRecordNotFound
	}

	token.Active = false
	db.Save(&token)

	return db.Error
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

	b := isActive(t)
	if !b {
		return fmt.Errorf("token with id: %v is not active", c.UUID)
	}

	return nil
}

func checkRequiredClaims(claims *CommonClaims) error {
	if claims.ExpiresAt == 0 ||
		claims.IssuedAt == 0 ||
		claims.Subject == "" ||
		claims.ACOID == "" ||
		claims.UUID == "" {
		return fmt.Errorf("missing one or more required claims")
	}
	return nil
}

func isActive(token *jwt.Token) bool {
	c := token.Claims.(*CommonClaims)

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	var dbt Token
	return !db.Find(&dbt, "UUID = ? AND active = ?", c.UUID, true).RecordNotFound()
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
