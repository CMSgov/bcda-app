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

var connections = make(map[string]*gorm.DB)

func (s *AlphaAuthPluginTestSuite) BeforeTest(suiteName, testName string) {
	connections[testName] = database.GetGORMDbConnection()
}

func (s *AlphaAuthPluginTestSuite) AfterTest(suiteName, testName string) {
	c, ok := connections[testName]
	if !ok {
		s.FailNow("WTF? no db connection for %s", testName)
	}
	if err := c.Close(); err != nil {
		s.FailNow("error closing db connection for %s because %s", testName, err)
	}
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
	// Warning: do not try to use s.T().Name() to lookup the connection
	err = connections["TestRegisterClient"].Save(&aco).Error
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
	err = connections["TestGenerateClientCredentials"].Save(&aco).Error
	assert.Nil(s.T(), err, "wtf? %v", err)
	j := []byte(fmt.Sprintf(`{"clientID":"%s", "ttl":720}`, aco.UUID.String()))
	// we know that we use aco.UUID as the ClientID

	r, err = s.p.GenerateClientCredentials(j)
	assert.Nil(s.T(), r)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "have a registered client")

	// quick and dirty register client
	aco.ClientID = aco.UUID.String()
	err = connections["TestGenerateClientCredentials"].Save(&aco).Error
	assert.Nil(s.T(), err, "wtf? %v", err)
	user, err := models.CreateUser("Fake User", "fake@genclientcredstest.com", aco.UUID)
	assert.Nil(s.T(), err, "wtf? %v", err)

	r, err = s.p.GenerateClientCredentials(j)
	assert.NotNil(s.T(), r)
	assert.Nil(s.T(), err)

	connections["TestGenerateClientCredentials"].Delete(&user, &aco)
}

func (s *AlphaAuthPluginTestSuite) TestRevokeClientCredentials() {
	acoID := uuid.NewRandom()
	clientID := uuid.NewRandom().String()
	var aco = models.ACO{
		UUID:     acoID,
		Name:     "RevokeClientCredentials Test ACO",
		ClientID: clientID,
	}
	db := connections["TestRevokeClientCredentials"]
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
	db := connections["TestRevokeAccessToken"]

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
