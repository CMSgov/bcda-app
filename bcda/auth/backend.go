package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/secutils"
	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
)

var (
	jwtExpirationDelta  string                    = os.Getenv("JWT_EXPIRATION_DELTA")
	authBackendInstance *JWTAuthenticationBackend = nil
)

type Hash struct{}

func (c *Hash) Generate(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func (c *Hash) Compare(hash string, s string) bool {
	return hash == c.Generate(s)
}

type JWTAuthenticationBackend struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

func InitAuthBackend() *JWTAuthenticationBackend {
	if authBackendInstance == nil {
		authBackendInstance = &JWTAuthenticationBackend{
			PrivateKey: getPrivateKey(),
			PublicKey:  getPublicKey(),
		}
	}

	return authBackendInstance
}

// For testing.  Probably no real use case.
func (backend *JWTAuthenticationBackend) ResetAuthBackend() {

	authBackendInstance = &JWTAuthenticationBackend{
		PrivateKey: getPrivateKey(),
		PublicKey:  getPublicKey(),
	}
}

func (backend *JWTAuthenticationBackend) GenerateTokenString(userID, acoID string) (string, error) {
	expirationDelta, err := strconv.Atoi(jwtExpirationDelta)
	if err != nil {
		expirationDelta = 72
	}

	token := jwt.New(jwt.SigningMethodRS512)
	token.Claims = jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * time.Duration(expirationDelta)).Unix(),
		"iat": time.Now().Unix(),
		"sub": userID,
		"aco": acoID,
		"id":  uuid.NewRandom(),
	}
	return token.SignedString(backend.PrivateKey)
}

func (backend *JWTAuthenticationBackend) RevokeToken(tokenString string) error {
	db := database.GetGORMDbConnection()
	claims := backend.GetJWTClaims(tokenString)

	if claims == nil {
		return errors.New("Could not read token claims")
	}

	tokenID := claims["id"].(string)

	var token Token
	hash := Hash{}
	if db.First(&token, "value = ? and UUID = ? and active = ?", hash.Generate(tokenString), tokenID, true).RecordNotFound() {
		return gorm.ErrRecordNotFound
	} else {
		token.Active = false
		db.Save(&token)
	}

	return db.Error
}

func (backend *JWTAuthenticationBackend) RevokeUserTokens(user models.User) error {
	db := database.GetGORMDbConnection()
	var token Token
	db.Model(&token).Where("active = ? and User_id = ?", true, user.UUID).Update("active", false)
	return db.Error
}

func (backend *JWTAuthenticationBackend) RevokeACOTokens(aco models.ACO) error {
	db := database.GetGORMDbConnection()
	users := []models.User{} // a slice of users
	db.Find(&users, "aco_id = ?", aco.UUID)
	for _, user := range users {
		if err := backend.RevokeUserTokens(user); err != nil {
			return err
		}

	}
	return db.Error
}

func (backend *JWTAuthenticationBackend) IsBlacklisted(jwtToken *jwt.Token) bool {
	claims, _ := jwtToken.Claims.(jwt.MapClaims)
	db := database.GetGORMDbConnection()
	defer db.Close()

	var token Token
	// Look for an inactive token with the uuid; if found, it is blacklisted or otherwise revoked
	if db.Find(&token, "UUID = ? AND active = ?", claims["id"], false).RecordNotFound() {
		return false
	} else {
		return true
	}
}

// This method and its sibling, getPublicKey(), get the private key from the file system and environment variables.
// They accesses external resources and so may panic and bubble up an error if the file is not present or otherwise corrupted
func getPrivateKey() *rsa.PrivateKey {
	privateKeyFile, err := os.Open(os.Getenv("JWT_PRIVATE_KEY_FILE"))
	if err != nil {
		log.Panic(err)
	}
	return secutils.OpenPrivateKeyFile(privateKeyFile)
}

// panics if file is not found, corrupted, or otherwise unreadable
func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open(os.Getenv("JWT_PUBLIC_KEY_FILE"))
	if err != nil {
		panic(err)
	}
	return secutils.OpenPublicKeyFile(publicKeyFile)
}

func (backend *JWTAuthenticationBackend) GetJWTClaims(tokenString string) jwt.MapClaims {
	token, err := backend.GetJWToken(tokenString)

	// err is returned if anything goes wrong, including expired token
	if err != nil {
		return nil
	}

	return token.Claims.(jwt.MapClaims)
}

func (backend *JWTAuthenticationBackend) GetJWToken(tokenString string) (*jwt.Token, error) {

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return backend.PublicKey, nil
	})
	return token, err
}

// Save a token to the DB for a user
func (backend *JWTAuthenticationBackend) CreateToken(user models.User) (Token, string, error) {
	db := database.GetGORMDbConnection()
	tokenString, err := backend.GenerateTokenString(
		user.UUID.String(),
		user.AcoID.String(),
	)
	if err != nil {
		panic(err)
	}
	// Get the claims of the token to find the token ID that was created
	claims := backend.GetJWTClaims(tokenString)
	token := Token{
		UUID:   uuid.Parse(claims["id"].(string)),
		UserID: user.UUID,
		Value:  tokenString,
		Active: true,
	}
	db.Create(&token)

	return token, tokenString, err
}

// CLI command only support; note that we are choosing to fail quickly and let the user (one of us) figure it out
func createAlphaACO(db *gorm.DB) (models.ACO, error) {
	var count int
	db.Table("acos").Count(&count)
	aco := models.ACO{Name: fmt.Sprintf("Alpha ACO %d", count), UUID: uuid.NewRandom()}
	db.Create(&aco)

	return aco, db.Error
}

// CLI command only support; note that we are choosing to fail quickly and let the user (one of us) figure it out
func createAlphaUser(db *gorm.DB, aco models.ACO) (models.User, error) {
	var count int
	db.Table("users").Count(&count)
	user := models.User{UUID: uuid.NewRandom(),
		Name:  fmt.Sprintf("Alpha User%d", count),
		Email: fmt.Sprintf("alpha.user.%d@nosuchdomain.com", count), AcoID: aco.UUID}
	db.Create(&user)

	return user, db.Error
}

func assignBeneficiaries(db *gorm.DB, aco models.ACO, acoSize string) error {
	s := "insert into beneficiaries (patient_id, aco_id) select patient_id, '" + aco.UUID.String() +
		"' from beneficiaries where aco_id = (select uuid from acos where name = 'ACO " + acoSize + "')"
	return db.Exec(s).Error
}

func (backend *JWTAuthenticationBackend) CreateAlphaToken(timeToLive, acoSize string) (string, error) {
	var aco models.ACO
	var user models.User
	var tokenString string
	var err error
	var originalJwtExpirationDelta = jwtExpirationDelta

	defer func() {
		jwtExpirationDelta = originalJwtExpirationDelta
	}()

	if len(timeToLive) > 0 {
		jwtExpirationDelta = timeToLive
	}

	tx := database.GetGORMDbConnection().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return "", tx.Error
	}

	if aco, err = createAlphaACO(tx); err != nil {
		tx.Rollback()
		return "", err
	}

	if err = assignBeneficiaries(tx, aco, acoSize); err != nil {
		tx.Rollback()
		return "", err
	}

	if user, err = createAlphaUser(tx, aco); err != nil {
		tx.Rollback()
		return "", err
	}

	if tx.Commit().Error != nil {
		tx.Rollback()
		return "", tx.Error
	}

	_, tokenString, err = backend.CreateToken(user)

	return tokenString, err
}

func (backend *JWTAuthenticationBackend) SignJwtToken(token jwt.Token) (string, error) {
	return token.SignedString(backend.PrivateKey)
}
