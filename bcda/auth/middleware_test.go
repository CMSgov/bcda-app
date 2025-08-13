package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	customErrors "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	responseutils "github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/dgrijalva/jwt-go"

	"github.com/go-chi/chi/v5"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/ccoveille/go-safecast"
)

var mockHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
	// mockHandler function for testing http requests related to middleware test scripts
}
var bearerStringMsg string = "Bearer %s"

type MiddlewareTestSuite struct {
	suite.Suite
	rr *httptest.ResponseRecorder
	db *sql.DB
}

func (s *MiddlewareTestSuite) SetupSuite() {
	s.db = database.Connect()
}

func (s *MiddlewareTestSuite) CreateRouter(p auth.Provider) http.Handler {
	am := auth.NewAuthMiddleware(p)
	router := chi.NewRouter()
	router.Use(am.ParseToken)
	router.With(auth.RequireTokenAuth).Get("/v1/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test router"))
		if err != nil {
			log.Fatal(err)
		}
	})

	return router
}

func (s *MiddlewareTestSuite) CreateServer(p auth.Provider) *httptest.Server {
	return httptest.NewServer(s.CreateRouter(p))
}

func (s *MiddlewareTestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
}

// integration test: makes HTTP request & asserts HTTP response
func (s *MiddlewareTestSuite) TestReturn400WhenInvalidTokenAuthWithInvalidSignature() {
	server := s.CreateServer(auth.NewProvider(s.db))
	defer server.Close()
	client := server.Client()
	badT := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, badT))
	resp, err := client.Do(req)

	assert.NotNil(s.T(), resp)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 400, resp.StatusCode)
	assert.Nil(s.T(), err)
}

// integration test: makes HTTP request & asserts HTTP response
func (s *MiddlewareTestSuite) TestReturn401WhenExpiredToken() {
	server := s.CreateServer(auth.NewProvider(s.db))
	defer server.Close()
	client := server.Client()
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodRS512, &auth.CommonClaims{
		StandardClaims: jwt.StandardClaims{
			Issuer:    "ssas",
			ExpiresAt: time.Now().Unix(),
		},
		ClientID: uuid.New(),
		SystemID: uuid.New(),
		Data:     `{"cms_ids":["A9994"]}`,
	})
	pk, _ := rsa.GenerateKey(rand.Reader, 2048)
	tokenString, _ := expiredToken.SignedString(pk)

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, tokenString))
	resp, err := client.Do(req)

	assert.NotNil(s.T(), resp)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 401, resp.StatusCode)
	assert.Equal(s.T(), "401 Unauthorized", resp.Status)
	assert.Contains(s.T(), testUtils.ReadResponseBody(resp), "Expired Token")
	assert.Nil(s.T(), err)
}

// integration test: makes HTTP request & asserts HTTP response
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

	mockP := &auth.MockProvider{}
	mockP.On("VerifyToken", mock.Anything, bearerString).Return(token, nil)
	mockP.On("getAuthDataFromClaims", token.Claims).Return(authData, nil)

	server := s.CreateServer(mockP)
	defer server.Close()
	client := server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(s.T(), 200, resp.StatusCode)

	mockP.AssertExpectations(s.T())
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

	tests := []struct {
		ScenarioName          string
		ErrorToReturn         error
		StatusCode            int
		ResponseBodyString    string
		HeaderRetryAfterValue string
	}{
		{"Requestor Data Error Return 400", &customErrors.RequestorDataError{Err: errors.New(errorHappened), Msg: errMsg}, 400, responseutils.RequestErr, constants.EmptyString},
		{"Internal Parsing Error Return 500", &customErrors.InternalParsingError{Err: errors.New(errorHappened), Msg: errMsg}, 500, responseutils.InternalErr, constants.EmptyString},
		{"Config Error Return 500", &customErrors.ConfigError{Err: errors.New(errorHappened), Msg: errMsg}, 500, responseutils.InternalErr, constants.EmptyString},
		{"Request Timeout Error Return 503", &customErrors.RequestTimeoutError{Err: errors.New(errorHappened), Msg: errMsg}, 503, responseutils.InternalErr, "1"},
		{"Unexpected SSAS Error Return 500", &customErrors.UnexpectedSSASError{Err: errors.New(errorHappened), Msg: errMsg}, 500, responseutils.InternalErr, constants.EmptyString},
		{"Expired Token Error Return 401", &customErrors.ExpiredTokenError{Err: errors.New(errorHappened), Msg: errMsg}, 401, responseutils.ExpiredErr, constants.EmptyString},
		{"Default Error Return 401", errors.New(errorHappened), 401, responseutils.TokenErr, constants.EmptyString},
	}

	for _, tt := range tests {
		s.T().Run(tt.ScenarioName, func(t *testing.T) {

			//setup mocks
			mockP := &auth.MockProvider{}
			mockP.On("VerifyToken", mock.Anything, bearerString).Return(nil, tt.ErrorToReturn)
			server := s.CreateServer(mockP)
			defer server.Close()
			client := server.Client()

			req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
			if err != nil {
				log.Fatal(err)
			}
			req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

			//Act
			resp, err := client.Do(req)
			if err != nil {
				log.Fatal(err)
			}

			//Assert
			assert.Equal(s.T(), tt.StatusCode, resp.StatusCode)
			assert.Equal(s.T(), tt.HeaderRetryAfterValue, resp.Header.Get("Retry-After"))
			assert.Contains(s.T(), testUtils.ReadResponseBody(resp), tt.ResponseBodyString)
			mockP.AssertExpectations(s.T())
		})
	}

}

