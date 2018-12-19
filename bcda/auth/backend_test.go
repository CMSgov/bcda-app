package auth_test

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BackendTestSuite struct {
	testUtils.AuthTestSuite
}

func (s *BackendTestSuite) SetupSuite() {
	fmt.Println("Initializing models for auth.backend testing")
	models.InitializeGormModels()
	auth.InitializeGormModels()
}

func (s *BackendTestSuite) SetupTest() {
	s.SetupAuthBackend()
}

func (s *BackendTestSuite) TestInitAuthBackend() {
	assert.IsType(s.T(), &auth.JWTAuthenticationBackend{}, s.AuthBackend)
	assert.IsType(s.T(), &rsa.PrivateKey{}, s.AuthBackend.PrivateKey)
	assert.IsType(s.T(), &rsa.PublicKey{}, s.AuthBackend.PublicKey)
}

func (s *BackendTestSuite) TestHashCompare() {
	uuidString := uuid.NewRandom().String()
	hash := auth.Hash{}
	hashString := hash.Generate(uuidString)
	assert.True(s.T(), hash.Compare(hashString, uuidString))
	assert.False(s.T(), hash.Compare(hashString, uuid.NewRandom().String()))
}

func (s *BackendTestSuite) TestGenerateToken() {
	userUUIDString, acoUUIDString := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	token, err := s.AuthBackend.GenerateTokenString(userUUIDString, acoUUIDString)

	// No errors, token is not nil
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	// Wipe the keys
	s.AuthBackend.PrivateKey = nil
	s.AuthBackend.PublicKey = nil
	defer s.AuthBackend.ResetAuthBackend()
	assert.Panics(s.T(), func() { _, _ = s.AuthBackend.GenerateTokenString(userUUIDString, acoUUIDString) })
}

func (s *BackendTestSuite) TestCreateToken() {
	userID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	db := database.GetGORMDbConnection()
	var user models.User
	if db.Find(&user, "UUID = ?", userID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to locate user"))
	}
	token, _, err := s.AuthBackend.CreateToken(user)
	assert.NotNil(s.T(), token.UUID)
	assert.Nil(s.T(), err)

	// Wipe the keys
	s.AuthBackend.PrivateKey = nil
	s.AuthBackend.PublicKey = nil
	defer s.AuthBackend.ResetAuthBackend()
	assert.Panics(s.T(), func() { _, _, _ = s.AuthBackend.CreateToken(user) })
}

func (s *BackendTestSuite) TestGetJWClaims() {
	acoID, userID := uuid.NewRandom().String(), uuid.NewRandom().String()
	goodToken, err := s.AuthBackend.GenerateTokenString(acoID, userID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), s.AuthBackend.GetJWTClaims(goodToken))

	// Check an expired token
	expiredToken := jwt.New(jwt.SigningMethodRS512)
	expiredToken.Claims = jwt.MapClaims{
		"exp": 12345,
		"iat": 123,
		"sub": userID,
		"aco": acoID,
		"id":  uuid.NewRandom(),
	}
	expiredTokenString, err := expiredToken.SignedString(s.AuthBackend.PrivateKey)
	assert.Nil(s.T(), err)
	invalidClaims := s.AuthBackend.GetJWTClaims(expiredTokenString)
	assert.Nil(s.T(), invalidClaims)

	// Check an incorrectly signed token.
	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"
	badClaims := s.AuthBackend.GetJWTClaims(badToken)
	assert.Nil(s.T(), badClaims)
}

func (s *BackendTestSuite) TestRevokeToken() {
	db := database.GetGORMDbConnection()
	userID, acoID := "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	var user models.User
	db.Find(&user, "UUID = ? AND aco_id = ?", userID, acoID)
	// Good Revoke test
	_, tokenString, _ := s.AuthBackend.CreateToken(user)

	err := s.AuthBackend.RevokeToken(userID)
	assert.NotNil(s.T(), err)
	err = s.AuthBackend.RevokeToken(tokenString)
	assert.Nil(s.T(), err)
	jwtToken, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	assert.True(s.T(), s.AuthBackend.IsBlacklisted(jwtToken))

	// Revoke the token again, you can't
	err = s.AuthBackend.RevokeToken(tokenString)
	assert.NotNil(s.T(), err)

	// Revoke a token that doesn't exist
	tokenString, _ = s.AuthBackend.GenerateTokenString(uuid.NewRandom().String(), acoID)
	err = s.AuthBackend.RevokeToken(tokenString)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), gorm.IsRecordNotFoundError(err))

}

