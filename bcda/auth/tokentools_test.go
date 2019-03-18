package auth

import (
	"os"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TokenToolsTestSuite struct {
	suite.Suite
	originalEnvValue string
}

func (s *TokenToolsTestSuite) SetupSuite() {
	s.originalEnvValue = os.Getenv("JWT_EXPIRATION_DELTA")
}

func (s *TokenToolsTestSuite) TearDownSuite() {
	os.Setenv("JWT_EXPIRATION_DELTA", s.originalEnvValue)
}

func (s *TokenToolsTestSuite) AfterTest() {
	os.Setenv("JWT_EXPIRATION_DELTA", "60")
}

func (s *TokenToolsTestSuite) TestTokenDurationDefault() {
	assert.NotEmpty(s.T(), tokenTTL)
	assert.Equal(s.T(), tokenTTL, time.Hour)
}

func (s *TokenToolsTestSuite) TestTokenDurationOverride() {
	assert.NotEmpty(s.T(), tokenTTL)
	assert.Equal(s.T(), time.Hour, tokenTTL)
	os.Setenv("JWT_EXPIRATION_DELTA", "5")
	setTokenDuration()
	assert.Equal(s.T(), 5 * time.Minute, tokenTTL)
}

func (s *TokenToolsTestSuite) TestTokenDurationEmptyOverride() {
	assert.NotEmpty(s.T(), tokenTTL)
	assert.Equal(s.T(), time.Hour, tokenTTL)
	os.Setenv("JWT_EXPIRATION_DELTA", "")
	setTokenDuration()
	assert.Equal(s.T(), time.Hour, tokenTTL)
}

func (s *TokenToolsTestSuite) TestUnavailableSigner() {
	var (
		userUUID = "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
		acoUUID  = "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	)
	token, err := TokenStringWithIDs(uuid.NewRandom().String(), userUUID, acoUUID)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	// Wipe the keys
	alphaBackend.PrivateKey = nil
	alphaBackend.PublicKey = nil
	defer alphaBackend.ResetAlphaBackend()
	assert.Panics(s.T(), func() {
		_, _ = TokenStringWithIDs(uuid.NewRandom().String(), userUUID, acoUUID)
	})
}

func TestTokenToolsTestSuite(t *testing.T) {
	suite.Run(t, new(TokenToolsTestSuite))
}