package auth_test

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AlphaAuthPluginTestSuite struct {
	suite.Suite
	p       auth.AlphaAuthPlugin
	backend *auth.AlphaBackend
	reset   func()

	db         *sql.DB
	repository models.Repository
}

func (s *AlphaAuthPluginTestSuite) SetupSuite() {
	private := testUtils.SetAndRestoreEnvKey("JWT_PRIVATE_KEY_FILE", "../../shared_files/api_unit_test_auth_private.pem")
	public := testUtils.SetAndRestoreEnvKey("JWT_PUBLIC_KEY_FILE", "../../shared_files/api_unit_test_auth_public.pem")
	s.reset = func() {
		private()
		public()
	}
	s.backend = auth.InitAlphaBackend()

	s.db = database.GetDbConnection()
	s.repository = postgres.NewRepository(s.db)
}

func (s *AlphaAuthPluginTestSuite) TearDownSuite() {
	s.reset()
	s.db.Close()
}

func (s *AlphaAuthPluginTestSuite) SetupTest() {
	s.p = auth.AlphaAuthPlugin{postgres.NewRepository(s.db)}
}

func (s *AlphaAuthPluginTestSuite) TestRegisterSystem() {
	cmsID := testUtils.RandomHexID()[0:4]
	acoUUID, _ := models.CreateACO("TestRegisterSystem", &cmsID)
	c, err := s.p.RegisterSystem(acoUUID.String(), "", "")
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), c)
	assert.NotEqual(s.T(), "", c.ClientSecret)
	assert.Equal(s.T(), acoUUID.String(), c.ClientID)
	aco, err := s.repository.GetACOByUUID(context.Background(), acoUUID)
	assert.NoError(s.T(), err)
	assert.True(s.T(), auth.Hash(aco.AlphaSecret).IsHashOf(c.ClientSecret))
	defer postgrestest.DeleteACO(s.T(), s.db, aco.UUID)

	c, err = s.p.RegisterSystem(acoUUID.String(), "", "")
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "has a secret")

	c, err = s.p.RegisterSystem("", "", "")
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "provide a non-empty string")

	c, err = s.p.RegisterSystem(uuid.NewRandom().String(), "", "")
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), c.ClientID)
	assert.Contains(s.T(), err.Error(), "no ACO record found")
}

func (s *AlphaAuthPluginTestSuite) TestUpdateSystem() {
	c, err := s.p.UpdateSystem([]byte(`{}`))
	assert.Nil(s.T(), c)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "not yet implemented", err.Error())
}

func (s *AlphaAuthPluginTestSuite) TestDeleteSystem() {
	cmsID := testUtils.RandomHexID()[0:4]
	acoUUID, _ := models.CreateACO("TestRegisterSystem", &cmsID)
	c, _ := s.p.RegisterSystem(acoUUID.String(), "", "")
	aco, _ := s.repository.GetACOByClientID(context.Background(), c.ClientID)
	assert.NotEmpty(s.T(), aco.ClientID)
	assert.NotEmpty(s.T(), aco.AlphaSecret)

	err := s.p.DeleteSystem(c.ClientID)
	assert.Nil(s.T(), err)
	aco, _ = s.repository.GetACOByClientID(context.Background(), c.ClientID)
	assert.Nil(s.T(), aco)
}

func (s *AlphaAuthPluginTestSuite) TestResetSecret() {
	validClientID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	c, err := s.p.ResetSecret(validClientID)
	assert.Nil(s.T(), err)
	assert.NotEqual(s.T(), "", c.ClientSecret)
	assert.Equal(s.T(), validClientID, c.ClientID)

	invalidClientID := "IDontexist"
	c, err = s.p.ResetSecret(invalidClientID)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", c.ClientSecret)
	assert.Equal(s.T(), "", c.ClientID)
}

