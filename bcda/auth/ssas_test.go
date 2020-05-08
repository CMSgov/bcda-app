package auth

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
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

type SSASPluginTestSuite struct {
	suite.Suite
	p SSASPlugin
}

const (
	ssasUseTLSKey          = "SSAS_USE_TLS"
	ssasURLKey             = "SSAS_URL"
	ssasPublicURLKey       = "SSAS_PUBLIC_URL"
	bcdaSsasClientIDKey    = "BCDA_SSAS_CLIENT_ID"
	bcdaSsasSecretKey      = "BCDA_SSAS_SECRET"
	noSsasErrorMessage     = "no client for SSAS; %s"
	fakeClientID           = "fake-client-id"
	mockClient             = "mock-client"
	unexpectedErrorMessage = "unexpected error; "
)

func (s *SSASPluginTestSuite) SetupSuite() {
	// original values must be saved before we run any tests that might change them
	origSSASUseTLS = os.Getenv(ssasUseTLSKey)
	origSSASURL = os.Getenv(ssasURLKey)
	origPublicURL = os.Getenv(ssasPublicURLKey)
	origSSASClientKeyFile = os.Getenv("SSAS_CLIENT_KEY_FILE")
	origSSASClientCertFile = os.Getenv("SSAS_CLIENT_CERT_FILE")
	origSSASClientID = os.Getenv(bcdaSsasClientIDKey)
	origSSASSecret = os.Getenv(bcdaSsasSecretKey)
}

func (s *SSASPluginTestSuite) SetupTest() {
	db := database.GetGORMDbConnection()
	defer db.Close()

	db.Create(&models.ACO{
		UUID:     uuid.Parse(testACOUUID),
		Name:     "SSAS Plugin Test ACO",
		ClientID: testACOUUID,
	})
}

func (s *SSASPluginTestSuite) TearDownTest() {
	os.Setenv(ssasUseTLSKey, origSSASUseTLS)
	os.Setenv(ssasURLKey, origSSASURL)
	os.Setenv(ssasPublicURLKey, origPublicURL)
	os.Setenv("SSAS_CLIENT_KEY_FILE", origSSASClientKeyFile)
	os.Setenv("SSAS_CLIENT_CERT_FILE", origSSASClientCertFile)
	os.Setenv(bcdaSsasClientIDKey, origSSASClientID)
	os.Setenv(bcdaSsasSecretKey, origSSASSecret)

	db := database.GetGORMDbConnection()
	defer db.Close()

	db.Unscoped().Delete(models.ACO{}, "uuid = ?", testACOUUID)
}

func (s *SSASPluginTestSuite) TestRegisterSystem() {
	// TODO: Mock client instead of server
	router := chi.NewRouter()
	router.Post("/system", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `{ "system_id": "1", "client_id": "fake-client-id", "client_secret": "fake-secret", "client_name": "fake-name" }`)
	})
	server := httptest.NewServer(router)

	os.Setenv("SSAS_URL", server.URL)
	os.Setenv("SSAS_PUBLIC_URL", server.URL)
	os.Setenv("SSAS_USE_TLS", "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf("no client for SSAS; %s", err.Error())
	}
	s.p = SSASPlugin{client: c}

	creds, err := s.p.RegisterSystem(testACOUUID, "", "")
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "1", creds.SystemID)
	assert.Equal(s.T(), fakeClientID, creds.ClientID)
}

func (s *SSASPluginTestSuite) TestRegisterSystem_InvalidJSON() {
	router := chi.NewRouter()
	router.Post("/system", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `"this is": "invalid"`)
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(noSsasErrorMessage, err.Error())
	}
	s.p = SSASPlugin{client: c}

	creds, err := s.p.RegisterSystem(testACOUUID, "", "")
	assert.Contains(s.T(), err.Error(), "failed to unmarshal response json")
	assert.Empty(s.T(), creds.SystemID)
}

func (s *SSASPluginTestSuite) TestUpdateSystem() {}

func (s *SSASPluginTestSuite) TestDeleteSystem() {}

