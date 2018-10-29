package auth

import (
	"bufio"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
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

func (backend *JWTAuthenticationBackend) CreateACO(name string) (uuid.UUID, error) {
	db := database.GetGORMDbConnection()
	defer db.Close()
	aco := ACO{Name: name, UUID: uuid.NewRandom()}
	db.Create(&aco)

	return aco.UUID, db.Error
}

func (backend *JWTAuthenticationBackend) CreateUser(name string, email string, acoUUID uuid.UUID) (User, error) {
	db := database.GetGORMDbConnection()
	defer db.Close()
	var aco ACO
	var user User
	// If we don't find the ACO return a blank user and an error
	if db.First(&aco, "UUID = ?", acoUUID).RecordNotFound() {
		return user, fmt.Errorf("unable to locate ACO with id of %v", acoUUID)
	}
	// check for duplicate email addresses and only make one if it isn't found
	if db.First(&user, "email = ?", email).RecordNotFound() {
		user = User{UUID: uuid.NewRandom(), Name: name, Email: email, AcoID: aco.UUID}
		db.Create(&user)
		return user, nil
	} else {

		return user, fmt.Errorf("unable to create user for %v because a user with that Email address already exists", email)
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
	//userID := claims["sub"].(string)

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

func (backend *JWTAuthenticationBackend) RevokeUserTokens(user User) error {
	db := database.GetGORMDbConnection()
	var token Token
	db.Model(&token).Where("active = ? and User_id = ?", true, user.UUID).Update("active", false)
	return db.Error
}

func (backend *JWTAuthenticationBackend) RevokeACOTokens(aco ACO) error {
	db := database.GetGORMDbConnection()
	users := []User{} // a slice of users
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
	// Look for the token, if found, it must be blacklisted, if not it should be fine.
	if db.Find(&token, "UUID = ? AND active = ?", claims["id"], false).RecordNotFound() {
		return false
	} else {
		return true
	}

}

func getPrivateKey() *rsa.PrivateKey {
	privateKeyFile, err := os.Open(os.Getenv("JWT_PRIVATE_KEY_FILE"))
	if err != nil {
		log.Panic(err)
	}
	pemfileinfo, _ := privateKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)
	if err != nil {
		// Above buffer.Read succeeded on a blank file Not Sure how to reach this
		log.Panic(err)
	}

	data, _ := pem.Decode([]byte(pembytes))
	privateKeyFile.Close()

	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)
	if err != nil {
		// Above function panicked when receiving a bad and blank key file.  This may be unreachable
		log.Panic(err)
	}

	return privateKeyImported
}

// THis method gets the private key from the file system and environment variables.  It accesses external resources
// so it may panic and bubble up an error if the file is not present or otherwise corrupted
func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open(os.Getenv("JWT_PUBLIC_KEY_FILE"))
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := publicKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(publicKeyFile)
	_, err = buffer.Read(pembytes)
	if err != nil {
		log.Fatal(err)
	}

	data, _ := pem.Decode([]byte(pembytes))

	publicKeyFile.Close()

	publicKeyImported, err := x509.ParsePKIXPublicKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	rsaPub, ok := publicKeyImported.(*rsa.PublicKey)

	if !ok {
		panic(err)
	}

	return rsaPub
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
func (backend *JWTAuthenticationBackend) CreateToken(user User) (Token, string, error) {
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
func createAlphaACO(db *gorm.DB) (ACO, error) {
	var count int
	db.Table("acos").Count(&count)
	aco := ACO{Name: fmt.Sprintf("Alpha ACO %d", count), UUID: uuid.NewRandom()}
	db.Create(&aco)

	return aco, db.Error
}

// CLI command only support; note that we are choosing to fail quickly and let the user (one of us) figure it out
func createAlphaUser(db *gorm.DB, aco ACO) (User, error) {
	var count int
	db.Table("users").Count(&count)
	user := User{UUID: uuid.NewRandom(),
	Name: fmt.Sprintf("Alpha User%d", count),
	Email: fmt.Sprintf("alpha.user.%d@nosuchdomain.com", count), AcoID: aco.UUID}
	db.Create(&user)

	return user, db.Error
}

func assignBeneficiaries(db *gorm.DB, aco ACO) error {
	s := "insert into beneficiaries (patient_id, aco_id) select patient_id, '" + aco.UUID.String() +
		"' from beneficiaries where aco_id = (select uuid from acos where name = 'ACO Dev')"
	return db.Exec(s).Error
}

func (backend *JWTAuthenticationBackend) CreateAlphaToken() (ACO, User, string, error) {
	var aco ACO
	var user User
	var tokenString string
	var err error

	tx := database.GetGORMDbConnection().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if tx.Error != nil {
		return aco, user, tokenString, tx.Error
	}

	if aco, err = createAlphaACO(tx); err != nil {
		tx.Rollback()
		return aco, user, tokenString, tx.Error
	}

	if err = assignBeneficiaries(tx, aco); err != nil {
		tx.Rollback()
		return aco, user, tokenString, tx.Error
	}

	if user, err = createAlphaUser(tx, aco); err != nil {
		tx.Rollback()
		return aco, user, tokenString, tx.Error
	}

	if tx.Commit().Error != nil {
		tx.Rollback()
		return aco, user, tokenString, tx.Error
	}

	_, tokenString, err = backend.CreateToken(user)

	return aco, user, tokenString, err
}
