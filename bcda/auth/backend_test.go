package auth_test

import (
	"crypto/rsa"
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
		"82503A18-BF3B-436D-BA7B-BAE09B7FFD2F", "DBBD1CE1-AE24-435C-807D-ED45953077D3")
	err := s.authBackend.RevokeToken(token)
	assert.Nil(s.T(), err)
}

func (s *BackendTestSuite) TestIsBlacklisted() {}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
