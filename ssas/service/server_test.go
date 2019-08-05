package service

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const unitSigningKeyPath string = "../../shared_files/ssas/unit_test_private_key.pem"

type ServerTestSuite struct {
	suite.Suite
	server *Server
}

func (s *ServerTestSuite) SetupSuite() {
	info := make(map[string][]string)
	info["public"] = []string{"token", "register"}
	s.server = NewServer("test-server", ":9999", "9.99.999", info, s.server.newBaseRouter(), true)
}

func (s *ServerTestSuite) TestSetSigningKeys() {
	err := s.server.SetSigningKeys("")
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "bad signing key", "testing bad key")

	err = s.server.SetSigningKeys(unitSigningKeyPath)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), s.server.privateSigningKey)
}

func (s *ServerTestSuite) TestTokenDurationDefault() {
	assert.NotEmpty(s.T(), s.server.tokenTTL)
	assert.Equal(s.T(), s.server.tokenTTL, time.Hour)
}

func (s *ServerTestSuite) TestTokenDurationOverride() {
	originalValue := os.Getenv("SSAS_TOKEN_TTL_IN_MINUTES")
	assert.NotEmpty(s.T(), s.server.tokenTTL)
	assert.Equal(s.T(), time.Hour, s.server.tokenTTL)
	os.Setenv("SSAS_TOKEN_TTL_IN_MINUTES", "5")
	s.server.initTokenDuration()
	assert.Equal(s.T(), 5*time.Minute, s.server.tokenTTL)
	os.Setenv("SSAS_TOKEN_TTL_IN_MINUTES", originalValue)
}

func (s *ServerTestSuite) TestTokenDurationEmptyOverride() {
	assert.NotEmpty(s.T(), s.server.tokenTTL)
	assert.Equal(s.T(), time.Hour, s.server.tokenTTL)
	os.Setenv("JWT_EXPIRATION_DELTA", "")
	s.server.initTokenDuration()
	assert.Equal(s.T(), time.Hour, s.server.tokenTTL)
}

func (s *ServerTestSuite) TestUnavailableSigner() {
	acoUUID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	token, ts, err := s.server.MintToken(CommonClaims{ACOID: acoUUID})

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)

	s.server.privateSigningKey = nil
	defer func() {
		_ = s.server.SetSigningKeys(unitSigningKeyPath)
	}()
	assert.Panics(s.T(), func() {
		_, _, _ = s.server.MintToken(CommonClaims{ACOID: acoUUID})
	})
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