func (s *SSASPluginTestSuite) TestResetSecret() {
	router := chi.NewRouter()
	router.Put("/system/{systemID}/credentials", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		fmt.Fprintf(w, `{ "client_id": "%s", "client_secret": "%s" }`, fakeClientID, "fake-secret")
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(noSsasErrorMessage, err.Error())
	}
	s.p = SSASPlugin{client: c}

	creds, err := s.p.ResetSecret(testACOUUID)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), "fake-client-id", creds.ClientID)
	assert.Equal(s.T(), "fake-secret", creds.ClientSecret)
}

func (s *SSASPluginTestSuite) TestRevokeSystemCredentials() {}

func (s *SSASPluginTestSuite) TestMakeAccessToken() {
	_, tokenString, _ := MockSSASToken()
	MockSSASServer(tokenString)

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(noSsasErrorMessage, err.Error())
	}
	s.p = SSASPlugin{client: c}

	ts, err := s.p.MakeAccessToken(Credentials{ClientID: mockClient, ClientSecret: "mock-secret"})
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), ts)
	assert.Regexp(s.T(), regexp.MustCompile(`[^.\s]+\.[^.\s]+\.[^.\s]+`), ts)

	ts, err = s.p.MakeAccessToken(Credentials{ClientID: "sad", ClientSecret: "customer"})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{ClientID: uuid.NewRandom().String()})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")

	ts, err = s.p.MakeAccessToken(Credentials{ClientID: uuid.NewRandom().String(), ClientSecret: testUtils.RandomBase64(20)})
	assert.NotNil(s.T(), err)
	assert.Empty(s.T(), ts)
	assert.Contains(s.T(), err.Error(), "401")
}

func (s *SSASPluginTestSuite) TestRevokeAccessToken() {
	router := chi.NewRouter()
	router.Delete("/token/{tokenID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")

	c, err := client.NewSSASClient()
	if err != nil {
		log.Fatalf(noSsasErrorMessage, err.Error())
	}
	s.p = SSASPlugin{client: c}

	err = s.p.RevokeAccessToken("i.am.not.a.token")
	assert.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestAuthorizeAccess() {
	_, ts, err := MockSSASToken()
	require.NotNil(s.T(), ts, "no token for SSAS", err)
	require.Nil(s.T(), err, unexpectedErrorMessage, err)
	MockSSASServer(ts)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, "no client for SSAS; ", err)
	s.p = SSASPlugin{client: c}
	err = s.p.AuthorizeAccess(ts)
	require.Nil(s.T(), err)
}

func (s *SSASPluginTestSuite) TestVerifyToken() {
	_, ts, err := MockSSASToken()
	require.NotNil(s.T(), ts, "no token for SSAS; ", err)
	require.Nil(s.T(), err, unexpectedErrorMessage, err)
	MockSSASServer(ts)

	c, err := client.NewSSASClient()
	require.NotNil(s.T(), c, "no client for SSAS; ")
	require.Nil(s.T(), err, unexpectedErrorMessage, err)
	s.p = SSASPlugin{client: c}

	t, err := s.p.VerifyToken(ts)
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
	os.Setenv(bcdaSsasClientIDKey, "bcda")
	os.Setenv(bcdaSsasSecretKey, "api")
	router := chi.NewRouter()
	router.Post("/introspect", func(w http.ResponseWriter, r *http.Request) {
		clientId, secret, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		var answer = make(map[string]bool)
		if clientId == os.Getenv(bcdaSsasClientIDKey) && secret == os.Getenv(bcdaSsasSecretKey) {
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

		if clientId == mockClient {
			render.JSON(w, r, client.TokenResponse{AccessToken: tokenString, TokenType: "access_token"})
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	})

	server := httptest.NewServer(router)

	os.Setenv(ssasURLKey, server.URL)
	os.Setenv(ssasPublicURLKey, server.URL)
	os.Setenv(ssasUseTLSKey, "false")
}

func MockSSASToken() (*jwt.Token, string, error) {
	// NB: currently, BCDA expects only 1 item in the array of cms_ids. At some point, ACO-MS will want to send more than one
	claims := CommonClaims{
		SystemID: "mock-system",
		Data:     `{"cms_ids":["A9995"]}`,
		ClientID: mockClient,
		StandardClaims: jwt.StandardClaims{
			Issuer:    "ssas",
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
			Id:        "mock-id",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	ts, err := InitAlphaBackend().SignJwtToken(t)
	return t, ts, err
}
