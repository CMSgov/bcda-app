package auth

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

const KnownFixtureACO = "DBBD1CE1-AE24-435C-807D-ED45953077D3"
const KnownClientID = "0oajfkq1mc7O1fdrk0h7"					// not a valid Okta ID

type OktaAuthPluginTestSuite struct {
	suite.Suite
	o OktaAuthPlugin
	m *Mokta
}

func (s *OktaAuthPluginTestSuite) SetupSuite() {
	models.InitializeGormModels()
	InitializeGormModels()

	db := database.GetGORMDbConnection()
	defer func() {
		if err := db.Close(); err != nil {
			assert.Failf(s.T(), err.Error(), "okta plugin test")
		}
	}()

	var aco models.ACO
	if db.Find(&aco, "UUID = ?", uuid.Parse(KnownFixtureACO)).RecordNotFound() {
		assert.NotNil(s.T(), fmt.Errorf("Unable to find ACO %s", KnownFixtureACO))
		return
	}
	aco.ClientID = KnownClientID
	if err := db.Save(aco).Error; err != nil {
		assert.FailNow(s.T(), "Unable to update fixture ACO for Okta tests")
	}
}

func (s *OktaAuthPluginTestSuite) SetupTest() {
	s.m = NewMokta()
	s.o = NewOktaAuthPlugin(s.m)
}

func (s *OktaAuthPluginTestSuite) TestOktaRegisterClient() {
	c, err := s.o.RegisterClient(KnownFixtureACO)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	assert.Regexp(s.T(), regexp.MustCompile("[!-~]+"), c.ClientID)
}

func (s *OktaAuthPluginTestSuite) TestOktaUpdateClient() {
	c, err := s.o.UpdateClient([]byte("{}"))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestOktaDeleteClient() {
	err := s.o.DeleteClient([]byte("{}"))
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestOktaGenerateClientCredentials() {
	r, err := s.o.GenerateClientCredentials([]byte("{}"))
	assert.Nil(s.T(), r)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestOktaRevokeClientCredentials() {
	err := s.o.RevokeClientCredentials([]byte("{}"))
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestOktaRequestAccessToken() {
	t, err := s.o.RequestAccessToken([]byte("{}"))
	assert.IsType(s.T(), Token{}, t)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestOktaRevokeAccessToken() {
	err := s.o.RevokeAccessToken("")
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *OktaAuthPluginTestSuite) TestValidateJWT() {
	// happy path
	token, err := s.m.NewToken(KnownClientID)
	require.Nil(s.T(), err)
	err = s.o.ValidateJWT(token)
	require.Nil(s.T(), err)

	// a variety of unhappy paths
	token, err = s.m.NewCustomToken(OktaToken{ClientID:randomClientID()})
	require.Nil(s.T(), err)
	err = s.o.ValidateJWT(token)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid cid")

	token, err = s.m.NewCustomToken(OktaToken{ClientID:KnownClientID, Issuer: "not_our_okta_server"})
	require.Nil(s.T(), err)
	err = s.o.ValidateJWT(token)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "invalid iss")

	token, err = s.m.NewCustomToken(OktaToken{ClientID:KnownClientID, ExpiresIn: -1})
	require.Nil(s.T(), err)
	err = s.o.ValidateJWT(token)
	require.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "expired")
}

func (s *OktaAuthPluginTestSuite) TestOktaDecodeJWT() {
	token, err := s.m.NewToken(KnownClientID)
	require.Nil(s.T(), err, "could not create token")
	t, err := s.o.DecodeJWT(token)
	assert.IsType(s.T(), &jwt.Token{}, t)
	require.Nil(s.T(), err, "no error should have occurred")
	assert.True(s.T(), t.Valid)
}

func TestOktaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(OktaAuthPluginTestSuite))
}
