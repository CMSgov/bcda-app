package auth

import (
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AlphaAuthPluginTestSuite struct {
	suite.Suite
	p *AlphaAuthPlugin
}

func (s *AlphaAuthPluginTestSuite) SetupTest() {
	s.p = new(AlphaAuthPlugin)
}

func (s *AlphaAuthPluginTestSuite) TestRegisterClient() {
	c, err := s.p.RegisterClient([]byte("{}"))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestUpdateClient() {
	c, err := s.p.UpdateClient([]byte("{}"))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDeleteClient() {
	err := s.p.DeleteClient([]byte("{}"))
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestGenerateClientCredentials() {
	r, err := s.p.GenerateClientCredentials([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestRevokeClientCredentials() {
	err := s.p.RevokeClientCredentials([]byte("{}"))
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestRequestAccessToken() {
	t, err := s.p.RequestAccessToken([]byte("{}"))
	assert.IsType(s.T(), jwt.Token{}, t)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestRevokeAccessToken() {
	err := s.p.RevokeAccessToken("")
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestValidateAccessToken() {
	err := s.p.ValidateAccessToken("")
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDecodeAccessToken() {
	t, err := s.p.DecodeAccessToken("")
	assert.IsType(s.T(), jwt.Token{}, t)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func TestAlphaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(AlphaAuthPluginTestSuite))
}