func (s *BackendTestSuite) TestRevokeUserTokens() {
	revokedEmail, validEmail := "userrevoked@email.com", "usernotrevoked@email.com"

	db := database.GetGORMDbConnection()
	// Get the User
	var revokedUser, validUser models.User
	if db.First(&revokedUser, "Email = ?", revokedEmail).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find User"))
	}
	if db.First(&validUser, "Email = ?", validEmail).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find User"))
	}
	// make 2 valid tokens for this user to be revoked later
	_, revokedTokenString, err := s.AuthBackend.CreateToken(revokedUser)
	revokedJWTToken, _ := s.AuthBackend.GetJWToken(revokedTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.AuthBackend.IsBlacklisted(revokedJWTToken))

	_, otherRevokedTokenString, err := s.AuthBackend.CreateToken(revokedUser)
	otherRevokedJWTToken, _ := s.AuthBackend.GetJWToken(otherRevokedTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.AuthBackend.IsBlacklisted(otherRevokedJWTToken))

	// Make 2 valid tokens for this user that won't be revoked
	_, validTokenString, err := s.AuthBackend.CreateToken(validUser)
	validJWTToken, _ := s.AuthBackend.GetJWToken(validTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.AuthBackend.IsBlacklisted(validJWTToken))

	_, otherValidTokenString, err := s.AuthBackend.CreateToken(validUser)
	otherValidJWTToken, _ := s.AuthBackend.GetJWToken(otherValidTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.AuthBackend.IsBlacklisted(otherValidJWTToken))

	err = s.AuthBackend.RevokeUserTokens(revokedUser)
	assert.Nil(s.T(), err)

	// This user's tokens are all blacklisted
	assert.True(s.T(), s.AuthBackend.IsBlacklisted(revokedJWTToken))
	assert.True(s.T(), s.AuthBackend.IsBlacklisted(otherRevokedJWTToken))

	// This user's tokens are all still valid
	assert.False(s.T(), s.AuthBackend.IsBlacklisted(validJWTToken))
	assert.False(s.T(), s.AuthBackend.IsBlacklisted(otherValidJWTToken))

}
func (s *BackendTestSuite) TestRevokeACOTokens() {
	revokedACOUUID, validACOUUID := "c14822fa-19ee-402c-9248-32af98419fe3", "82f55b6a-728e-4c8b-807e-535caad7b139"
	db := database.GetGORMDbConnection()

	// Get the ACO's
	var revokedACO, validACO models.ACO
	if db.First(&revokedACO, "UUID = ?", revokedACOUUID).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find ACO"))
	}
	if db.First(&validACO, "UUID = ?", validACOUUID).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find ACO"))
	}

	users := []models.User{}

	// Make a token for each user in the aco and then verify they have a valid token
	if db.Find(&users, "aco_id = ?", revokedACOUUID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("no users for revoked ACO"))
	}
	for _, user := range users {
		// Make sure we create a token for this user
		_, _, err := s.AuthBackend.CreateToken(user)
		assert.Nil(s.T(), err)
		tokens := []auth.Token{}
		db.Find(&tokens, "user_id = ? and active = ?", user.UUID, true)
		// Must have one or more tokens here
		numValidTokens := len(tokens)
		assert.True(s.T(), numValidTokens > 0)

	}

	// Do the same thing for the Valid ACO
	if db.Find(&users, "aco_id = ?", validACOUUID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("no users for valid ACO"))
	}
	for _, user := range users {
		// Make sure we create a token for this user
		_, _, err := s.AuthBackend.CreateToken(user)
		assert.Nil(s.T(), err)
		tokens := []auth.Token{}
		db.Find(&tokens, "user_id = ? and active = ?", user.UUID, true)
		// Must have one or more tokens here
		numValidTokens := len(tokens)
		assert.True(s.T(), numValidTokens > 0)

	}

	// Revoke the ACO tokens
	err := s.AuthBackend.RevokeACOTokens(revokedACO)
	assert.Nil(s.T(), err)
	// Find the users for the revoked ACO
	if db.Find(&users, "aco_id = ?", revokedACOUUID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("no users for revoked ACO"))
	}
	// Make sure none of them have a valid token
	for _, user := range users {
		tokens := []auth.Token{}
		db.Find(&tokens, "user_id = ? and active = ?", user.UUID, true)
		// Should be no valid tokens here
		numValidTokens := len(tokens)
		assert.True(s.T(), numValidTokens == 0)
		// Should be no errors
		assert.Nil(s.T(), db.Error)

	}

	// Find the users for the valid ACO
	if db.Find(&users, "aco_id = ?", validACOUUID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("no users for valid ACO"))
	}
	// Make sure none of them have a valid token
	for _, user := range users {
		tokens := []auth.Token{}
		db.Find(&tokens, "user_id = ? and active = ?", user.UUID, true)
		// Should be valid tokens here
		numValidTokens := len(tokens)
		assert.True(s.T(), numValidTokens > 0)
		// Should not be this kind of error
		assert.Nil(s.T(), db.Error)

	}
}

