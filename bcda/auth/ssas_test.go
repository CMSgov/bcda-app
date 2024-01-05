package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	m "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	origSSASURL            string
	origPublicURL          string
	origSSASUseTLS         string
	origSSASClientKeyFile  string
	origSSASClientCertFile string
	origSSASClientID       string
	origSSASSecret         string
)

var testACOUUID = "dd0a19eb-c614-46c7-9ec0-95bbae959f37"

var sSasTokenErrorMsg string = "no token for SSAS; "
var sSasClientErrorMsg string = "no client for SSAS; "
var unexpectedErrorMsg string = "unexpected error; "

type SSASPluginTestSuite struct {
	suite.Suite
	p SSASPlugin

	db *sql.DB
	r  models.Repository
}

func (s *SSASPluginTestSuite) SetupSuite() {
	// original values must be saved before we run any tests that might change them
	origSSASUseTLS = conf.GetEnv("SSAS_USE_TLS")
	origSSASURL = conf.GetEnv("SSAS_URL")
	origPublicURL = conf.GetEnv("SSAS_PUBLIC_URL")
	origSSASClientKeyFile = conf.GetEnv("SSAS_CLIENT_KEY_FILE")
	origSSASClientCertFile = conf.GetEnv("SSAS_CLIENT_CERT_FILE")
	origSSASClientID = conf.GetEnv("BCDA_SSAS_CLIENT_ID")
	origSSASSecret = conf.GetEnv("BCDA_SSAS_SECRET")

	s.db = database.Connection
	s.r = postgres.NewRepository(s.db)
}

func (s *SSASPluginTestSuite) SetupTest() {
	postgrestest.CreateACO(s.T(), s.db,
		models.ACO{
			UUID:     uuid.Parse(testACOUUID),
			Name:     "SSAS Plugin Test ACO",
			ClientID: testACOUUID,
		})
}

func (s *SSASPluginTestSuite) TearDownTest() {
	conf.SetEnv(s.T(), "SSAS_USE_TLS", origSSASUseTLS)
	conf.SetEnv(s.T(), "SSAS_URL", origSSASURL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", origPublicURL)
	conf.SetEnv(s.T(), "SSAS_CLIENT_KEY_FILE", origSSASClientKeyFile)
	conf.SetEnv(s.T(), "SSAS_CLIENT_CERT_FILE", origSSASClientCertFile)
	conf.SetEnv(s.T(), "BCDA_SSAS_CLIENT_ID", origSSASClientID)
	conf.SetEnv(s.T(), "BCDA_SSAS_SECRET", origSSASSecret)

	postgrestest.DeleteACO(s.T(), s.db, uuid.Parse(testACOUUID))
}

func (s *SSASPluginTestSuite) TestRegisterSystem() {
	// These variables will allow us to swap out expectations without
	// reinstantiating the server
	var (
		response string
		ips      []string
		tester   *testing.T
	)

	// TODO: Mock client instead of server
	router := chi.NewRouter()
	router.Post("/system", func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := io.ReadAll(r.Body)
		assert.NoError(tester, err)
		var obj map[string]interface{}
		assert.NoError(tester, json.Unmarshal(reqBody, &obj))
		if obj["ips"] == nil {
			assert.Equal(tester, 0, len(ips), "ips should be empty since request contained no ips field")
		} else {
			var ipsReceived []string
			for _, ip := range obj["ips"].([]interface{}) {
				ipsReceived = append(ipsReceived, ip.(string))
			}
			assert.Equal(tester, ipsReceived, ips)
		}

		w.WriteHeader(201)
		fmt.Fprint(w, response)
	})
	server := httptest.NewServer(router)

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(constants.SsasClientErr, err.Error())
	}
	s.p = SSASPlugin{client: c, repository: s.r}

	validResp := `{ "system_id": "1", "client_id": ` + constants.FakeClientIDBt + `, "client_secret": ` + constants.FakeSecretBt + `, "client_name": "fake-name" }`
	tests := []struct {
		name      string
		ips       []string
		ssasResp  string
		expErrMsg string
	}{
		{"Successful response", nil, validResp, ""},
		{"Successful response with IPs", []string{testUtils.GetRandomIPV4Address(s.T()), testUtils.GetRandomIPV4Address(s.T())}, validResp, ""},
		{"Invalid JSON response", nil, `"this is": "invalid"`, "failed to unmarshal response json"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			ips, response, tester = tt.ips, tt.ssasResp, t
			creds, err := s.p.RegisterSystem(testACOUUID, "", "", tt.ips...)
			if tt.expErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to unmarshal response json")
				assert.Empty(t, creds.SystemID)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, "1", creds.SystemID)
			assert.Equal(t, constants.FakeClientID, creds.ClientID)
		})
	}
}

