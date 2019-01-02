package auth

import (
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/models"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const KnownFixtureACO = "DBBD1CE1-AE24-435C-807D-ED45953077D3"

type AlphaAuthPluginTestSuite struct {
	testUtils.AuthTestSuite
	p *AlphaAuthPlugin
}

func (s *AlphaAuthPluginTestSuite) SetupSuite() {
	models.InitializeGormModels()
	auth.InitializeGormModels()
	s.SetupAuthBackend()
}

func (s *AlphaAuthPluginTestSuite) SetupTest() {
	s.p = new(AlphaAuthPlugin)
}

func (s *AlphaAuthPluginTestSuite) TestRegisterClient() {
	c, err := s.p.RegisterClient([]byte(`{"clientID": "DBBD1CE1-AE24-435C-807D-ED45953077D3"}`))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	var result map[string]interface{}
	if err = json.Unmarshal(c, &result); err != nil {
		assert.Fail(s.T(), "Bad json result value")
	}
	assert.Equal(s.T(), KnownFixtureACO, result["clientID"].(string))

	c, err = s.p.RegisterClient([]byte(`{"clientID":""}`))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), c)
	assert.Contains(s.T(), err.Error(), "provide a non-empty string")

	c, err = s.p.RegisterClient([]byte(`{"clientID": "correct length, but not a valid UUID"}`))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), c)
	assert.Contains(s.T(), err.Error(), "valid UUID string")

	c, err = s.p.RegisterClient([]byte(fmt.Sprintf("{\"clientID\": \"%s\"}", uuid.NewRandom().String())))
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), c)
}

func (s *AlphaAuthPluginTestSuite) TestUpdateClient() {
	c, err := s.p.UpdateClient([]byte(`{}`))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "Not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDeleteClient() {
	err := s.p.DeleteClient([]byte(`{}`))
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
	t, err := s.p.RequestAccessToken([]byte(`{"clientID": "DBBD1CE1-AE24-435C-807D-ED45953077D3", "ttl": 720}`))
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), jwt.Token{}, t)
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
