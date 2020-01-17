package auth_test

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type AuthAPITestSuite struct {
	suite.Suite
	rr      *httptest.ResponseRecorder
	db      *gorm.DB
	backend *auth.AlphaBackend
	reset   func()
}

func (s *AuthAPITestSuite) SetupSuite() {
	private := testUtils.SetAndRestoreEnvKey("JWT_PRIVATE_KEY_FILE", "../../shared_files/api_unit_test_auth_private.pem")
	public := testUtils.SetAndRestoreEnvKey("JWT_PUBLIC_KEY_FILE", "../../shared_files/api_unit_test_auth_public.pem")
	s.reset = func() {
		private()
		public()
	}
	s.backend = auth.InitAlphaBackend()
}

func (s *AuthAPITestSuite) TearDownSuite() {
	s.reset()
}

func (s *AuthAPITestSuite) SetupTest() {
	models.InitializeGormModels()
	auth.InitializeGormModels()
	s.db = database.GetGORMDbConnection()
	s.rr = httptest.NewRecorder()
}

func (s *AuthAPITestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *AuthAPITestSuite) TestAuthToken() {
	var aco models.ACO
	err := s.db.Where("uuid = ?", constants.DevACOUUID).First(&aco).Error
	assert.Nil(s.T(), err)
	aco.AlphaSecret = ""
	s.db.Save(&aco)

	// Missing authorization header
	req := httptest.NewRequest("POST", "/auth/token", nil)
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)

	// Malformed authorization header
	s.rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/auth/token", nil)
	req.Header.Add("Authorization", "Basic not_an_encoded_client_and_secret")
	req.Header.Add("Accept", "application/json")
	handler = http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)

	// Invalid credentials
	s.rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth("not_a_client", "not_a_secret")
	req.Header.Add("Accept", "application/json")
	handler = http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)

	// Success!?
	s.rr = httptest.NewRecorder()
	t := TokenResponse{}
	creds, err := auth.GetProvider().RegisterSystem(constants.DevACOUUID, "", "")
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), creds.ClientID)
	assert.NotEmpty(s.T(), creds.ClientSecret)

	req = httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	req.Header.Add("Accept", "application/json")
	handler = http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.NoError(s.T(), json.NewDecoder(s.rr.Body).Decode(&t))
	assert.NotEmpty(s.T(), t)
	assert.NotEmpty(s.T(), t.AccessToken)
}

func (s *AuthAPITestSuite) TestWelcome() {
	// Setup
	var aco models.ACO
	err := s.db.Where("uuid = ?", constants.DevACOUUID).First(&aco).Error
	assert.Nil(s.T(), err)
	aco.AlphaSecret = ""
	s.db.Save(&aco)
	s.rr = httptest.NewRecorder()
	t := TokenResponse{}
	creds, err := auth.GetProvider().RegisterSystem(constants.DevACOUUID, "", "")
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), creds.ClientID)
	assert.NotEmpty(s.T(), creds.ClientSecret)

	// Get token
	req := httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth(creds.ClientID, creds.ClientSecret)
	req.Header.Add("Accept", "application/json")
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)
	assert.NoError(s.T(), json.NewDecoder(s.rr.Body).Decode(&t))
	assert.NotEmpty(s.T(), t)
	assert.NotEmpty(s.T(), t.AccessToken)

	// Expect failure with invalid token
	router := chi.NewRouter()
	router.Use(auth.ParseToken)
	router.With(auth.RequireTokenAuth).Get("/", auth.Welcome)
	server := httptest.NewServer(router)
	client := server.Client()
	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"
	req, err = http.NewRequest("GET", server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", badToken))
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)

	// Expect success with valid token
	req, err = http.NewRequest("GET", server.URL, nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.AccessToken))
	req.Header.Add("Accept", "application/json")
	resp, err = client.Do(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	respMap := make(map[string]string)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(s.T(), err)
	assert.NoError(s.T(), json.Unmarshal(bodyBytes, &respMap))
	assert.NotEmpty(s.T(), respMap)
	assert.Equal(s.T(), "Welcome to the Beneficiary Claims Data API!", respMap["success"])
}

func TestAuthAPITestSuite(t *testing.T) {
	suite.Run(t, new(AuthAPITestSuite))
}
