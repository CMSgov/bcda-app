package auth

import (
	"testing"

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
	err := s.p.RegisterClient([]byte("{}"))
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestUpdateClient() {
	err := s.p.UpdateClient([]byte("{}"))
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
	r, err := s.p.RequestAccessToken([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestRevokeAccessToken() {
	err := s.p.RevokeAccessToken([]byte("{}"))
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestValidateAccessToken() {
	r, err := s.p.ValidateAccessToken([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDecodeAccessToken() {
	r, err := s.p.DecodeAccessToken([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func TestAlphaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(AlphaAuthPluginTestSuite))
}
