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

	c, err = s.p.RegisterClient([]byte(fmt.Sprintf(`{"clientID": "%s"}`, uuid.NewRandom().String())))
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
	// missing required param
	r, err := s.p.GenerateClientCredentials([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid string value")

	aco := models.ACO{
		UUID: uuid.NewRandom(),
		Name: "Gen Client Creds Test",
	}
	err = database.GetGORMDbConnection().Save(&aco).Error
	assert.Nil(s.T(), err, "wtf? %v", err)
	j := []byte(fmt.Sprintf(`{"clientID":"%s", "ttl":720}`, aco.UUID.String()))
	// we know that we use aco.UUID as the ClientID

	r, err = s.p.GenerateClientCredentials(j)
	assert.Nil(s.T(), r)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "have a registered client")

	// quick and dirty register client
	aco.ClientID = aco.UUID.String()
	err = database.GetGORMDbConnection().Save(&aco).Error
	assert.Nil(s.T(), err,"wtf? %v", err)
	user, err := models.CreateUser("Fake User", "fake@genclientcredstest.com", aco.UUID)
	assert.Nil(s.T(), err,"wtf? %v", err)

	r, err = s.p.GenerateClientCredentials(j)
	assert.NotNil(s.T(), r)
	assert.Nil(s.T(), err)

	database.GetGORMDbConnection().Delete(&user, &aco)
}

func (s *AlphaAuthPluginTestSuite) TestRevokeClientCredentials() {
	acoID := uuid.NewRandom()
	clientID := uuid.NewRandom().String()
	var aco = models.ACO{
		UUID:     acoID,
		Name:     "RevokeClientCredentials Test ACO",
		ClientID: clientID,
	}
	db := database.GetGORMDbConnection()
	defer db.Close()
	db.Save(&aco)

	var user = models.User{
		UUID:  uuid.NewRandom(),
		Name:  "RevokeClientCredentials Test User",
		Email: "revokeclientcredentialstest@example.com",
		Aco:   aco,
		AcoID: aco.UUID,
	}
	db.Save(&user)

	var token = auth.Token{
		UUID:   uuid.NewRandom(),
		User:   user,
		UserID: user.UUID,
		Active: true,
	}
	db.Save(&token)

	assert := assert.New(s.T())

	err := s.p.RevokeClientCredentials([]byte(fmt.Sprintf(`{"clientID": "%s"}`, clientID)))
	assert.NotNil(err)
	assert.Equal("1 of 1 token(s) could not be revoked due to errors", err.Error())
	// TODO: Update this test when RevokeAccessToken() is implemented
	//assert.False(token.Active)

	db.Delete(&token, &user, &aco)
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
