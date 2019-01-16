package auth_test

import (
	"crypto/rsa"
	"errors"
	"os"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type BackendTestSuite struct {
	testUtils.AuthTestSuite
	expectedSizes map[string]int
}

func (s *BackendTestSuite) SetupSuite() {
	models.InitializeGormModels()
	auth.InitializeGormModels()
}

func (s *BackendTestSuite) SetupTest() {
	s.SetupAuthBackend()
	s.expectedSizes = map[string]int{
		"dev":    50,
		"small":  10,
		"medium": 25,
		"large":  100,
	}
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
	t, tokenString, err := s.AuthBackend.CreateToken(user)
	assert.Nil(s.T(), err)
	// Convert tokenString to a jwtToken
	jwtToken, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	// Test to see if this is blacklisted
	blacklisted := s.AuthBackend.IsBlacklisted(jwtToken)
	assert.False(s.T(), blacklisted)

	t.Active = false
	db.Save(&t)

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

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
