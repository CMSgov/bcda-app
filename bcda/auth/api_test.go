package auth_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/models/postgres"

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
	rr    *httptest.ResponseRecorder
	db    *sql.DB
	r     models.Repository
	reset func()
}

func (s *AuthAPITestSuite) SetupSuite() {
	private := testUtils.SetAndRestoreEnvKey("JWT_PRIVATE_KEY_FILE", "../../shared_files/api_unit_test_auth_private.pem")
	public := testUtils.SetAndRestoreEnvKey("JWT_PUBLIC_KEY_FILE", "../../shared_files/api_unit_test_auth_public.pem")
	s.reset = func() {
		private()
		public()
	}

	s.db = database.Connection
	s.r = postgres.NewRepository(s.db)
}

func (s *AuthAPITestSuite) TearDownSuite() {
	s.reset()
}

func (s *AuthAPITestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
}

func (s *AuthAPITestSuite) TestAuthToken() {
	clientID, clientSecret, accessToken := uuid.New(), uuid.New(), uuid.New()
	mock := &auth.MockProvider{}
	mock.On("MakeAccessToken", auth.Credentials{ClientID: clientID, ClientSecret: clientSecret}).
		Return(accessToken, nil)
	mock.On("MakeAccessToken", auth.Credentials{ClientID: "not_a_client", ClientSecret: "not_a_secret"}).
		Return("", errors.New("some auth error"))
	auth.SetMockProvider(s.T(), mock)

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
	req = httptest.NewRequest("POST", "/auth/token", nil)
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Add("Accept", "application/json")
	handler = http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)

	var t TokenResponse
	assert.NoError(s.T(), json.NewDecoder(s.rr.Body).Decode(&t))
	assert.Equal(s.T(), accessToken, t.AccessToken)

	mock.AssertExpectations(s.T())
}

func (s *AuthAPITestSuite) TestWelcome() {
	goodToken, badToken := uuid.New(), uuid.New()
	mock := &auth.MockProvider{}
	mock.On("VerifyToken", goodToken).Return(&jwt.Token{Raw: goodToken}, nil)
	mock.On("VerifyToken", badToken).Return(nil, errors.New("bad token"))
	mock.On("AuthorizeAccess", goodToken).Return(nil)
	auth.SetMockProvider(s.T(), mock)

	// Expect failure with invalid token
	router := chi.NewRouter()
	router.Use(auth.ParseToken)
	router.With(auth.RequireTokenAuth).Get("/", auth.Welcome)
	server := httptest.NewServer(router)
	client := server.Client()
	req, err := http.NewRequest("GET", server.URL, nil)
	assert.NoError(s.T(), err)
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
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", goodToken))
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
