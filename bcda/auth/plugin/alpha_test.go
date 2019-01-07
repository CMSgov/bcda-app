package plugin

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"

	"github.com/dgrijalva/jwt-go"
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

	// make sure we can't duplicate the ACO UUID
	aco := models.ACO{
		UUID: uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		Name: "Duplicate UUID Test",
	}
	err = database.GetGORMDbConnection().Save(&aco).Error
	assert.NotNil(s.T(), err)
}

func (s *AlphaAuthPluginTestSuite) TestUpdateClient() {
	c, err := s.p.UpdateClient([]byte(`{}`))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDeleteClient() {
	err := s.p.DeleteClient([]byte(`{}`))
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestGenerateClientCredentials() {
	r, err := s.p.GenerateClientCredentials([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestRevokeClientCredentials() {
<<<<<<< HEAD
	err := s.p.RevokeClientCredentials([]byte("{}"))
	assert.Equal(s.T(), "not yet implemented", err.Error())
=======
	c, err := s.p.RegisterClient([]byte(fmt.Sprintf(`{"clientID": "%s"}`, KnownFixtureACO)))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)

	err = s.p.RevokeClientCredentials([]byte(fmt.Sprintf(`{"clientID": "%s"}`, KnownFixtureACO)))
	// TODO: Update this test when RevokeAccessToken() is implemented
	assert.NotNil(s.T(), err)
>>>>>>> RevokeClientCredentials() in progress
}

func (s *AlphaAuthPluginTestSuite) TestRequestAccessToken() {
	t, err := s.p.RequestAccessToken([]byte(`{"clientID": "DBBD1CE1-AE24-435C-807D-ED45953077D3", "ttl": 720}`))
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), jwt.Token{}, t)

	t, err = s.p.RequestAccessToken([]byte(`{ "ttl": 720}`))
	assert.NotNil(s.T(), err)
	assert.IsType(s.T(), jwt.Token{}, t)
	assert.Contains(s.T(), err.Error(), "invalid string value")

	t, err = s.p.RequestAccessToken([]byte(`{"clientID": "DBBD1CE1-AE24-435C-807D-ED45953077D3"}`))
	assert.NotNil(s.T(), err)
	assert.IsType(s.T(), jwt.Token{}, t)
	assert.Contains(s.T(), err.Error(), "invalid int value")
}

func (s *AlphaAuthPluginTestSuite) TestRevokeAccessToken() {
	err := s.p.RevokeAccessToken("")
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestValidateAccessToken() {
	err := s.p.ValidateAccessToken("")
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDecodeAccessToken() {
	userID := uuid.NewRandom().String()
	acoID := uuid.NewRandom().String()
	ts, _ := auth.InitAuthBackend().GenerateTokenString(userID, acoID)
	t, err := s.p.DecodeAccessToken(ts)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), jwt.Token{}, t)
	assert.Equal(s.T(), userID, t.Claims.(*CustomClaims).Subject)
	assert.Equal(s.T(), acoID, t.Claims.(*CustomClaims).Aco)
}

func TestAlphaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(AlphaAuthPluginTestSuite))
}
