package auth_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/dgrijalva/jwt-go"

	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
)

var mockHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	// mockHandler function for testing http requests related to middleware test scripts
}
var bearerStringMsg string = "Bearer %s"

type MiddlewareTestSuite struct {
	suite.Suite
	server *httptest.Server
	rr     *httptest.ResponseRecorder
}

func (s *MiddlewareTestSuite) CreateRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(auth.ParseToken)
	router.With(auth.RequireTokenAuth).Get("/v1/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test router"))
		if err != nil {
			log.Fatal(err)
		}
	})

	return router
}

func (s *MiddlewareTestSuite) SetupTest() {
	s.server = httptest.NewServer(s.CreateRouter())
	s.rr = httptest.NewRecorder()
}

func (s *MiddlewareTestSuite) TearDownTest() {
	s.server.Close()
}

//integration test: makes HTTP request & asserts HTTP response
func (s *MiddlewareTestSuite) TestReturn400WhenInvalidTokenAuthWithInvalidSignature() {
	client := s.server.Client()
	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, badToken))
	resp, err := client.Do(req)

	assert.NotNil(s.T(), resp)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 400, resp.StatusCode)
	assert.Nil(s.T(), err)
}

//unit test
func (s *MiddlewareTestSuite) TestRequireTokenAuthReturn401WhenInvalidToken() {
	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenAuth(mockHandler)

	tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String()
	token := &jwt.Token{Raw: base64.StdEncoding.EncodeToString([]byte("SOME_INVALID_BEARER_TOKEN"))}

	mock := &auth.MockProvider{}
	mock.On("AuthorizeAccess", token.Raw).Return(errors.New("invalid token"))
	auth.SetMockProvider(s.T(), mock)

	ad := auth.AuthData{
		ACOID:   acoID,
		TokenID: tokenID,
	}

	ctx := req.Context()
	ctx = context.WithValue(ctx, auth.TokenContextKey, token)
	ctx = context.WithValue(ctx, auth.AuthDataContextKey, ad)
	req = req.WithContext(ctx)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 401, s.rr.Code)

	mock.AssertExpectations(s.T())
}

//integration test: makes HTTP request & asserts HTTP response
func (s *MiddlewareTestSuite) TestAuthMiddlewareReturnResponse200WhenValidBearerTokenSupplied() {
	bearerString := uuid.New()

	tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String()

	authData := auth.AuthData{
		ACOID:   acoID,
		TokenID: tokenID,
	}

	token := &jwt.Token{
		Claims: &auth.CommonClaims{
			StandardClaims: jwt.StandardClaims{
				Issuer: "ssas",
			},
			ClientID: uuid.New(),
			SystemID: uuid.New(),
			Data:     `{"cms_ids":["A9994"]}`,
		},
		Raw:   uuid.New(),
		Valid: true}

	mock := &auth.MockProvider{}
	mock.On("VerifyToken", bearerString).Return(token, nil)
	mock.On("getAuthDataFromClaims", token.Claims).Return(authData, nil)
	mock.On("AuthorizeAccess", token.Raw).Return(nil)
	auth.SetMockProvider(s.T(), mock)

	client := s.server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(s.T(), 200, resp.StatusCode)

	mock.AssertExpectations(s.T())
}

func setupDataForAuthMiddlewareTest() (bearerString string, authData auth.AuthData, token *jwt.Token, cmsID string) {
	cmsID = testUtils.RandomHexID()[0:4]

	bearerString, tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String(), uuid.NewRandom().String()

	authData = auth.AuthData{
		ACOID:   acoID,
		TokenID: tokenID,
	}

	token = &jwt.Token{
		Claims: &auth.CommonClaims{
			StandardClaims: jwt.StandardClaims{
				Issuer: "ssas",
			},
			ClientID: uuid.New(),
			SystemID: uuid.New(),
			Data:     fmt.Sprintf(`{"cms_ids":["%s"]}`, cmsID),
		},
		Raw:   uuid.New(),
		Valid: true}

	return bearerString, authData, token, cmsID
}

func (s *MiddlewareTestSuite) TestTokenVerificationErrorHandling() {
	bearerString := uuid.NewRandom().String()
	const errorHappened = "Error Happened!"
	const errMsg = "Error Message"

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

	client := s.server.Client()

	tests := []struct {
		ScenarioName       string
		ErrorToReturn      error
		StatusCode         int
		ResponseBodyString string
	}{
		{"Requestor Data Error Return 400", &customErrors.RequestorDataError{Err: errors.New(errorHappened), Msg: errMsg}, 400, responseutils.InternalErr},
		{"Internal Parsing Error Return 500", &customErrors.InternalParsingError{Err: errors.New(errorHappened), Msg: errMsg}, 500, responseutils.InternalErr},
		{"Config Error Return 500", &customErrors.ConfigError{Err: errors.New(errorHappened), Msg: errMsg}, 500, responseutils.InternalErr},
		{"Request Timeout Error Return 503", &customErrors.RequestTimeoutError{Err: errors.New(errorHappened), Msg: errMsg}, 503, responseutils.InternalErr},
		{"Unexpected SSAS Error Return 500", &customErrors.UnexpectedSSASError{Err: errors.New(errorHappened), Msg: errMsg}, 500, responseutils.InternalErr},
		{"Expired Token Error Return 401", &customErrors.ExpiredTokenError{Err: errors.New(errorHappened), Msg: errMsg}, 401, responseutils.TokenErr},
		{"Default Error Return 401", errors.New(errorHappened), 401, responseutils.TokenErr},
	}

	for _, tt := range tests {
		s.T().Run(tt.ScenarioName, func(t *testing.T) {

			//setup mocks
			mock := &auth.MockProvider{}
			mock.On("VerifyToken", bearerString).Return(nil, tt.ErrorToReturn)
			auth.SetMockProvider(s.T(), mock)

			//Act
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			//Assert
			assert.Equal(s.T(), tt.StatusCode, resp.StatusCode)
			assert.Contains(s.T(), testUtils.ReadResponseBody(resp), tt.ResponseBodyString)
			mock.AssertExpectations(s.T())
		})
	}

}

