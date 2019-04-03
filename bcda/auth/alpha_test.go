package auth_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type AlphaAuthPluginTestSuite struct {
	testUtils.AuthTestSuite
	p auth.AlphaAuthPlugin
}

func (s *AlphaAuthPluginTestSuite) SetupSuite() {
	models.InitializeGormModels()
	auth.InitializeGormModels()
	s.SetupAuthBackend()
}

func (s *AlphaAuthPluginTestSuite) SetupTest() {
	s.p = auth.AlphaAuthPlugin{}
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
	cmsID := testUtils.RandomHexID()[0:4]
	acoUUID, _ := models.CreateACO("TestRegisterClient", &cmsID)
	c, err := s.p.RegisterClient(acoUUID.String())
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	assert.NotEqual(s.T(), "", c.ClientSecret)
	assert.Equal(s.T(), acoUUID.String(), c.ClientID)
	var aco models.ACO
	aco.UUID = acoUUID
	connections["TestRegisterClient"].Find(&aco, "UUID = ?", acoUUID)
	assert.True(s.T(), auth.Hash(aco.AlphaSecret).IsHashOf(c.ClientSecret))
	defer connections["TestRegisterClient"].Delete(&aco)

	c, err = s.p.RegisterClient(acoUUID.String())
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "has a secret")

	c, err = s.p.RegisterClient("")
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "provide a non-empty string")

	c, err = s.p.RegisterClient(uuid.NewRandom().String())
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "no ACO record found")
}

func (s *AlphaAuthPluginTestSuite) TestUpdateClient() {
	c, err := s.p.UpdateClient([]byte(`{}`))
	assert.Nil(s.T(), c)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDeleteClient() {
	cmsID := testUtils.RandomHexID()[0:4]
	acoUUID, _ := models.CreateACO("TestRegisterClient", &cmsID)
	c, _ := s.p.RegisterClient(acoUUID.String())
	aco, _ := auth.GetACOByClientID(c.ClientID)
	assert.NotEmpty(s.T(), aco.ClientID)
	assert.NotEmpty(s.T(), aco.AlphaSecret)

	err := s.p.DeleteClient(c.ClientID)
	assert.Nil(s.T(), err)
	aco, _ = auth.GetACOByClientID(c.ClientID)
	assert.Empty(s.T(), aco.ClientID)
	assert.Empty(s.T(), aco.AlphaSecret)
}

func (s *AlphaAuthPluginTestSuite) TestGenerateClientCredentials() {
	r, err := s.p.GenerateClientCredentials("", 0)
	assert.Empty(s.T(), r)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not implemented")
}

func (s *AlphaAuthPluginTestSuite) TestAccessToken() {
	cmsID := testUtils.RandomHexID()[0:4]
	acoUUID, _ := models.CreateACO("TestAccessToken", &cmsID)
	user, _ := models.CreateUser("Test Access Token", "testaccesstoken@examplecom", acoUUID)
	cc, err := s.p.RegisterClient(acoUUID.String())
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), cc)
	ts, err := s.p.MakeAccessToken(auth.Credentials{ClientID: cc.ClientID, ClientSecret: cc.ClientSecret})
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), ts)
	assert.Regexp(s.T(), regexp.MustCompile(`[^.\s]+\.[^.\s]+\.[^.\s]+`), ts)
	connections["TestAccessToken"].Where("client_id = ?", cc.ClientID).Delete(&models.ACO{})
	connections["TestAccessToken"].Delete(&user)

	ts, err = s.p.MakeAccessToken(auth.Credentials{})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "missing or incomplete credentials")

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientID: uuid.NewRandom().String()})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "missing or incomplete credentials")

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "missing or incomplete credentials")

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientID: uuid.NewRandom().String(), ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "invalid credentials")
}

func (s *AlphaAuthPluginTestSuite) TestRequestAccessToken() {
	const userID, acoID = "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	t, err := s.p.RequestAccessToken(auth.Credentials{ClientID: acoID}, 720)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), auth.Token{}, t)
	assert.NotEmpty(s.T(), t.TokenString)

	t, err = s.p.RequestAccessToken(auth.Credentials{}, 720)
	assert.NotNil(s.T(), err)
	assert.IsType(s.T(), auth.Token{}, t)
	assert.Nil(s.T(), t.ACOID)
	assert.Contains(s.T(), err.Error(), "must provide either UserID or ClientID")

	t, err = s.p.RequestAccessToken(auth.Credentials{ClientID: acoID}, -1)
	assert.NotNil(s.T(), err)
	assert.IsType(s.T(), auth.Token{}, t)
	assert.Contains(s.T(), err.Error(), "invalid TTL")

	t, err = s.p.RequestAccessToken(auth.Credentials{UserID: userID}, 720)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), auth.Token{}, t)
	assert.NotEmpty(s.T(), t.TokenString)
	assert.NotNil(s.T(), t.ACOID)
}