func (s *SSASPluginTestSuite) TestUpdateSystem() {
	// TestUpdateSystem is a test script that needs to showcase updating system group credentials/client name/api scope in the bcda ssas app then relay status to bcda app
}

func (s *SSASPluginTestSuite) TestDeleteSystem() {
	// TestDeleteSystem is a test script that need to showcase deleting system ips in the bcda ssas app then relay status to bcda app
}

func (s *SSASPluginTestSuite) TestResetSecret() {
	router := chi.NewRouter()
	router.Put("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, constants.FakeClientID, constants.FakeSecret)
	})
	server := httptest.NewServer(router)

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(constants.SsasClientErr, err.Error())
	}
	s.p = SSASPlugin{client: c, repository: s.r}

	creds, err := s.p.ResetSecret(testACOUUID)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), constants.FakeClientID, creds.ClientID)
	assert.Equal(s.T(), constants.FakeSecret, creds.ClientSecret)
}

func (s *SSASPluginTestSuite) TestRevokeSystemCredentials() {
	// TestRevokeSystemCredentials is a test script to showcase the ability to revoke client ids/credentials
}

func (s *SSASPluginTestSuite) TestMakeAccessToken() {
	_, tokenString, _, _ := MockSSASToken()
	MockSSASServer(tokenString)

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(constants.SsasClientErr, err.Error())
	}
	s.p = SSASPlugin{client: c, repository: s.r}

	tokenInfo, err := s.p.MakeAccessToken(Credentials{ClientID: "mock-client", ClientSecret: "mock-secret"})
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), string(tokenInfo))
	assert.Contains(s.T(), string(tokenInfo), constants.ExpiresInDefault)
	assert.Regexp(s.T(), regexp.MustCompile(`[^.\s]+\.[^.\s]+\.[^.\s]+`), string(tokenInfo))

	tokenInfo, err = s.p.MakeAccessToken(Credentials{ClientID: "sad", ClientSecret: "customer"})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), string(tokenInfo))
	assert.Contains(s.T(), err.Error(), "401")

	tokenInfo, err = s.p.MakeAccessToken(Credentials{})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), string(tokenInfo))
	assert.Contains(s.T(), err.Error(), "401")

	tokenInfo, err = s.p.MakeAccessToken(Credentials{ClientID: uuid.NewRandom().String()})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), string(tokenInfo))
	assert.Contains(s.T(), err.Error(), "401")

	tokenInfo, err = s.p.MakeAccessToken(Credentials{ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), string(tokenInfo))
	assert.Contains(s.T(), err.Error(), "401")

	tokenInfo, err = s.p.MakeAccessToken(Credentials{ClientID: uuid.NewRandom().String(), ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), string(tokenInfo))
	assert.Contains(s.T(), err.Error(), "401")
}

