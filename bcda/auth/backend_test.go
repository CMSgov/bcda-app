package auth_test

import (
	//"fmt"
	"crypto/rsa"
	"errors"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"os"
	//"strings"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BackendTestSuite struct {
	AuthTestSuite
}

func (s *BackendTestSuite) SetupTest() {
	auth.InitializeGormModels()
	s.SetupAuthBackend()
}

func (s *BackendTestSuite) TestInitAuthBackend() {
	assert.IsType(s.T(), &auth.JWTAuthenticationBackend{}, s.authBackend)
	assert.IsType(s.T(), &rsa.PrivateKey{}, s.authBackend.PrivateKey)
	assert.IsType(s.T(), &rsa.PublicKey{}, s.authBackend.PublicKey)
}

func (s *BackendTestSuite) TestHashCompare() {
	uuidString := uuid.NewRandom().String()
	hash := auth.Hash{}
	hashString := hash.Generate(uuidString)
	assert.True(s.T(), hash.Compare(hashString, uuidString))
	assert.False(s.T(), hash.Compare(hashString, uuid.NewRandom().String()))
}

func (s *BackendTestSuite) TestCreateACO() {
	acoUUID, err := s.authBackend.CreateACO("ACO Name")

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), acoUUID)
}

func (s *BackendTestSuite) TestCreateUser() {
	name, email, sampleUUID, duplicateName := "First Last", "firstlast@exaple.com", "DBBD1CE1-AE24-435C-807D-ED45953077D3", "Duplicate Name"

	// Make a user for an ACO that doesn't exist
	badACOUser, err := s.authBackend.CreateUser(name, email, uuid.NewRandom())
	//No ID because it wasn't saved
	assert.True(s.T(), badACOUser.ID == 0)
	// Should get an error
	assert.NotNil(s.T(), err)

	// Make a good user
	user, err := s.authBackend.CreateUser(name, email, uuid.Parse(sampleUUID))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), user.UUID)
	assert.NotNil(s.T(), user.ID)

	// Try making a duplicate user for the same E-mail address
	duplicateUser, err := s.authBackend.CreateUser(duplicateName, email, uuid.Parse(sampleUUID))
	// Got a user, not the one that was requested
	assert.True(s.T(), duplicateUser.Name == name)
	assert.NotNil(s.T(), err)
}

func (s *BackendTestSuite) TestGenerateToken() {
	userUUIDString, acoUUIDString := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	token, err := s.authBackend.GenerateTokenString(userUUIDString, acoUUIDString)

	// No errors, token is not nil
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	// Wipe the keys
	s.authBackend.PrivateKey = nil
	s.authBackend.PublicKey = nil
	defer s.authBackend.ResetAuthBackend()
	assert.Panics(s.T(), func() { _, _ = s.authBackend.GenerateTokenString(userUUIDString, acoUUIDString) })
}

func (s *BackendTestSuite) TestCreateToken() {
	userID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	db := database.GetGORMDbConnection()
	var user auth.User
	if db.Find(&user, "UUID = ?", userID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to locate user"))
	}
	token, _, err := s.authBackend.CreateToken(user)
	assert.NotNil(s.T(), token.UUID)
	assert.Nil(s.T(), err)

	// Wipe the keys
	s.authBackend.PrivateKey = nil
	s.authBackend.PublicKey = nil
	defer s.authBackend.ResetAuthBackend()
	assert.Panics(s.T(), func() { _, _, _ = s.authBackend.CreateToken(user) })
}

func (s *BackendTestSuite) TestGetJWClaims() {
	acoID, userID := uuid.NewRandom().String(), uuid.NewRandom().String()
	goodToken, err := s.authBackend.GenerateTokenString(acoID, userID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), s.authBackend.GetJWTClaims(goodToken))

	// Check an expired token
	expiredToken := jwt.New(jwt.SigningMethodRS512)
	expiredToken.Claims = jwt.MapClaims{
		"exp": 12345,
		"iat": 123,
		"sub": userID,
		"aco": acoID,
		"id":  uuid.NewRandom(),
	}
	expiredTokenString, err := expiredToken.SignedString(s.authBackend.PrivateKey)
	assert.Nil(s.T(), err)
	invalidClaims := s.authBackend.GetJWTClaims(expiredTokenString)
	assert.Nil(s.T(), invalidClaims)

	// Check an incorrectly signed token.
	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"
	badClaims := s.authBackend.GetJWTClaims(badToken)
	assert.Nil(s.T(), badClaims)
}

func (s *BackendTestSuite) TestRevokeToken() {
	db := database.GetGORMDbConnection()
	userID, acoID := "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	var user auth.User
	db.Find(&user, "UUID = ? AND aco_id = ?", userID, acoID)
	// Good Revoke test
	_, tokenString, _ := s.authBackend.CreateToken(user)

	err := s.authBackend.RevokeToken(userID)
	assert.NotNil(s.T(), err)
	err = s.authBackend.RevokeToken(tokenString)
	assert.Nil(s.T(), err)
	jwtToken, err := s.authBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	assert.True(s.T(), s.authBackend.IsBlacklisted(jwtToken))

	// Revoke the token again, you can't
	err = s.authBackend.RevokeToken(tokenString)
	assert.NotNil(s.T(), err)

	// Revoke a token that doesn't exist
	tokenString, _ = s.authBackend.GenerateTokenString(uuid.NewRandom().String(), acoID)
	err = s.authBackend.RevokeToken(tokenString)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), gorm.IsRecordNotFoundError(err))

}

