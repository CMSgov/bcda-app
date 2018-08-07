package auth_test

import (
	"crypto/rsa"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
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

func (s *BackendTestSuite) TestGenerateToken() {
	token, err := s.authBackend.GenerateToken(
		"82503A18-BF3B-436D-BA7B-BAE09B7FFD2F", "DBBD1CE1-AE24-435C-807D-ED45953077D3")

	// No errors, token is not nil
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
}

func (s *BackendTestSuite) TestIsBlacklisted() {}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
