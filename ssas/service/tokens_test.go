package service

import (
	"crypto/rsa"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	originalEnvValue string
	srvr *Server
}

func (s *ServerTestSuite) SetupSuite() {
	s.srvr = NewServer("test", "9999", "9.99.999", `{"foo" : "bar"}`, chi.NewRouter(), true)
	s.srvr.privateSigningKey = &rsa.PrivateKey{} // TODO real value here
}

func (s *ServerTestSuite) TestTokenDurationDefault() {
	assert.NotEmpty(s.T(), s.srvr.tokenTTL)
	assert.Equal(s.T(), s.srvr.tokenTTL, time.Hour)
}

func (s *ServerTestSuite) TestTokenDurationOverride() {
	originalValue := os.Getenv("SSAS_TOKEN_TTL_IN_MINUTES")
	assert.NotEmpty(s.T(), s.srvr.tokenTTL)
	assert.Equal(s.T(), time.Hour, s.srvr.tokenTTL)
	os.Setenv("SSAS_TOKEN_TTL_IN_MINUTES", "5")
	s.srvr.initTokenDuration()
	assert.Equal(s.T(), 5*time.Minute, s.srvr.tokenTTL)
	os.Setenv("SSAS_TOKEN_TTL_IN_MINUTES", originalValue)
}

func (s *ServerTestSuite) TestTokenDurationEmptyOverride() {
	assert.NotEmpty(s.T(), s.srvr.tokenTTL)
	assert.Equal(s.T(), time.Hour, s.srvr.tokenTTL)
	os.Setenv("JWT_EXPIRATION_DELTA", "")
	s.srvr.initTokenDuration()
	assert.Equal(s.T(), time.Hour, s.srvr.tokenTTL)
}

func (s *ServerTestSuite) TestUnavailableSigner() {
	acoUUID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	token, ts, err := s.srvr.MintToken(acoUUID)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	assert.NotNil(s.T(), ts)

	s.srvr.privateSigningKey = nil
	defer func() {
		s.srvr.privateSigningKey = &rsa.PrivateKey{} // TODO real value here
	}()
	assert.Panics(s.T(), func() {
		_, _, _ = s.srvr.MintToken(acoUUID)
	})
}

func TestTokenToolsTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