func (s *SSASPluginTestSuite) TestRevokeAccessToken() {
	router := chi.NewRouter()
	router.Delete("/token/{tokenID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(constants.SsasClientErr, err.Error())
	}
	s.p = SSASPlugin{client: c, repository: s.r}

	err = s.p.RevokeAccessToken("i.am.not.a.token")
	assert.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestAuthorizeAccessErrIsNilWhenHappyPath() {
	_, tokenString, _, err := MockSSASToken()
	require.NotNil(s.T(), tokenString, sSasTokenErrorMsg, err)
	require.Nil(s.T(), err, unexpectedErrorMsg, err)
	MockSSASServer(tokenString)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, sSasClientErrorMsg, err)
	s.p = SSASPlugin{client: c, repository: s.r}
	_, _, err = AuthorizeAccess(tokenString)
	require.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestAuthorizeAccessErrISReturnedWhenVerifyTokenCheckFails() {
	_, tokenString, _, err := MockSSASToken()
	require.NotNil(s.T(), tokenString, sSasTokenErrorMsg, err)
	require.Nil(s.T(), err, unexpectedErrorMsg, err)
	MockSSASServer(tokenString)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, sSasClientErrorMsg, err)
	s.p = SSASPlugin{client: c, repository: s.r}

	invalidTokenString := ""
	_, _, err = AuthorizeAccess(invalidTokenString)
	assert.EqualError(s.T(), err, "Requestor Data Error encountered - unable to parse provided tokenString to jwt.token. Err: token contains an invalid number of segments")
}

func (s *SSASPluginTestSuite) TestVerifyTokenErrorHandling() {
	_, goodTokenString, _, _ := MockSSASToken()

	nonSsasIssuerClaims := createCommonClaimsForTesting(constants.EmptyString, "random12") //Data setup as non-ssas to trigger error
	badNonSsasIssuerTokenString := createTokenStringFromCommonClaims(nonSsasIssuerClaims)

	emptyDataClaims := createCommonClaimsForTesting(constants.EmptyString, constants.IssuerSSAS) //Data setup as empty to trigger error
	badEmptyClaimsDataTokenString := createTokenStringFromCommonClaims(emptyDataClaims)

	tests := []struct {
		scenarioName         string
		tokenString          string
		server               *httptest.Server
		errTypeToReturn      error
		errMsgStringExpected string
	}{
		{"confirmTokenStringLegitimacy Fails due to parsing to jwt Token - return err", "abcd", testUtils.MakeTestServerWithIntrospectEndpoint(true), &customErrors.RequestorDataError{Msg: constants.EmptyString, Err: nil}, "unable to parse provided tokenString to jwt.token"},
		{"confirmTokenStringLegitimacy Fails due to issuer not SSAS - return err", badNonSsasIssuerTokenString, testUtils.MakeTestServerWithIntrospectEndpoint(true), &customErrors.RequestorDataError{Msg: constants.EmptyString, Err: nil}, "invalid issuer supplied in token CommonClaims"},
		{"confirmTokenStringLegitimacy Fails due to Claims Data Empty - return err", badEmptyClaimsDataTokenString, testUtils.MakeTestServerWithIntrospectEndpoint(true), &customErrors.RequestorDataError{Msg: constants.EmptyString, Err: nil}, "token CommonClaims Data is missing/empty"},
		{"CallSSASIntrospect Fails - return err", goodTokenString, testUtils.MakeTestServerWithIntrospectReturn502(), &customErrors.UnexpectedSSASError{Msg: constants.EmptyString, Err: nil}, "Unexpected SSAS Error encountered - Status code received in introspect response is 502"},
		{"CheckTokenExpiration Fails - return err", goodTokenString, testUtils.MakeTestServerWithIntrospectEndpoint(false), &customErrors.ExpiredTokenError{Msg: constants.EmptyString, Err: nil}, "the provided token has expired (is not active)"},
	}

	for _, tt := range tests {
		s.T().Run(tt.scenarioName, func(t *testing.T) {
			conf.SetEnv(t, "SSAS_URL", tt.server.URL)
			conf.SetEnv(t, "SSAS_PUBLIC_URL", tt.server.URL)

			c, err := client.NewSSASClient()
			require.NotNil(t, c, sSasClientErrorMsg, err)
			s.p = SSASPlugin{client: c, repository: s.r}

			_, err = s.p.VerifyToken(tt.tokenString)
			assert.IsType(t, tt.errTypeToReturn, err)
			assert.Contains(t, err.Error(), tt.errMsgStringExpected)
		})
	}
}