func (s *MiddlewareTestSuite) TestAuthMiddlewareReturnResponse403WhenEntityNotFoundError() {
	bearerString, authData, token, cmsID := setupDataForAuthMiddlewareTest()

	//custom error expected
	dbErr := errors.New("DB Error: ACO Does Not Exist!")
	entityNotFoundError := &customErrors.EntityNotFoundError{Err: dbErr, CMSID: cmsID}

	//setup mocks
	mockP := &auth.MockProvider{}
	mockP.On("VerifyToken", mock.Anything, bearerString).Return(token, nil)
	mockP.On("getAuthDataFromClaims", token.Claims).Return(authData, entityNotFoundError)

	server := s.CreateServer(mockP)
	client := server.Client()
	s.rr = httptest.NewRecorder()

	//fill http request
	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

	//Act
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	//Assert
	assert.Equal(s.T(), 403, resp.StatusCode)
	assert.Contains(s.T(), testUtils.ReadResponseBody(resp), responseutils.UnauthorizedErr)

	mockP.AssertExpectations(s.T())
}

func (s *MiddlewareTestSuite) TestAuthMiddlewareReturn401WhenNonEntityNotFoundError() {
	bearerString, authData, token, _ := setupDataForAuthMiddlewareTest()

	//custom error expected
	thrownErr := errors.New("error123")

	//setup mocks
	mockP := &auth.MockProvider{}
	mockP.On("VerifyToken", mock.Anything, bearerString).Return(token, nil)
	mockP.On("getAuthDataFromClaims", token.Claims).Return(authData, thrownErr)

	server := s.CreateServer(mockP)
	defer server.Close()
	client := server.Client()

	//fill http request
	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf(bearerStringMsg, bearerString))

	//Act
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	// Assert
	assert.Equal(s.T(), 401, resp.StatusCode)

	mockP.AssertExpectations(s.T())
}

// integration test: makes HTTP request & asserts HTTP response
func (s *MiddlewareTestSuite) TestAuthMiddlewareReturnResponse401WhenNoBearerTokenSupplied() {
	server := s.CreateServer(auth.NewProvider(s.db))
	defer server.Close()
	client := server.Client()

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(s.T(), 401, resp.StatusCode)
	assert.Equal(s.T(), "401 Unauthorized", resp.Status)
}

