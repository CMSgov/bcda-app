package auth

import (
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type OktaAuthPluginTestSuite struct {
	suite.Suite
	o *OktaAuthPlugin
}

func (s *OktaAuthPluginTestSuite) SetupTest() {
	s.o = new(OktaAuthPlugin)
}

func (s *OktaAuthPluginTestSuite) TestRegisterClient() {
	c, err := s.o.RegisterClient([]byte("{}"))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestUpdateClient() {
	c, err := s.o.UpdateClient([]byte("{}"))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestDeleteClient() {
	err := s.o.DeleteClient([]byte("{}"))
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestGenerateClientCredentials() {
	r, err := s.o.GenerateClientCredentials([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestRevokeClientCredentials() {
	err := s.o.RevokeClientCredentials([]byte("{}"))
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestRequestAccessToken() {
	t, err := s.o.RequestAccessToken([]byte("{}"))
	assert.IsType(s.T(), Token{}, t)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestRevokeAccessToken() {
	err := s.o.RevokeAccessToken("")
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestValidateJWT() {
	err := s.o.ValidateJWT("")
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestDecodeJWT() {
	t, err := s.o.DecodeJWT("")
	assert.IsType(s.T(), jwt.Token{}, t)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func TestOktaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(OktaAuthPluginTestSuite))
}