func createCommonClaimsForTesting(data string, issuer string) CommonClaims {
	return CommonClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer: issuer,
		},
		ClientID: uuid.New(),
		SystemID: uuid.New(),
		Data:     data,
	}
}

func createTokenStringFromCommonClaims(claims CommonClaims) (tokenString string) {
	t := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	pk, _ := rsa.GenerateKey(rand.Reader, 2048)
	tokenString, _ = t.SignedString(pk)
	return tokenString
}

func (s *SSASPluginTestSuite) TestAuthorizeAccessErrIsReturnedWhenGetAuthDataFromClaimsFails() {
	claims := createCommonClaimsForTesting("ac", constants.IssuerSSAS) //Data setup as bad string to trigger error
	ts := createTokenStringFromCommonClaims(claims)

	MockSSASServer(ts)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, sSasClientErrorMsg, err)
	s.p = SSASPlugin{client: c, repository: s.r}

	_, _, err = AuthorizeAccess(ts)
	assert.EqualError(s.T(), err, "can't decode data claim ac; invalid character 'a' looking for beginning of value")
}

func (s *SSASPluginTestSuite) TestGetAuthDataFromClaimsErrIsNilWhenHappyPath() {
	//setup data
	cmsID := testUtils.RandomHexID()[0:4]
	clientID := uuid.New()

	commonClaims := &CommonClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer: constants.IssuerSSAS,
		},
		ClientID: clientID,
		SystemID: uuid.New(),
		Data:     fmt.Sprintf(`{"cms_ids":["%s"]}`, cmsID),
	}

	aco := &models.ACO{Name: "ACO Test Name", CMSID: &cmsID, UUID: uuid.NewUUID(), ClientID: clientID, TerminationDetails: nil}

	//setup a mock repository (don't make actual repository call)
	mock := &models.MockRepository{}
	mock.On("GetACOByCMSID", m.MatchedBy(func(req context.Context) bool { return true }), cmsID).Return(aco, nil)
	models.SetMockRepository(s.T(), mock)

	//set the SSASPlugin to use the mock repository
	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, sSasClientErrorMsg, err)
	s.p = SSASPlugin{client: c, repository: mock}

	//assert no error
	_, err = s.p.getAuthDataFromClaims(commonClaims)
	require.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestGetAuthDataFromClaimsReturnEntityNotFoundErrorWhenErrFromGetACOByCMSID() {
	//setup data
	cmsID := testUtils.RandomHexID()[0:4]
	clientID := uuid.New()

	commonClaims := &CommonClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer: constants.IssuerSSAS,
		},
		ClientID: clientID,
		SystemID: uuid.New(),
		Data:     fmt.Sprintf(`{"cms_ids":["%s"]}`, cmsID),
	}

	//blank aco struct (returned for test)
	aco := &models.ACO{}

	//custom error expected
	dbErr := errors.New("DB Error: ACO Does Not Exist!")
	expectedErr := &customErrors.EntityNotFoundError{Err: dbErr, CMSID: cmsID}

	//setup a mock repository (don't make actual repository call)
	mock := &models.MockRepository{}
	mock.On("GetACOByCMSID", m.MatchedBy(func(req context.Context) bool { return true }), cmsID).Return(aco, expectedErr)
	models.SetMockRepository(s.T(), mock)

	//set the SSASPlugin to use the mock repository
	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, sSasClientErrorMsg, err)
	s.p = SSASPlugin{client: c, repository: mock}

	//assert no error and is of correct custom type
	_, err = s.p.getAuthDataFromClaims(commonClaims)
	require.NotNil(s.T(), err)
	assert.IsType(s.T(), &customErrors.EntityNotFoundError{}, err, "expected error of custom type EntityNotFoundError")
}

