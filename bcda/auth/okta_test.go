package auth_test

import (
	"regexp"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
)

const KnownFixtureACO = "DBBD1CE1-AE24-435C-807D-ED45953077D3"

type OktaAuthPluginTestSuite struct {
	suite.Suite
	o auth.OktaAuthPlugin
}

func (s *OktaAuthPluginTestSuite) SetupTest() {
	s.o = auth.OktaAuthPlugin{}
}

func (s *OktaAuthPluginTestSuite) TestRegisterClient() {
	c, err := s.o.RegisterClient(KnownFixtureACO)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	assert.Regexp(s.T(), regexp.MustCompile("[!-~]"), c.ClientID)
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
	r, err := s.o.GenerateClientCredentials("")
	assert.Equal(s.T(), auth.Credentials{}, r)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestRevokeClientCredentials() {
	err := s.o.RevokeClientCredentials([]byte("{}"))
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestRequestAccessToken() {
	t, err := s.o.RequestAccessToken([]byte("{}"))
	assert.IsType(s.T(), auth.Token{}, t)
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