func (s *MiddlewareTestSuite) TestAuthMiddlewareReturnResponse403WhenEntityNotFoundError() {
	bearerString, authData, token, cmsID := setupDataForAuthMiddlewareTest()

	//custom error expected
	dbErr := errors.New("DB Error: ACO Does Not Exist!")
	entityNotFoundError := &customErrors.EntityNotFoundError{Err: dbErr, CMSID: cmsID}

	//setup mocks
	mock := &auth.MockProvider{}
	mock.On("VerifyToken", bearerString).Return(token, nil)
	mock.On("getAuthDataFromClaims", token.Claims).Return(authData, entityNotFoundError)
	auth.SetMockProvider(s.T(), mock)

	//fill http request
	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

	client := s.server.Client()
	s.rr = httptest.NewRecorder()

	//Act
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	//Assert
	assert.Equal(s.T(), 403, resp.StatusCode)
	assert.Contains(s.T(), testUtils.ReadResponseBody(resp), responseutils.UnknownEntityErr)

	mock.AssertExpectations(s.T())
}

func (s *MiddlewareTestSuite) TestAuthMiddlewareReturn401WhenNonEntityNotFoundError() {

	bearerString, authData, token, _ := setupDataForAuthMiddlewareTest()

	//custom error expected
	thrownErr := errors.New("error123")

	//setup mocks
	mock := &auth.MockProvider{}
	mock.On("VerifyToken", bearerString).Return(token, nil)
	mock.On("getAuthDataFromClaims", token.Claims).Return(authData, thrownErr)
	auth.SetMockProvider(s.T(), mock)

	//fill http request
	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

	client := s.server.Client()

	//Act
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	// Assert
	assert.Equal(s.T(), 401, resp.StatusCode)

	mock.AssertExpectations(s.T())
}

//integration test: makes HTTP request & asserts HTTP response
func (s *MiddlewareTestSuite) TestAuthMiddlewareReturnResponse401WhenNoBearerTokenSupplied() {
	client := s.server.Client()

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(s.T(), 401, resp.StatusCode)
}

//integration test: involves db connection to postgres
func (s *MiddlewareTestSuite) TestRequireTokenJobMatchReturn404WhenMismatchingDataProvided() {
	db := database.Connection
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusFailed,
	}

	postgrestest.CreateJobs(s.T(), db, &j)
	jobID := strconv.Itoa(int(j.ID))

	tests := []struct {
		name  string
		jobID string
		ACOID string
	}{
		{"Invalid JobID", "someNonNumericInput", j.ACOID.String()},
		{"Mismatching JobID", "0", j.ACOID.String()},
		{"Mismatching ACOID", jobID, uuid.New()},
	}

	handler := auth.RequireTokenJobMatch(mockHandler)

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.rr = httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)

			req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
			assert.NoError(t, err)

			ad := auth.AuthData{
				ACOID:   tt.ACOID,
				TokenID: uuid.New(),
			}
			ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)

			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
			handler.ServeHTTP(s.rr, req)
			assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)
		})
	}
}

//integration test: involves db connection to postgres
func (s *MiddlewareTestSuite) TestRequireTokenJobMatchReturn200WhenCorrectAccountableCareOrganizationAndJob() {
	db := database.Connection

	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusFailed,
	}
	postgrestest.CreateJobs(s.T(), db, &j)
	jobID := strconv.Itoa(int(j.ID))

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)

	handler := auth.RequireTokenJobMatch(mockHandler)

	ad := auth.AuthData{
		ACOID:   j.ACOID.String(),
		TokenID: uuid.New(),
	}
	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)

	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 200, s.rr.Code)
}

//integration test: involves db connection to postgres
func (s *MiddlewareTestSuite) TestRequireTokenJobMatchReturn404WhenNoAuthDataProvidedInContext() {
	db := database.Connection

	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusFailed,
	}

	postgrestest.CreateJobs(s.T(), db, &j)
	jobID := strconv.Itoa(int(j.ID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, s.server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenJobMatch(mockHandler)

	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)
}

//unit test
func (s *MiddlewareTestSuite) TestCheckBlacklist() {
	blacklisted := testUtils.RandomHexID()[0:4]
	notBlacklisted := testUtils.RandomHexID()[0:4]

	handler := auth.CheckBlacklist(mockHandler)
	tests := []struct {
		name            string
		ad              *auth.AuthData
		expectedCode    int
		expectedMessage string
	}{
		{"No auth data found", nil, http.StatusNotFound, "AuthData not found"},
		{"Blacklisted ACO", &auth.AuthData{CMSID: blacklisted, Blacklisted: true}, http.StatusForbidden,
			fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", blacklisted)},
		{"Non-blacklisted ACO", &auth.AuthData{CMSID: notBlacklisted, Blacklisted: false}, http.StatusOK, ""},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			ctx := context.Background()
			if tt.ad != nil {
				ctx = context.WithValue(ctx, auth.AuthDataContextKey, *tt.ad)
			}
			req, err := http.NewRequestWithContext(ctx, "GET", "/v1/", nil)
			assert.NoError(t, err)
			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedCode, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.expectedMessage)
		})
	}

}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
