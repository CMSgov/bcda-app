package plugin

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/jinzhu/gorm"

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

	token, _, _ := s.AuthBackend.CreateToken(user)

	assert := assert.New(s.T())

	err := s.p.RevokeClientCredentials([]byte(fmt.Sprintf(`{"clientID": "%s"}`, clientID)))
	assert.Nil(err)

	db.First(&token, "UUID = ?", token.UUID)
	assert.False(token.Active)

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
	db := database.GetGORMDbConnection()
	defer db.Close()

	userID, acoID := "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	var user models.User
	db.Find(&user, "UUID = ? AND aco_id = ?", userID, acoID)
	// Good Revoke test
	_, tokenString, _ := s.AuthBackend.CreateToken(user)

	assert := assert.New(s.T())

	err := s.p.RevokeAccessToken(userID)
	assert.NotNil(err)

	err = s.p.RevokeAccessToken(tokenString)
	assert.Nil(err)
	jwtToken, err := s.p.DecodeAccessToken(tokenString)
	assert.Nil(err)
	c, _ := jwtToken.Claims.(CustomClaims)
	var tokenFromDB jwt.Token
	assert.False(db.Find(&tokenFromDB, "UUID = ? AND active = false", c.ID).RecordNotFound())

	// Revoke the token again, you can't
	err = s.p.RevokeAccessToken(tokenString)
	assert.NotNil(err)

	// Revoke a token that doesn't exist
	tokenString, _ = s.AuthBackend.GenerateTokenString(uuid.NewRandom().String(), acoID)
	err = s.p.RevokeAccessToken(tokenString)
	assert.NotNil(err)
	assert.True(gorm.IsRecordNotFoundError(err))
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
