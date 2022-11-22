package auth_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	mock.On("MakeAccessToken", auth.Credentials{ClientID: "", ClientSecret: ""}).
		Return("", errors.New("some auth error"))
	mock.On("MakeAccessToken", auth.Credentials{ClientID: "not_a_client", ClientSecret: "not_a_secret"}).
		Return("", errors.New("some auth error"))
	mock.On("MakeAccessToken", auth.Credentials{ClientID: clientID, ClientSecret: clientSecret}).
		Return(fmt.Sprintf(`{ "token_type": "bearer", "access_token": "%s", "expires_in": "%s" }`, accessToken, constants.ExpiresInDefault), nil)
	auth.SetMockProvider(s.T(), mock)
}

func (s *AuthAPITestSuite) TestAuthToken() {
	const goodClientId, goodClientSecret, goodToken = "happy", "client", "goodToken"
	const badClientId, badClientSecret = "not_a_client", "not_a_secret"
	const badAuthHeader = "not_an_encoded_client_and_secret"

	tests := []struct {
		scenarioName       string
		clientId           string
		clientSecret       string
		authHeader         string
		tokenString        string
		expiresIn          string
		expectedStatusCode int
	}{
		{"Uauthorized Auth Token", constants.EmptyString, constants.EmptyString, constants.EmptyString, constants.EmptyString, constants.EmptyString, http.StatusUnauthorized},
		{"Uauthorized Auth Token Header", constants.EmptyString, constants.EmptyString, badAuthHeader, constants.EmptyString, constants.EmptyString, http.StatusUnauthorized},
		{"Uauthorized Token Basic Auth", badClientId, badClientSecret, constants.EmptyString, constants.EmptyString, constants.EmptyString, http.StatusUnauthorized},
		{"Authorized Token Basic Auth", goodClientId, goodClientSecret, constants.EmptyString, goodToken, constants.ExpiresInDefault, http.StatusOK},
	}

	for _, tt := range tests {
		s.T().Run(tt.scenarioName, func(t *testing.T) {
			s.MockMakeAccessToken()

			s.rr = httptest.NewRecorder()
			req := httptest.NewRequest("POST", constants.TokenPath, nil)
			req.Header.Add("Authorization", fmt.Sprintf("Basic %s", tt.authHeader))
			req.Header.Add("Accept", constants.JsonContentType)
			req.SetBasicAuth(tt.clientId, tt.clientSecret)
			handler := http.HandlerFunc(auth.GetAuthToken)
			handler.ServeHTTP(s.rr, req)

			assert.Equal(s.T(), tt.expectedStatusCode, s.rr.Code)

			if s.rr.Code == 200 {
				respMap := make(map[string]string)
				bodyBytes, err := io.ReadAll(s.rr.Body)
				assert.Nil(s.T(), err)
				assert.NoError(s.T(), json.Unmarshal(bodyBytes, &respMap))
				assert.Equal(s.T(), tt.tokenString, respMap["access_token"])
				assert.Equal(s.T(), tt.expiresIn, respMap["expires_in"])
			}
		})
	}
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
