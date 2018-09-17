package auth_test

import (
	//"fmt"
	"crypto/rsa"
	"errors"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcdagorm"
	"github.com/jinzhu/gorm"

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
	s.SetupAuthBackend()
}

func (s *BackendTestSuite) TestInitAuthBackend() {
	assert.IsType(s.T(), &auth.JWTAuthenticationBackend{}, s.authBackend)
	assert.IsType(s.T(), &rsa.PrivateKey{}, s.authBackend.PrivateKey)
	assert.IsType(s.T(), &rsa.PublicKey{}, s.authBackend.PublicKey)
}

func (s *BackendTestSuite) TestCreateACO() {
	acoUUID, err := s.authBackend.CreateACO("ACO Name")

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), acoUUID)
}

func (s *BackendTestSuite) TestCreateUser() {
	name, email, sampleUUID := "First Last", "firstlast@exaple.com", "DBBD1CE1-AE24-435C-807D-ED45953077D3"

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
	duplicateUser, err := s.authBackend.CreateUser(name, email, uuid.NewRandom())
	assert.True(s.T(), duplicateUser.ID == 0)
	assert.NotNil(s.T(), err)
}

func (s *BackendTestSuite) TestGenerateToken() {
	userUUIDString, acoUUIDString := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	token, err := s.authBackend.GenerateTokenString(userUUIDString, acoUUIDString)

	// No errors, token is not nil
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
}

func (s *BackendTestSuite) TestCreateToken() {
	userID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	db := database.GetGORMDbConnection()
	var user bcdagorm.User
	if db.Find(&user, "UUID = ?", userID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to locate user"))
	}
	token, err := s.authBackend.CreateToken(user)
	assert.NotNil(s.T(), token.UUID)
	assert.Nil(s.T(), err)
}

func (s *BackendTestSuite) TestRevokeToken() {
	db := database.GetGORMDbConnection()
	userID, acoID := "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	var user bcdagorm.User
	db.Find(&user, "UUID = ? AND aco_id = ?", userID, acoID)
	// Good Revoke test
	token, _ := s.authBackend.CreateToken(user)
	err := s.authBackend.RevokeToken(token.Value)
	assert.Nil(s.T(), err)
	jwtToken, err := s.authBackend.GetJWTToken(token.Value)
	assert.Nil(s.T(), err)
	assert.True(s.T(), s.authBackend.IsBlacklisted(jwtToken))

	// Revoke the token again, you can't
	err = s.authBackend.RevokeToken(token.Value)
	assert.NotNil(s.T(), err)

	// Revoke a token that doesn't exist
	tokenString, _ := s.authBackend.GenerateTokenString(uuid.NewRandom().String(), acoID)
	err = s.authBackend.RevokeToken(tokenString)
	assert.NotNil(s.T(), err)
	assert.True(s.T(), gorm.IsRecordNotFoundError(err))

}

func (s *BackendTestSuite) TestRevokeUserTokens() {
	revokedEmail, validEmail := "userrevoked@email.com", "usernotrevoked@email.com"

	db := database.GetGORMDbConnection()
	// Get the User
	var revokedUser, validUser bcdagorm.User
	if db.First(&revokedUser, "Email = ?", revokedEmail).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find User"))
	}
	if db.First(&validUser, "Email = ?", validEmail).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find User"))
	}
	// make 2 valid tokens for this user to be revoked later
	revokedToken, err := s.authBackend.CreateToken(revokedUser)
	revokedJWTToken, _ := s.authBackend.GetJWTToken(revokedToken.Value)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.authBackend.IsBlacklisted(revokedJWTToken))

	otherRevokedToken, err := s.authBackend.CreateToken(revokedUser)
	otherRevokedJWTToken, _ := s.authBackend.GetJWTToken(otherRevokedToken.Value)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.authBackend.IsBlacklisted(otherRevokedJWTToken))

	// Make 2 valid tokens for this user that won't be revoked
	validToken, err := s.authBackend.CreateToken(validUser)
	validJWTToken, _ := s.authBackend.GetJWTToken(validToken.Value)
	assert.Nil(s.T(), err)
	assert.False(s.T(), s.authBackend.IsBlacklisted(validJWTToken))

	otherValidToken, err := s.authBackend.CreateToken(validUser)
	otherValidJWTToken, _ := s.authBackend.GetJWTToken(otherValidToken.Value)
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
	var revokedACO, validACO bcdagorm.ACO
	if db.First(&revokedACO, "UUID = ?", revokedACOUUID).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find ACO"))
	}
	if db.First(&validACO, "UUID = ?", validACOUUID).RecordNotFound() {
		// If this user doesn't exist the test has failed
		assert.NotNil(s.T(), errors.New("unable to find ACO"))
	}

	users := []bcdagorm.User{}

	// Make a token for each user in the aco and then verify they have a valid token
	if db.Find(&users, "aco_id = ?", revokedACOUUID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("no users for revoked ACO"))
	}
	for _, user := range users {
		// Make sure we create a token for this user
		_, err := s.authBackend.CreateToken(user)
		assert.Nil(s.T(), err)
		tokens := []bcdagorm.Token{}
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
		_, err := s.authBackend.CreateToken(user)
		assert.Nil(s.T(), err)
		tokens := []bcdagorm.Token{}
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
		tokens := []bcdagorm.Token{}
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
		tokens := []bcdagorm.Token{}
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

	var aco bcdagorm.ACO
	var user bcdagorm.User
	// Bad test if we can't find the ACO
	if db.Find(&aco, "UUID = ?", acoID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to find ACO"))
	}
	// Bad test if we can't find the User or they are for a different aco
	if db.Find(&user, "UUID = ? AND aco_id = ?", userID, acoID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to find User"))
	}
	token, err := s.authBackend.CreateToken(user)
	assert.Nil(s.T(), err)
	jwtToken, err := s.authBackend.GetJWTToken(token.Value)
	assert.Nil(s.T(), err)
	blacklisted := s.authBackend.IsBlacklisted(jwtToken)
	assert.False(s.T(), blacklisted)

	_ = s.authBackend.RevokeToken(token.Value)

	blacklisted = s.authBackend.IsBlacklisted(jwtToken)
	assert.True(s.T(), blacklisted)
}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
