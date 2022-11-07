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
	"github.com/go-chi/chi/v5"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in,omitempty"`
	TokenType   string `json:"token_type"`
}

type AuthAPITestSuite struct {
	suite.Suite
	rr *httptest.ResponseRecorder
	db *sql.DB
	r  models.Repository
}

func (s *AuthAPITestSuite) SetupSuite() {
	s.db = database.Connection
	s.r = postgres.NewRepository(s.db)
}

func (s *AuthAPITestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
}

func (s *AuthAPITestSuite) MockMakeAccessToken() {
	clientID, clientSecret, accessToken := "happy", "client", "goodToken"
	mock := &auth.MockProvider{}
	mock.On("MakeAccessToken", auth.Credentials{ClientID: clientID, ClientSecret: clientSecret}).
		Return(accessToken, constants.ExpiresInDefault, nil)
	mock.On("MakeAccessToken", auth.Credentials{ClientID: "not_a_client", ClientSecret: "not_a_secret"}).
		Return("", "", errors.New("some auth error"))
	auth.SetMockProvider(s.T(), mock)
}

func (s *AuthAPITestSuite) TestBadRequestAuthToken() {
	s.MockMakeAccessToken()
	// Missing authorization header
	req := httptest.NewRequest("POST", constants.TokenPath, nil)
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func (s *AuthAPITestSuite) TestBadRequestAuthTokenHeader() {
	s.MockMakeAccessToken()

	// Malformed authorization header
	s.rr = httptest.NewRecorder()
	req := httptest.NewRequest("POST", constants.TokenPath, nil)
	req.Header.Add("Authorization", "Basic not_an_encoded_client_and_secret")
	req.Header.Add("Accept", constants.JsonContentType)
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusBadRequest, s.rr.Code)
}

func (s *AuthAPITestSuite) TestUnauthorizedAuthTokenBasicAuth() {
	s.MockMakeAccessToken()

	// Invalid credentials
	s.rr = httptest.NewRecorder()
	req := httptest.NewRequest("POST", constants.TokenPath, nil)
	req.SetBasicAuth("not_a_client", "not_a_secret")
	req.Header.Add("Accept", constants.JsonContentType)
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)
}

func (s *AuthAPITestSuite) TestSuccessAuthToken() {
	s.MockMakeAccessToken()

	// Success!?
	s.rr = httptest.NewRecorder()
	req := httptest.NewRequest("POST", constants.TokenPath, nil)
	req.SetBasicAuth("happy", "client")
	req.Header.Add("Accept", constants.JsonContentType)
	handler := http.HandlerFunc(auth.GetAuthToken)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusOK, s.rr.Code)

	var t TokenResponse
	assert.NoError(s.T(), json.NewDecoder(s.rr.Body).Decode(&t))
	assert.Equal(s.T(), "goodToken", t.AccessToken)
	assert.Equal(s.T(), constants.ExpiresInDefault, t.ExpiresIn)
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
	router.With(auth.RequireTokenAuth).Get("/v1/", auth.Welcome)
	server := httptest.NewServer(router)
	client := server.Client()
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/", server.URL), nil)
	assert.NoError(s.T(), err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", badToken))
	req.Header.Add("Accept", constants.JsonContentType)
	resp, err := client.Do(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)

	// Expect success with valid token
	req, err = http.NewRequest("GET", fmt.Sprintf("%s/v1/", server.URL), nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", goodToken))
	req.Header.Add("Accept", constants.JsonContentType)
	resp, err = client.Do(req)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	respMap := make(map[string]string)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(s.T(), err)
	assert.NoError(s.T(), json.Unmarshal(bodyBytes, &respMap))
	assert.NotEmpty(s.T(), respMap)
	assert.Equal(s.T(), "Welcome to the Beneficiary Claims Data API!", respMap["success"])

	mock.AssertExpectations(s.T())
}

func TestAuthAPITestSuite(t *testing.T) {
	suite.Run(t, new(AuthAPITestSuite))
}
