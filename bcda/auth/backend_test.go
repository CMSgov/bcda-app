package auth_test

import (
	"crypto/rsa"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	jwt "github.com/dgrijalva/jwt-go"
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
	userUUID, err := s.authBackend.CreateUser("First Last", "firstlast@example.com", uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"))

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), userUUID)
}

func (s *BackendTestSuite) TestGenerateToken() {
	token, err := s.authBackend.GenerateToken(
		"82503A18-BF3B-436D-BA7B-BAE09B7FFD2F", "DBBD1CE1-AE24-435C-807D-ED45953077D3")

	// No errors, token is not nil
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
}

func (s *BackendTestSuite) TestRevokeToken() {
	token, _ := s.authBackend.GenerateToken(
		"EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3")
	err := s.authBackend.RevokeToken(token)
	assert.Nil(s.T(), err)
}

func (s *BackendTestSuite) TestIsBlacklisted() {
	tokenString, _ := s.authBackend.GenerateToken(
		"EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3")

	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.authBackend.PublicKey, nil
	})

	blacklisted, err := s.authBackend.IsBlacklisted(token)
	assert.Nil(s.T(), err)
	assert.False(s.T(), blacklisted)

	_ = s.authBackend.RevokeToken(tokenString)

	blacklisted, err = s.authBackend.IsBlacklisted(token)
	assert.Nil(s.T(), err)
	assert.True(s.T(), blacklisted)
}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
