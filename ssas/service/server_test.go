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
	token, ts, err := s.server.MintAccessToken(acoUUID, nil)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)

	s.server.privateSigningKey = nil
	defer func() {
		_ = s.server.SetSigningKeys(unitSigningKeyPath)
	}()
	assert.Panics(s.T(), func() {
		_, _, _ = s.server.MintAccessToken(acoUUID, nil)
	})
}

func (s *ServerTestSuite) TestMintMFAToken() {
	err := s.server.SetSigningKeys(unitSigningKeyPath)
	assert.Nil(s.T(), err)
	token, ts, err := s.server.MintMFAToken("my_okta_id")

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)
}

func (s *ServerTestSuite) TestMintMFATokenMissingID() {
	token, ts, err := s.server.MintMFAToken("")

	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), token)
	assert.Equal(s.T(),"", ts)
}

func (s *ServerTestSuite) TestMintRegistrationToken() {
	groupIDs := []string{"A0000", "A0001"}
	token, ts, err := s.server.MintRegistrationToken("my_okta_id", groupIDs)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)
}

func (s *ServerTestSuite) TestMintRegistrationTokenMissingID() {
	groupIDs := []string{"", ""}
	token, ts, err := s.server.MintRegistrationToken("my_okta_id", groupIDs)

	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), token)
	assert.Equal(s.T(), "", ts)
}

func (s *ServerTestSuite) TestEmpty() {
	groupIDs := []string{"", ""}
	assert.True(s.T(), empty(groupIDs))

	groupIDs = []string{"", "asdf"}
	assert.False(s.T(), empty(groupIDs))
}


func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
