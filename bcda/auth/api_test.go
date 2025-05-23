package auth_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi/v5"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	bcdaLog "github.com/CMSgov/bcda-app/log"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models"
)

type AuthAPITestSuite struct {
	suite.Suite
	rr     *httptest.ResponseRecorder
	db     *sql.DB
	r      models.Repository
	server *httptest.Server
}

func (s *AuthAPITestSuite) CreateRouter() http.Handler {
	r := auth.NewAuthRouter()
	return r
}

func (s *AuthAPITestSuite) SetupSuite() {
	s.db = database.Connection
	s.r = postgres.NewRepository(s.db)
}

func (s *AuthAPITestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
	s.server = httptest.NewServer(s.CreateRouter())
}

func (s *AuthAPITestSuite) TestGetAuthTokenErrorSwitchCases() {
	const errorHappened = "Error Happened!"
	const errMsg = "Error Message"

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/token", s.server.URL), nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	//req.Header.Add("Authorization", fmt.Sprintf("Basic %s", tt.authHeader))
	req.Header.Add("Accept", constants.JsonContentType)
	req.SetBasicAuth("good", "client")

	client := s.server.Client()

	tests := []struct {
		ScenarioName          string
		ErrorToReturn         error
		StatusCode            int
		HeaderRetryAfterValue string
	}{
		{"Token Request Timeout Error Return 503", &customErrors.RequestTimeoutError{Err: errors.New(errorHappened), Msg: errMsg}, 503, "1"},
		{"Token Unexpected SSAS Error Return 500", &customErrors.UnexpectedSSASError{Err: errors.New(errorHappened), Msg: errMsg}, 500, constants.EmptyString},
		{"Token Unauthorized SSAS Error Return 401", &customErrors.SSASErrorUnauthorized{Err: errors.New(errorHappened), Msg: errMsg}, 401, constants.EmptyString},
		{"Token Bad Request SSAS Error Return 400", &customErrors.SSASErrorBadRequest{Err: errors.New(errorHappened), Msg: errMsg}, 400, constants.EmptyString},
		{"Token Internal Parsing Error Return 500", &customErrors.InternalParsingError{Err: errors.New(errorHappened), Msg: errMsg}, 500, constants.EmptyString},
		{"Token Default Error Return 500", errors.New(errorHappened), 500, constants.EmptyString},
	}

	for _, tt := range tests {
		//setup logging hook for log message assertion
		testLogger := test.NewLocal(testUtils.GetLogger(bcdaLog.API))

		s.T().Run(tt.ScenarioName, func(t *testing.T) {
			//setup mocks
			mockP := &auth.MockProvider{}
			mockP.On("MakeAccessToken", auth.Credentials{ClientID: "good", ClientSecret: "client"}, mock.Anything).Return("", tt.ErrorToReturn)
			auth.SetMockProvider(s.T(), mockP)

			//Act
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			//Assert
			assert.Equal(s.T(), tt.StatusCode, resp.StatusCode)
			responseBody := testUtils.ReadResponseBody(resp)
			assert.Equal(s.T(), http.StatusText(tt.StatusCode), (strings.TrimSuffix(responseBody, "\n")))
			assert.Equal(s.T(), tt.HeaderRetryAfterValue, resp.Header.Get("Retry-After"))
			mockP.AssertExpectations(s.T())

			//assert the correct log message wording was logged to API log
			assert.Equal(t, 1, len(testLogger.Entries))
			assert.Equal(t, fmt.Sprintf("Error making access token - %s | HTTPS Status Code: %v", tt.ErrorToReturn.Error(), tt.StatusCode), testLogger.LastEntry().Message)
			testLogger.Reset()
		})
	}
}

func (s *AuthAPITestSuite) TestGetAuthToken() {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/auth/token", s.server.URL), nil)
	if err != nil {
		assert.FailNow(s.T(), err.Error())
	}
	req.Header.Add("Accept", constants.JsonContentType)
	req.SetBasicAuth("good", "client")

	client := s.server.Client()

	tests := []struct {
		ScenarioName          string
		ErrorToReturn         error
		StatusCode            int
		HeaderRetryAfterValue string
	}{
		{"Authorized Token Basic Auth", nil, 200, constants.EmptyString},
	}

	for _, tt := range tests {
		s.T().Run(tt.ScenarioName, func(t *testing.T) {

			//setup mocks
			mockP := &auth.MockProvider{}
			mockP.On("MakeAccessToken", auth.Credentials{ClientID: "good", ClientSecret: "client"}, mock.Anything).Return(fmt.Sprintf(`{ "token_type": "bearer", "access_token": "goodToken", "expires_in": "%s" }`, constants.ExpiresInDefault), tt.ErrorToReturn)
			auth.SetMockProvider(s.T(), mockP)

			//Act
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			respMap := make(map[string]string)
			bodyBytes, err := io.ReadAll(resp.Body)

			//Assert
			assert.Nil(s.T(), err)
			assert.NoError(s.T(), json.Unmarshal(bodyBytes, &respMap))
			assert.Equal(s.T(), tt.StatusCode, resp.StatusCode)
			assert.Equal(s.T(), tt.HeaderRetryAfterValue, resp.Header.Get("Retry-After"))
			assert.Equal(s.T(), resp.Header.Get("Content-Type"), "application/json")
			assert.Equal(s.T(), resp.Header.Get("Cache-Control"), "no-store")
			assert.Equal(s.T(), resp.Header.Get("Pragma"), "no-cache")
			assert.Equal(s.T(), "goodToken", respMap["access_token"])
			assert.Equal(s.T(), constants.ExpiresInDefault, respMap["expires_in"])
			mockP.AssertExpectations(s.T())
		})
	}

}

func (s *AuthAPITestSuite) TestWelcome() {
	goodToken, badToken := uuid.New(), uuid.New()
	mockP := &auth.MockProvider{}

	var ad auth.AuthData
	token := &jwt.Token{Raw: goodToken, Valid: true, Claims: &auth.CommonClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer: "ssas",
		},
		ClientID: uuid.New(),
		SystemID: uuid.New(),
		Data:     `{"cms_ids":["A9994"]}`,
	}}

	mockP.On("VerifyToken", mock.Anything, goodToken).Return(token, nil)
	mockP.On("VerifyToken", mock.Anything, badToken).Return(nil, errors.New("bad token"))
	mockP.On("getAuthDataFromClaims", token.Claims).Return(ad, nil)
	auth.SetMockProvider(s.T(), mockP)

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
	bodyBytes, err := io.ReadAll(resp.Body)
	assert.Nil(s.T(), err)
	assert.NoError(s.T(), json.Unmarshal(bodyBytes, &respMap))
	assert.NotEmpty(s.T(), respMap)
	assert.Equal(s.T(), "Welcome to the Beneficiary Claims Data API!", respMap["success"])

	mockP.AssertExpectations(s.T())
}

func TestAuthAPITestSuite(t *testing.T) {
	suite.Run(t, new(AuthAPITestSuite))
}