func (s *AlphaAuthPluginTestSuite) TestRevokeAccessToken() {
	err := s.p.RevokeAccessToken("token-value-is-not-significant-here")
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not implemented")
}

func (s *AlphaAuthPluginTestSuite) TestValidateAccessToken() {
	userID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	acoID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	validClaims := jwt.MapClaims{
		"sub": userID,
		"aco": acoID,
		"id":  "d63205a8-d923-456b-a01b-0992fcb40968",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(999999999)).Unix(),
	}

	validToken := *jwt.New(jwt.SigningMethodRS512)
	validToken.Claims = validClaims
	validTokenString, _ := s.AuthBackend.SignJwtToken(validToken)
	err := s.p.ValidateJWT(validTokenString)
	assert.Nil(s.T(), err)

	unknownAco := *jwt.New(jwt.SigningMethodRS512)
	unknownAco.Claims = jwt.MapClaims{
		"sub": userID,
		"aco": uuid.NewRandom().String(),
		"id":  "d63205a8-d923-456b-a01b-0992fcb40968",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(999999999)).Unix(),
	}
	unknownAcoString, _ := s.AuthBackend.SignJwtToken(unknownAco)
	err = s.p.ValidateJWT(unknownAcoString)
	assert.Contains(s.T(), err.Error(), "no ACO record found")

	badSigningMethod := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"
	err = s.p.ValidateJWT(badSigningMethod)
	assert.Contains(s.T(), err.Error(), "unexpected signing method")

	wrongKey := "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.MejLezWY6hjGgbIXkq6Qbvx_-q5vWaTR6qPiNHphvla-XaZD3up1DN6Ib5AEOVtuB3fC9l-0L36noK4qQA79lhpSK3gozXO6XPIcCp4C8MU_ACzGtYe7IwGnnK3Emr6IHQE0bpGinHX1Ak1pAuwJNawaQ6Nvmz2ozZPsyxmiwoo"
	err = s.p.ValidateJWT(wrongKey)
	assert.Contains(s.T(), err.Error(), "crypto/rsa: verification error")

	missingClaims := *jwt.New(jwt.SigningMethodRS512)
	missingClaims.Claims = jwt.MapClaims{
		"sub": userID,
		"aco": acoID,
		"id":  "d63205a8-d923-456b-a01b-0992fcb40968",
	}
	missingClaimsString, _ := s.AuthBackend.SignJwtToken(missingClaims)
	err = s.p.ValidateJWT(missingClaimsString)
	assert.Contains(s.T(), err.Error(), "missing one or more required claims")

	noSuchTokenID := *jwt.New(jwt.SigningMethodRS512)
	noSuchTokenID.Claims = jwt.MapClaims{
		"sub": userID,
		"aco": acoID,
		"id":  uuid.NewRandom().String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(999999999)).Unix(),
	}
	noSuchTokenIDString, _ := s.AuthBackend.SignJwtToken(noSuchTokenID)
	err = s.p.ValidateJWT(noSuchTokenIDString)
	assert.Contains(s.T(), err.Error(), "is not active")

	invalidTokenID := *jwt.New(jwt.SigningMethodRS512)
	invalidTokenID.Claims = jwt.MapClaims{
		"sub": userID,
		"aco": acoID,
		"id":  uuid.NewRandom().String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(999999999)).Unix(),
	}
	invalidTokenIDString, _ := s.AuthBackend.SignJwtToken(invalidTokenID)
	err = s.p.ValidateJWT(invalidTokenIDString)
	assert.Contains(s.T(), err.Error(), "is not active")
}

func (s *AlphaAuthPluginTestSuite) TestDecodeJWT() {
	acoID := uuid.NewRandom().String()
	ts, _ := auth.TokenStringWithIDs(uuid.NewRandom().String(), acoID)
	t, err := s.p.DecodeJWT(ts)
	c := t.Claims.(*auth.CommonClaims)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), &jwt.Token{}, t)
	assert.Equal(s.T(), acoID, c.ACOID)
}

func TestAlphaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(AlphaAuthPluginTestSuite))
}
