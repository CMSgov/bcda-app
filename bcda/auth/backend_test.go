package auth_test

import (
	"crypto/rsa"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
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
	assert.IsType(s.T(), &auth.AlphaBackend{}, s.AuthBackend)
	assert.IsType(s.T(), &rsa.PrivateKey{}, s.AuthBackend.PrivateKey)
	assert.IsType(s.T(), &rsa.PublicKey{}, s.AuthBackend.PublicKey)
}

func (s *BackendTestSuite) TestHashCompare() {
	uuidString := uuid.NewRandom().String()
	hash, err := auth.NewHash(uuidString)
	assert.Nil(s.T(), err)
	assert.True(s.T(), hash.IsHashOf(uuidString))
	assert.False(s.T(), hash.IsHashOf(uuid.NewRandom().String()))
}

func (s *BackendTestSuite) TestHashUnique() {
	uuidString := uuid.NewRandom().String()
	hash1, _ := auth.NewHash(uuidString)
	hash2, _ := auth.NewHash(uuidString)
	assert.NotEqual(s.T(), hash1.String(), hash2.String())
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
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// Empty file
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// File contains not a key
	os.Setenv("JWT_PRIVATE_KEY_FILE", "../static/badPrivate.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

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
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// Empty file
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/emptyFile.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

	// File contains not a key
	os.Setenv("JWT_PUBLIC_KEY_FILE", "../static/badPublic.pem")
	assert.Panics(s.T(), s.AuthBackend.ResetAlphaBackend)

}

func TestBackendTestSuite(t *testing.T) {
	suite.Run(t, new(BackendTestSuite))
}
