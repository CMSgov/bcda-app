package auth

import (
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/secutils"
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

func (backend *JWTAuthenticationBackend) IsBlacklisted(jwtToken *jwt.Token) bool {
	claims, _ := jwtToken.Claims.(jwt.MapClaims)
	db := database.GetGORMDbConnection()
	defer database.Close(db)

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

// should be removed during BCDA-764; must be left in place until then
func (backend *JWTAuthenticationBackend) CreateToken(user models.User) (token Token, tokenString string, err error) {
	if user.UUID == nil || user.ACOID == nil {
		err = errors.New("invalid user model parameter")
		return
	}

	tokenID := uuid.NewRandom().String()
	tokenString, err = TokenStringWithIDs(
		tokenID,
		user.UUID.String(),
		user.ACOID.String(),
	)
	if err != nil {
		panic(err)
	}
	// Get the claims of the token to find the token ID that was created
	claims := backend.GetJWTClaims(tokenString)
	tid, ok := claims["id"].(string)
	if !ok || tid == "" {
		err = fmt.Errorf(`missing claim "id"; got claims %v`, claims)
		return
	}

	token = Token{
		UUID:   uuid.Parse(tid),
		UserID: user.UUID,
		ACOID:  user.ACOID,
		Value:  tokenString,
		Active: true,
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err = db.Create(&token).Error
	if err != nil {
		log.Errorf("unable to create token for aco %v because %s", user.ACOID, err)
	}

	return token, tokenString, err
}

func (backend *JWTAuthenticationBackend) SignJwtToken(token jwt.Token) (string, error) {
	return token.SignedString(backend.PrivateKey)
}
