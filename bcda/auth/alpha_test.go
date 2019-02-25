package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	c, err := s.p.RegisterClient(KnownFixtureACO)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	assert.Equal(s.T(), KnownFixtureACO, c.ClientID)

	c, err = s.p.RegisterClient("")
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "provide a non-empty string")

	c, err = s.p.RegisterClient("correct length, but not a valid UUID")
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "valid UUID string")

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
	var aco = models.ACO{
		UUID:     acoID,
		Name:     "RevokeClientCredentials Test ACO",
		ClientID: acoID.String(),
	}
	db := connections["TestRevokeClientCredentials"]
	db.Save(&aco)

	var user = models.User{
		UUID:  uuid.NewRandom(),
		Name:  "RevokeClientCredentials Test User",
		Email: "revokeclientcredentialstest@example.com",
		ACO:   aco,
		ACOID: aco.UUID,
	}
	db.Save(&user)

	params := fmt.Sprintf(`{"clientID":"%s", "ttl":720}`, user.ACOID.String())
	_, err := s.p.GenerateClientCredentials([]byte(params))
	if err != nil {
		assert.FailNow(s.T(), fmt.Sprintf(`can't create client credentials for %s because %s`, user.ACOID.String(), err))
	}

	assert := assert.New(s.T())

	err = s.p.RevokeClientCredentials([]byte(fmt.Sprintf(`{"clientID": "%s"}`, aco.ClientID)))
	assert.Nil(err)

	var token auth.Token
	err = db.First(&token, "user_id = ?", user.UUID).Error
	require.Nil(s.T(), err)
	assert.False(token.Active)

	db.Delete(&token, &user, &aco)
}

func (s *AlphaAuthPluginTestSuite) TestRequestAccessToken() {
	t, err := s.p.RequestAccessToken([]byte(`{"clientID": "DBBD1CE1-AE24-435C-807D-ED45953077D3", "ttl": 720}`))
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), auth.Token{}, t)

	t, err = s.p.RequestAccessToken([]byte(`{ "ttl": 720}`))
	assert.NotNil(s.T(), err)
	assert.IsType(s.T(), auth.Token{}, t)
	assert.Contains(s.T(), err.Error(), "invalid string value")

	t, err = s.p.RequestAccessToken([]byte(`{"clientID": "DBBD1CE1-AE24-435C-807D-ED45953077D3"}`))
	assert.NotNil(s.T(), err)
	assert.IsType(s.T(), auth.Token{}, t)
	assert.Contains(s.T(), err.Error(), "invalid int value")
}

func (s *AlphaAuthPluginTestSuite) TestRevokeAccessToken() {
	db := connections["TestRevokeAccessToken"]

	const userID, acoID = "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73", "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	assert := assert.New(s.T())

	// Good Revoke test
	token, err := s.p.RequestAccessToken([]byte(fmt.Sprintf(`{"clientID": "%s", "ttl": 720}`, acoID)))
	if err != nil {
		assert.FailNow("no access token for %s because %s", acoID, err.Error())
	}

	err = s.p.RevokeAccessToken(userID)
	assert.NotNil(err)

	err = s.p.RevokeAccessToken(token.TokenString)
	assert.Nil(err)
	var tokenFromDB jwt.Token
	assert.False(db.Find(&tokenFromDB, "UUID = ? AND active = false", token.UUID).RecordNotFound())

	// Revoke the token again, you can't
	err = s.p.RevokeAccessToken(token.TokenString)
	assert.NotNil(err)

	// Revoke a token that doesn't exist
	tokenString, _ := s.AuthBackend.GenerateTokenString(uuid.NewRandom().String(), acoID)
	err = s.p.RevokeAccessToken(tokenString)
	assert.NotNil(err)
	assert.True(gorm.IsRecordNotFoundError(err))
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
	userID := uuid.NewRandom().String()
	acoID := uuid.NewRandom().String()
	ts, _ := s.AuthBackend.GenerateTokenString(userID, acoID)
	t, err := s.p.DecodeJWT(ts)
	c := t.Claims.(jwt.MapClaims)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), &jwt.Token{}, t)
	assert.Equal(s.T(), userID, c["sub"])
	assert.Equal(s.T(), acoID, c["aco"])
}

func TestAlphaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(AlphaAuthPluginTestSuite))
}