func (s *BackendTestSuite) TestRevokeUserTokens() {
	revokedEmail, validEmail := "userrevoked@email.com", "usernotrevoked@email.com"

	db := database.GetGORMDbConnection()
	// Get the User
	var revokedUser, validUser auth.User
	if db.First(&revokedUser, "Email = ?", revokedEmail).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find User"))
	}
	if db.First(&validUser, "Email = ?", validEmail).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find User"))
	}
	// make 2 valid tokens for this user to be revoked later
	_, revokedTokenString, err := s.authBackend.CreateToken(revokedUser)
	revokedJWTToken, _ := s.authBackend.GetJWToken(revokedTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.authBackend.IsBlacklisted(revokedJWTToken))

	_, otherRevokedTokenString, err := s.authBackend.CreateToken(revokedUser)
	otherRevokedJWTToken, _ := s.authBackend.GetJWToken(otherRevokedTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.authBackend.IsBlacklisted(otherRevokedJWTToken))

	// Make 2 valid tokens for this user that won't be revoked
	_, validTokenString, err := s.authBackend.CreateToken(validUser)
	validJWTToken, _ := s.authBackend.GetJWToken(validTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.authBackend.IsBlacklisted(validJWTToken))

	_, otherValidTokenString, err := s.authBackend.CreateToken(validUser)
	otherValidJWTToken, _ := s.authBackend.GetJWToken(otherValidTokenString)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.authBackend.IsBlacklisted(otherValidJWTToken))

	err = s.authBackend.RevokeUserTokens(revokedUser)
	assert.Nil(s.T(), err)

	// This user's tokens are all blacklisted
	assert.True(s.T(), s.authBackend.IsBlacklisted(revokedJWTToken))
	assert.True(s.T(), s.authBackend.IsBlacklisted(otherRevokedJWTToken))

	// This user's tokens are all still valid
	assert.False(s.T(), s.authBackend.IsBlacklisted(validJWTToken))
	assert.False(s.T(), s.authBackend.IsBlacklisted(otherValidJWTToken))

}
func (s *BackendTestSuite) TestRevokeACOTokens() {
	revokedACOUUID, validACOUUID := "c14822fa-19ee-402c-9248-32af98419fe3", "82f55b6a-728e-4c8b-807e-535caad7b139"
	db := database.GetGORMDbConnection()

	// Get the ACO's
	var revokedACO, validACO auth.ACO
	if db.First(&revokedACO, "UUID = ?", revokedACOUUID).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find ACO"))
	}
	if db.First(&validACO, "UUID = ?", validACOUUID).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find ACO"))
	}

	users := []auth.User{}

	// Make a token for each user in the aco and then verify they have a valid token
	if db.Find(&users, "aco_id = ?", revokedACOUUID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("no users for revoked ACO"))
	}
	for _, user := range users {
		// Make sure we create a token for this user
		_, _, err := s.authBackend.CreateToken(user)
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
		_, _, err := s.authBackend.CreateToken(user)
		assert.Nil(s.T(), err)
		tokens := []auth.Token{}
		db.Find(&tokens, "user_id = ? and active = ?", user.UUID, true)
		// Must have one or more tokens here
		numValidTokens := len(tokens)
		assert.True(s.T(), numValidTokens > 0)

	}

	// Revoke the ACO tokens
	err := s.authBackend.RevokeACOTokens(revokedACO)
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

	var aco auth.ACO
	var user auth.User
	// Bad test if we can't find the ACO
	if db.Find(&aco, "UUID = ?", acoID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to find ACO"))
	}
	// Bad test if we can't find the User or they are for a different aco
	if db.Find(&user, "UUID = ? AND aco_id = ?", userID, acoID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to find User"))
	}
	_, tokenString, err := s.authBackend.CreateToken(user)
	assert.Nil(s.T(), err)
	// Convert tokenString to a jwtToken
	jwtToken, err := s.authBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	// Test to see if this is blacklisted
	blacklisted := s.authBackend.IsBlacklisted(jwtToken)
	assert.False(s.T(), blacklisted)

	_ = s.authBackend.RevokeToken(tokenString)

	blacklisted = s.authBackend.IsBlacklisted(jwtToken)
	assert.Nil(s.T(), err)
	assert.True(s.T(), blacklisted)
}

func (s *BackendTestSuite) TestPrivateKey() {
	privateKey := s.authBackend.PrivateKey
	assert.NotNil(s.T(), privateKey)
	// get the real Key File location
	actualPrivateKeyFile := os.Getenv("JWT_PRIVATE_KEY_FILE")
	defer func() { os.Setenv("JWT_PRIVATE_KEY_FILE", actualPrivateKeyFile) }()

	// set the Private Key File to a bogus value to test negative scenarios
	// File does not exist
	os.Setenv("JWT_PRIVATE_KEY_FILE", "/static/thisDoesNotExist.pem")
	assert.Panics(s.T(), s.authBackend.ResetAuthBackend)

	// Empty file
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.authBackend.ResetAuthBackend)

	// File contains not a key
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/badPrivate.pem")
	assert.Panics(s.T(), s.authBackend.ResetAuthBackend)

}

func (s *BackendTestSuite) TestPublicKey() {
	privateKey := s.authBackend.PublicKey
	assert.NotNil(s.T(), privateKey)
	// get the real Key File location
	actualPublicKeyFile := os.Getenv("JWT_PUBLIC_KEY_FILE")
	defer func() { os.Setenv("JWT_PUBLIC_KEY_FILE", actualPublicKeyFile) }()

	// set the Private Key File to a bogus value to test negative scenarios
	// File does not exist
	os.Setenv("JWT_PUBLIC_KEY_FILE", "/static/thisDoesNotExist.pem")
	assert.Panics(s.T(), s.authBackend.ResetAuthBackend)

	// Empty file
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.authBackend.ResetAuthBackend)

	// File contains not a key
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/badPublic.pem")
	assert.Panics(s.T(), s.authBackend.ResetAuthBackend)

}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