// integration test: involves db connection to postgres
func (s *MiddlewareTestSuite) TestRequireTokenJobMatchReturn404WhenMismatchingDataProvided() {
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusFailed,
	}

	postgrestest.CreateJobs(s.T(), s.db, &j)
	id, err := safecast.ToInt(j.ID)
	if err != nil {
		log.Fatal(err)
	}
	jobID := strconv.Itoa(id)

	tests := []struct {
		name    string
		jobID   string
		ACOID   string
		errCode int
	}{
		{"Invalid JobID", "someNonNumericInput", j.ACOID.String(), http.StatusBadRequest},
		{"Mismatching JobID", "0", j.ACOID.String(), http.StatusNotFound},
		{"Mismatching ACOID", jobID, uuid.New(), http.StatusUnauthorized},
	}

	p := auth.NewProvider(s.db)
	am := auth.NewAuthMiddleware(p)
	handler := am.RequireTokenJobMatch(s.db)(mockHandler)
	server := s.CreateServer(p)
	defer server.Close()

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.rr = httptest.NewRecorder()

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)

			req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
			assert.NoError(t, err)

			ad := auth.AuthData{
				ACOID:   tt.ACOID,
				TokenID: uuid.New(),
			}
			ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)

			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
			handler.ServeHTTP(s.rr, req)
			assert.Equal(s.T(), tt.errCode, s.rr.Code)
		})
	}
}

// integration test: involves db connection to postgres
func (s *MiddlewareTestSuite) TestRequireTokenJobMatchReturn200WhenCorrectAccountableCareOrganizationAndJob() {
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusFailed,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)
	id, err := safecast.ToInt(j.ID)
	if err != nil {
		log.Fatal(err)
	}
	jobID := strconv.Itoa(id)

	p := auth.NewProvider(s.db)
	am := auth.NewAuthMiddleware(p)
	handler := am.RequireTokenJobMatch(s.db)(mockHandler)
	server := s.CreateServer(p)
	defer server.Close()

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)

	ad := auth.AuthData{
		ACOID:   j.ACOID.String(),
		TokenID: uuid.New(),
	}
	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)

	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 200, s.rr.Code)
}

// integration test: involves db connection to postgres
func (s *MiddlewareTestSuite) TestRequireTokenJobMatchReturn404WhenNoAuthDataProvidedInContext() {
	j := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusFailed,
	}

	postgrestest.CreateJobs(s.T(), s.db, &j)
	id, err := safecast.ToInt(j.ID)
	if err != nil {
		log.Fatal(err)
	}
	jobID := strconv.Itoa(id)

	p := auth.NewProvider(s.db)
	am := auth.NewAuthMiddleware(p)
	handler := am.RequireTokenJobMatch(s.db)(mockHandler)
	server := s.CreateServer(p)
	defer server.Close()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)

	req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
	if err != nil {
		log.Fatal(err)
	}

	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusUnauthorized, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestRequireTokenJobMatchExpiredJob() {
	expiredJob := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusExpired,
	}

	archivedJob := models.Job{
		ACOID:      uuid.Parse(constants.TestACOID),
		RequestURL: constants.V1Path + constants.EOBExportPath,
		Status:     models.JobStatusExpired,
	}

	postgrestest.CreateJobs(s.T(), s.db, &expiredJob)
	expID, err := safecast.ToInt(expiredJob.ID)
	if err != nil {
		log.Fatal(err)
	}
	expiredJobID := strconv.Itoa(expID)

	postgrestest.CreateJobs(s.T(), s.db, &archivedJob)
	id, err := safecast.ToInt(archivedJob.ID)
	if err != nil {
		log.Fatal(err)
	}
	archivedJobID := strconv.Itoa(id)

	tests := []struct {
		name    string
		jobID   string
		ACOID   string
		errCode int
	}{
		{"Expired Job", expiredJobID, expiredJob.ACOID.String(), http.StatusNotFound},
		{"Archive Job", archivedJobID, archivedJob.ACOID.String(), http.StatusNotFound},
	}

	p := auth.NewProvider(s.db)
	am := auth.NewAuthMiddleware(p)
	handler := am.RequireTokenJobMatch(s.db)(mockHandler)
	server := s.CreateServer(p)
	defer server.Close()

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			s.rr = httptest.NewRecorder()
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("jobID", tt.jobID)
			req, err := http.NewRequest("GET", fmt.Sprintf(constants.ServerPath, server.URL), nil)
			assert.NoError(t, err)
			ad := auth.AuthData{
				ACOID:   tt.ACOID,
				TokenID: uuid.New(),
			}
			ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)
			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
			handler.ServeHTTP(s.rr, req)
			assert.Equal(s.T(), tt.errCode, s.rr.Code)
		})
	}
}

// unit test
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
