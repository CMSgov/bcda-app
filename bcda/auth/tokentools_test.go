package auth_test

import (
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
    configuration "github.com/CMSgov/bcda-app/config"
)

type TokenToolsTestSuite struct {
	suite.Suite
	originalEnvValue string
	abe              *auth.AlphaBackend
	reset            func()
}

func (s *TokenToolsTestSuite) SetupSuite() {
	s.originalEnvValue = configuration.GetEnv("JWT_EXPIRATION_DELTA")
	private := testUtils.SetAndRestoreEnvKey("JWT_PRIVATE_KEY_FILE", "../../shared_files/api_unit_test_auth_private.pem")
	public := testUtils.SetAndRestoreEnvKey("JWT_PUBLIC_KEY_FILE", "../../shared_files/api_unit_test_auth_public.pem")
	s.reset = func() {
		private()
		public()
	}
	s.abe = auth.InitAlphaBackend()
}

func (s *TokenToolsTestSuite) TearDownSuite() {
	configuration.SetEnv(&testing.T{}, "JWT_EXPIRATION_DELTA", s.originalEnvValue)
	s.reset()
}

func (s *TokenToolsTestSuite) AfterTest() {
	configuration.SetEnv(&testing.T{}, "JWT_EXPIRATION_DELTA", "60")
}

func (s *TokenToolsTestSuite) TestTokenDurationDefault() {
	assert.NotEmpty(s.T(), auth.TokenTTL)
	assert.Equal(s.T(), auth.TokenTTL, time.Hour)
}

func (s *TokenToolsTestSuite) TestTokenDurationOverride() {
	assert.NotEmpty(s.T(), auth.TokenTTL)
	assert.Equal(s.T(), time.Hour, auth.TokenTTL)
	configuration.SetEnv(&testing.T{}, "JWT_EXPIRATION_DELTA", "5")
	auth.SetTokenDuration()
	assert.Equal(s.T(), 5*time.Minute, auth.TokenTTL)
}

func (s *TokenToolsTestSuite) TestTokenDurationEmptyOverride() {
	assert.NotEmpty(s.T(), auth.TokenTTL)
	assert.Equal(s.T(), time.Hour, auth.TokenTTL)
	configuration.SetEnv(&testing.T{}, "JWT_EXPIRATION_DELTA", "")
	auth.SetTokenDuration()
	assert.Equal(s.T(), time.Hour, auth.TokenTTL)
}

func (s *TokenToolsTestSuite) TestUnavailableSigner() {
	acoUUID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	token, err := auth.TokenStringWithIDs(uuid.NewRandom().String(), acoUUID)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	// Wipe the keys
	s.abe.PublicKey = nil
	s.abe.PrivateKey = nil
	defer s.abe.ResetAlphaBackend()
	assert.Panics(s.T(), func() {
		_, _ = auth.TokenStringWithIDs(uuid.NewRandom().String(), acoUUID)
	})
}

func TestTokenToolsTestSuite(t *testing.T) {
	suite.Run(t, new(TokenToolsTestSuite))
}