func (s *AlphaAuthPluginTestSuite) TestAccessToken() {
	cmsID := testUtils.RandomHexID()[0:4]
	acoUUID, _ := models.CreateACO("TestAccessToken", &cmsID)
	cc, err := s.p.RegisterSystem(acoUUID.String(), "", "")
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), cc)
	ts, err := s.p.MakeAccessToken(auth.Credentials{ClientID: cc.ClientID, ClientSecret: cc.ClientSecret})
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), ts)
	assert.Regexp(s.T(), regexp.MustCompile(`[^.\s]+\.[^.\s]+\.[^.\s]+`), ts)
	t, _ := s.p.VerifyToken(ts)
	c := t.Claims.(*auth.CommonClaims)
	assert.True(s.T(), c.ExpiresAt <= time.Now().Unix()+3600)

	ts, err = s.p.MakeAccessToken(auth.Credentials{ClientID: cc.ClientID, ClientSecret: "not_the_right_secret"})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "invalid credentials")

	aco, err := s.repository.GetACOByClientID(context.Background(), cc.ClientID)
	assert.NoError(s.T(), err)
	postgrestest.DeleteACO(s.T(), s.db, aco.UUID)

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

func (s *AlphaAuthPluginTestSuite) TestRevokeAccessToken() {
	err := s.p.RevokeAccessToken("token-value-is-not-significant-here")
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not implemented")
}

func (s *AlphaAuthPluginTestSuite) TestValidateAccessToken() {
	acoID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	validClaims := jwt.MapClaims{
		"aco": acoID,
		"id":  "d63205a8-d923-456b-a01b-0992fcb40968",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(999999999)).Unix(),
	}

	validToken := jwt.New(jwt.SigningMethodRS512)
	validToken.Claims = validClaims
	validTokenString, _ := s.backend.SignJwtToken(validToken)
	err := s.p.AuthorizeAccess(validTokenString)
	assert.Nil(s.T(), err)

	unknownAco := jwt.New(jwt.SigningMethodRS512)
	unknownAco.Claims = jwt.MapClaims{
		"aco": uuid.NewRandom().String(),
		"id":  "d63205a8-d923-456b-a01b-0992fcb40968",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(999999999)).Unix(),
	}
	unknownAcoString, _ := s.backend.SignJwtToken(unknownAco)
	err = s.p.AuthorizeAccess(unknownAcoString)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "no ACO record found")

	badSigningMethod := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"
	err = s.p.AuthorizeAccess(badSigningMethod)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "unexpected signing method")

	wrongKey := "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.MejLezWY6hjGgbIXkq6Qbvx_-q5vWaTR6qPiNHphvla-XaZD3up1DN6Ib5AEOVtuB3fC9l-0L36noK4qQA79lhpSK3gozXO6XPIcCp4C8MU_ACzGtYe7IwGnnK3Emr6IHQE0bpGinHX1Ak1pAuwJNawaQ6Nvmz2ozZPsyxmiwoo"
	err = s.p.AuthorizeAccess(wrongKey)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "crypto/rsa: verification error")

	missingClaims := jwt.New(jwt.SigningMethodRS512)
	missingClaims.Claims = jwt.MapClaims{
		"aco": acoID,
		"id":  "d63205a8-d923-456b-a01b-0992fcb40968",
	}
	missingClaimsString, _ := s.backend.SignJwtToken(missingClaims)
	err = s.p.AuthorizeAccess(missingClaimsString)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "missing one or more required claims")

	expiredToken := jwt.New(jwt.SigningMethodRS512)
	expiredToken.Claims = jwt.MapClaims{
		"aco": acoID,
		"id":  uuid.NewRandom().String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Duration(-1) * time.Minute).Unix(),
	}
	expiredTokenString, _ := s.backend.SignJwtToken(expiredToken)
	err = s.p.AuthorizeAccess(expiredTokenString)
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "Token is expired")
}

func (s *AlphaAuthPluginTestSuite) TestVerifyToken() {
	acoID := uuid.NewRandom().String()
	ts, _ := auth.TokenStringWithIDs(uuid.NewRandom().String(), acoID)
	t, err := s.p.VerifyToken(ts)
	assert.NotNil(s.T(), t)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), &jwt.Token{}, t)
	c := t.Claims.(*auth.CommonClaims)
	assert.Equal(s.T(), acoID, c.ACOID)
}

func TestAlphaAuthPluginSuite(t *testing.T) {
	suite.Run(t, new(AlphaAuthPluginTestSuite))
}