func (s *SSASPluginTestSuite) TestgetAuthDataFromClaimsReturnErrorWhenCommonClaimsDataBad() {
	//setup tests
	tests := []struct {
		name            string
		commonClaims    CommonClaims
		expectedMessage string
	}{
		{"CommonClaims Data has no quotes to remove (incomplete ssas token)", createCommonClaimsForTesting("", constants.IssuerSSAS), "incomplete ssas token"},
		{"CommonClaims Data is unable to be decoded", createCommonClaimsForTesting("abcdefg", constants.IssuerSSAS), "can't decode data claim abcdefg; invalid character 'a' looking for beginning of value"},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			//set the SSASPlugin
			c, err := client.NewSSASClient()
			require.NotNil(s.T(), c, sSasClientErrorMsg, err)
			s.p = SSASPlugin{client: c, repository: s.r}

			//Act
			_, err = s.p.getAuthDataFromClaims(&tt.commonClaims)

			//Assert
			require.NotNil(s.T(), err)
			assert.EqualError(s.T(), err, tt.expectedMessage, "expected error messaging")
		})
	}
}

func (s *SSASPluginTestSuite) TestVerifyToken() {
	_, tokenString, _, err := MockSSASToken()
	require.NotNil(s.T(), tokenString, sSasTokenErrorMsg, err)
	require.Nil(s.T(), err, unexpectedErrorMsg, err)
	MockSSASServer(tokenString)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, sSasClientErrorMsg)
	require.Nil(s.T(), err, unexpectedErrorMsg, err)
	s.p = SSASPlugin{client: c, repository: s.r}

	t, err := s.p.VerifyToken(tokenString)
	assert.NotEmpty(s.T(), t)
	assert.Nil(s.T(), err)
	assert.IsType(s.T(), &jwt.Token{}, t, "expected jwt token")
	assert.True(s.T(), t.Valid)
	tc := t.Claims.(*CommonClaims)
	assert.Equal(s.T(), "mock-system", tc.SystemID)
	assert.Equal(s.T(), "mock-id", tc.Id)
}

func TestSSASPluginSuite(t *testing.T) {
	suite.Run(t, new(SSASPluginTestSuite))
}

func MockSSASServer(tokenString string) {
	conf.SetEnv(&testing.T{}, "BCDA_SSAS_CLIENT_ID", "bcda")
	conf.SetEnv(&testing.T{}, "BCDA_SSAS_SECRET", "api")
	router := chi.NewRouter()
	router.Post("/introspect", func(w http.ResponseWriter, r *http.Request) {
		clientId, secret, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var answer = make(map[string]bool)
		if clientId == conf.GetEnv("BCDA_SSAS_CLIENT_ID") && secret == conf.GetEnv("BCDA_SSAS_SECRET") {
			answer["active"] = true
		} else {
			answer["active"] = false
		}
		render.JSON(w, r, answer)
	})
	router.Post("/token", func(w http.ResponseWriter, r *http.Request) {
		clientId, _, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if clientId == constants.MockClient {
			render.JSON(w, r, client.TokenResponse{AccessToken: tokenString, ExpiresIn: constants.ExpiresInDefault, TokenType: "access_token"})
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})

	server := httptest.NewServer(router)

	conf.SetEnv(&testing.T{}, "SSAS_URL", server.URL)
	conf.SetEnv(&testing.T{}, "SSAS_PUBLIC_URL", server.URL)
	conf.SetEnv(&testing.T{}, "SSAS_USE_TLS", "false")
}

func MockSSASToken() (*jwt.Token, string, string, error) {
	// NB: currently, BCDA expects only 1 item in the array of cms_ids. At some point, ACO-MS will want to send more than one
	claims := CommonClaims{
		SystemID: "mock-system",
		Data:     `{"cms_ids":["A9995"]}`,
		ClientID: constants.MockClient,
		StandardClaims: jwt.StandardClaims{
			Issuer:    constants.IssuerSSAS,
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Id:        "mock-id",
		},
	}

	t := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	pk, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, "", "", err
	}

	tokenString, err := t.SignedString(pk)
	return t, tokenString, constants.ExpiresInDefault, err
}