func (s *BackendTestSuite) TestIsBlacklisted() {
	userID := "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73"
	acoID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"

	db := database.GetGORMDbConnection()

	var aco models.ACO
	var user models.User
	// Bad test if we can't find the ACO
	if db.Find(&aco, "UUID = ?", acoID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to find ACO"))
	}
	// Bad test if we can't find the User or they are for a different aco
	if db.Find(&user, "UUID = ? AND aco_id = ?", userID, acoID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to find User"))
	}
	_, tokenString, err := s.AuthBackend.CreateToken(user)
	assert.Nil(s.T(), err)
	// Convert tokenString to a jwtToken
	jwtToken, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	// Test to see if this is blacklisted
	blacklisted := s.AuthBackend.IsBlacklisted(jwtToken)
	assert.False(s.T(), blacklisted)

	_ = s.AuthBackend.RevokeToken(tokenString)

	blacklisted = s.AuthBackend.IsBlacklisted(jwtToken)
	assert.Nil(s.T(), err)
	assert.True(s.T(), blacklisted)
}

func (s *BackendTestSuite) TestPrivateKey() {
	privateKey := s.AuthBackend.PrivateKey
	assert.NotNil(s.T(), privateKey)
	// get the real Key File location
	actualPrivateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	defer func() { os.Setenv("JWT_PRIVATE_KEY_FILE", actualPrivateKeyFile) }()

	// set the Private Key File to a bogus value to test negative scenarios
	// File does not exist
	os.Setenv("JWT_PRIVATE_KEY_FILE", "/static/thisDoesNotExist.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAuthBackend)

	// Empty file
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAuthBackend)

	// File contains not a key
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/badPrivate.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAuthBackend)

}

func (s *BackendTestSuite) TestPublicKey() {
	privateKey := s.AuthBackend.PublicKey
	assert.NotNil(s.T(), privateKey)
	// get the real Key File location
	actualPublicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	defer func() { os.Setenv("JWT_PUBLIC_KEY_FILE", actualPublicKeyFile) }()

	// set the Private Key File to a bogus value to test negative scenarios
	// File does not exist
	os.Setenv("JWT_PUBLIC_KEY_FILE", "/static/thisDoesNotExist.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAuthBackend)

	// Empty file
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAuthBackend)

	// File contains not a key
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/badPublic.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAuthBackend)

}

func (s *BackendTestSuite) TestCreateAlphaToken() {
	ttl := os.Getenv("")
	claims := checkStructure(s, ttl)
	checkTTL(s, claims, ttl)
}

func (s *BackendTestSuite) TestCreateAlphaTokenWithDefaultTTL() {
	ttl := os.Getenv("JWT_EXPIRATION_DELTA")
	claims := checkStructure(s, ttl)
	checkTTL(s, claims, ttl)
}

func (s *BackendTestSuite) TestCreateAlphaTokenWithCustomTTL() {
	const ttl = "720"
	claims := checkStructure(s, ttl)
	checkTTL(s, claims, ttl)
}

func checkTTL(s *BackendTestSuite, claims jwt.MapClaims, ttl string) {
	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.NotNil(s.T(), iat)
	assert.NotNil(s.T(), exp)

	// assumes the hard-coded value in auth/backend.go has not been overridden by an environment variable
	var delta = 72 * time.Hour

	if ttl != "" {
		var err error
		if delta, err = time.ParseDuration(ttl + "h"); err != nil {
			assert.Fail(s.T(), "Can't parse ttl value of %s", ttl)
		}
	}
	assert.True(s.T(), assert.WithinDuration(s.T(), iat, exp, delta, "expires date %s not within %s hours of issued at", exp.Format(time.RFC850), ttl))
}

func checkStructure(s *BackendTestSuite, ttl string) (jwt.MapClaims) {
	db := database.GetGORMDbConnection()
	tokenString, err := s.AuthBackend.CreateAlphaToken(ttl)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), tokenString)
	claims := s.AuthBackend.GetJWTClaims(tokenString)

	acoUUID := claims["aco"].(string)
	assert.NotNil(s.T(), acoUUID)
	var count int
	db.Table("beneficiaries").Where("aco_id = ?", acoUUID).Count(&count)
	assert.Equal(s.T(), 50, count)
	return claims
}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
