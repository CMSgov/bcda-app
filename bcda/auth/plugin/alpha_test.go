package auth

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const KnownFixtureACO = "DBBD1CE1-AE24-435C-807D-ED45953077D3"

type AlphaAuthPluginTestSuite struct {
	suite.Suite
	p *AlphaAuthPlugin
}

func (s *AlphaAuthPluginTestSuite) SetupTest() {
	s.p = new(AlphaAuthPlugin)
	models.InitializeGormModels()
}

func (s *AlphaAuthPluginTestSuite) TestRegisterClient() {
	c, err := s.p.RegisterClient([]byte(KnownFixtureACO))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	assert.Equal(s.T(), KnownFixtureACO, string(c))

	c, err = s.p.RegisterClient([]byte(""))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), c)
	assert.Contains(s.T(), err.Error(), "provide a non-empty string")

	c, err = s.p.RegisterClient([]byte("Not a valid UUID"))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), c)
	assert.Contains(s.T(), err.Error(), "valid UUID string")

	c, err = s.p.RegisterClient([]byte(uuid.NewRandom()))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), c)
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
