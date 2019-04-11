package auth_test

import (
	"os"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type TokenToolsTestSuite struct {
	testUtils.AuthTestSuite
	originalEnvValue string
}

func (s *TokenToolsTestSuite) SetupSuite() {
	s.originalEnvValue = os.Getenv("JWT_EXPIRATION_DELTA")
	s.SetupAuthBackend()
}

func (s *TokenToolsTestSuite) TearDownSuite() {
	os.Setenv("JWT_EXPIRATION_DELTA", s.originalEnvValue)
}

func (s *TokenToolsTestSuite) AfterTest() {
	os.Setenv("JWT_EXPIRATION_DELTA", "60")
}

func (s *TokenToolsTestSuite) TestTokenDurationDefault() {
	assert.NotEmpty(s.T(), auth.TokenTTL)
	assert.Equal(s.T(), auth.TokenTTL, time.Hour)
}

func (s *TokenToolsTestSuite) TestTokenDurationOverride() {
	assert.NotEmpty(s.T(), auth.TokenTTL)
	assert.Equal(s.T(), time.Hour, auth.TokenTTL)
	os.Setenv("JWT_EXPIRATION_DELTA", "5")
	auth.SetTokenDuration()
	assert.Equal(s.T(), 5*time.Minute, auth.TokenTTL)
}

func (s *TokenToolsTestSuite) TestTokenDurationEmptyOverride() {
	assert.NotEmpty(s.T(), auth.TokenTTL)
	assert.Equal(s.T(), time.Hour, auth.TokenTTL)
	os.Setenv("JWT_EXPIRATION_DELTA", "")
	auth.SetTokenDuration()
	assert.Equal(s.T(), time.Hour, auth.TokenTTL)
}

func (s *TokenToolsTestSuite) TestUnavailableSigner() {
	acoUUID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	token, err := auth.TokenStringWithIDs(uuid.NewRandom().String(), acoUUID)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	// Wipe the keys
	s.AuthBackend.PublicKey = nil
	s.AuthBackend.PrivateKey = nil
	defer s.AuthBackend.ResetAlphaBackend()
	assert.Panics(s.T(), func() {
		_, _ = auth.TokenStringWithIDs(uuid.NewRandom().String(), acoUUID)
	})
}

func TestTokenToolsTestSuite(t *testing.T) {
	suite.Run(t, new(TokenToolsTestSuite))
}
