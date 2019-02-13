package auth_test

import (
	"regexp"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
)

const KnownFixtureACO = "DBBD1CE1-AE24-435C-807D-ED45953077D3"

type OktaAuthPluginTestSuite struct {
	suite.Suite
	o auth.OktaAuthPlugin
	m *auth.Mokta
}

func (s *OktaAuthPluginTestSuite) SetupTest() {
	s.m = auth.NewMokta()
	s.o = auth.NewOktaAuthPlugin(s.m)
}

func (s *OktaAuthPluginTestSuite) TestRegisterClient() {
	c, err := s.o.RegisterClient(KnownFixtureACO)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	assert.Regexp(s.T(), regexp.MustCompile("[!-~]+"), c.ClientID)
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
	token, err := s.m.NewToken("12345", "54321", 3600)
	require.Nil(s.T(), err, "could not create token")
	t, err := s.o.DecodeJWT(token)
	assert.IsType(s.T(), &jwt.Token{}, t)
	require.Nil(s.T(), err, "no error should have occurred")
	assert.True(s.T(), t.Valid)
}

func TestOktaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(OktaAuthPluginTestSuite))
}
