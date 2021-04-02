package auth_test

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"net/http/httptest"
// 	"strconv"
// 	"testing"

// 	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
// 	"github.com/CMSgov/bcda-app/bcda/testUtils"

// 	"github.com/go-chi/chi"
// 	"github.com/pborman/uuid"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/suite"

// 	"github.com/CMSgov/bcda-app/bcda/auth"
// 	"github.com/CMSgov/bcda-app/bcda/database"
// 	"github.com/CMSgov/bcda-app/bcda/models"
// )

// var mockHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

// type MiddlewareTestSuite struct {
// 	suite.Suite
// 	server *httptest.Server
// 	rr     *httptest.ResponseRecorder
// 	token  string
// 	ad     auth.AuthData
// }

// func (s *MiddlewareTestSuite) CreateRouter() http.Handler {
// 	router := chi.NewRouter()
// 	router.Use(auth.ParseToken)
// 	router.With(auth.RequireTokenAuth).Get("/", func(w http.ResponseWriter, r *http.Request) {
// 		_, err := w.Write([]byte("Test router"))
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 	})

// 	return router
// }

// func (s *MiddlewareTestSuite) SetupTest() {
// 	cmsID := "A9995"
// 	acoID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
// 	tokenID := "d63205a8-d923-456b-a01b-0992fcb40968"
// 	s.token, _ = auth.TokenStringWithIDs(tokenID, acoID)
// 	s.ad = auth.AuthData{
// 		TokenID: tokenID,
// 		CMSID:   cmsID,
// 		ACOID:   acoID,
// 	}
// 	s.server = httptest.NewServer(s.CreateRouter())
// 	s.rr = httptest.NewRecorder()
// }

// func (s *MiddlewareTestSuite) TearDownTest() {
// 	s.server.Close()
// }

// func (s *MiddlewareTestSuite) TestRequireTokenAuthWithInvalidSignature() {
// 	client := s.server.Client()
// 	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"

// 	req, err := http.NewRequest("GET", s.server.URL, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", badToken))
// 	resp, err := client.Do(req)

// 	assert.NotNil(s.T(), resp)
// 	assert.Nil(s.T(), err)
// 	assert.Equal(s.T(), 401, resp.StatusCode)
// 	assert.Nil(s.T(), err)
// }

// func (s *MiddlewareTestSuite) TestRequireTokenAuthWithInvalidToken() {
// 	req, err := http.NewRequest("GET", s.server.URL, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	handler := auth.RequireTokenAuth(mockHandler)

// 	tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String()
// 	tokenString, err := auth.TokenStringWithIDs(tokenID, acoID)
// 	assert.Nil(s.T(), err)

// 	token, err := auth.GetProvider().VerifyToken(tokenString)
// 	assert.Nil(s.T(), err)
// 	assert.NotNil(s.T(), token)
// 	token.Valid = false

// 	ad := auth.AuthData{
// 		ACOID:   acoID,
// 		TokenID: tokenID,
// 	}

// 	ctx := req.Context()
// 	ctx = context.WithValue(ctx, auth.TokenContextKey, token)
// 	ctx = context.WithValue(ctx, auth.AuthDataContextKey, ad)
// 	req = req.WithContext(ctx)
// 	handler.ServeHTTP(s.rr, req)
// 	assert.Equal(s.T(), 401, s.rr.Code)
// }

// func (s *MiddlewareTestSuite) TestRequireTokenAuthWithValidToken() {
// 	client := s.server.Client()

// 	// Valid token should return a 200 response
// 	req, err := http.NewRequest("GET", s.server.URL, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.token))

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	assert.Equal(s.T(), 200, resp.StatusCode)
// }

// func (s *MiddlewareTestSuite) TestRequireTokenAuthWithEmptyToken() {
// 	client := s.server.Client()

// 	// Valid token should return a 200 response
// 	req, err := http.NewRequest("GET", s.server.URL, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	req.Header.Add("Authorization", "")

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	assert.Equal(s.T(), 401, resp.StatusCode)
// }

// func (s *MiddlewareTestSuite) TestRequireTokenJobMatchWithMistmatchingData() {
// 	db := database.Connection

// 	j := models.Job{
// 		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
// 		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
// 		Status:     models.JobStatusFailed,
// 	}

// 	postgrestest.CreateJobs(s.T(), db, &j)
// 	jobID := strconv.Itoa(int(j.ID))

// 	tests := []struct {
// 		name  string
// 		jobID string
// 		ACOID string
// 	}{
// 		{"Invalid JobID", "someNonNumericInput", j.ACOID.String()},
// 		{"Mismatching JobID", "0", j.ACOID.String()},
// 		{"Mismatching ACOID", jobID, uuid.New()},
// 	}

// 	handler := auth.RequireTokenJobMatch(mockHandler)

// 	for _, tt := range tests {
// 		s.T().Run(tt.name, func(t *testing.T) {
// 			s.rr = httptest.NewRecorder()

// 			rctx := chi.NewRouteContext()
// 			rctx.URLParams.Add("jobID", tt.jobID)

// 			req, err := http.NewRequest("GET", s.server.URL, nil)
// 			assert.NoError(t, err)

// 			tokenID := uuid.New()
// 			tokenString, err := auth.TokenStringWithIDs(tokenID, tt.ACOID)
// 			assert.NoError(t, err)

// 			token, err := auth.GetProvider().VerifyToken(tokenString)
// 			assert.NoError(t, err)
// 			assert.NotNil(t, token)

// 			ctx := context.WithValue(req.Context(), auth.TokenContextKey, token)
// 			ad := auth.AuthData{
// 				ACOID:   tt.ACOID,
// 				TokenID: tokenID,
// 			}
// 			ctx = context.WithValue(ctx, auth.AuthDataContextKey, ad)

// 			req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
// 			handler.ServeHTTP(s.rr, req)
// 			assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)
// 		})
// 	}
// }

// func (s *MiddlewareTestSuite) TestRequireTokenJobMatchWithRightACO() {
// 	db := database.Connection

// 	j := models.Job{
// 		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
// 		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
// 		Status:     models.JobStatusFailed,
// 	}
// 	postgrestest.CreateJobs(s.T(), db, &j)
// 	jobID := strconv.Itoa(int(j.ID))

// 	req, err := http.NewRequest("GET", s.server.URL, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	rctx := chi.NewRouteContext()
// 	rctx.URLParams.Add("jobID", jobID)

// 	handler := auth.RequireTokenJobMatch(mockHandler)

// 	acoID, tokenID := "DBBD1CE1-AE24-435C-807D-ED45953077D3", uuid.NewRandom().String()
// 	tokenString, err := auth.TokenStringWithIDs(tokenID, acoID)
// 	assert.Nil(s.T(), err)

// 	token, err := auth.GetProvider().VerifyToken(tokenString)
// 	assert.Nil(s.T(), err)
// 	assert.NotNil(s.T(), token)

// 	ctx := context.WithValue(req.Context(), auth.TokenContextKey, token)
// 	ad := auth.AuthData{
// 		ACOID:   acoID,
// 		TokenID: tokenID,
// 	}
// 	ctx = context.WithValue(ctx, auth.AuthDataContextKey, ad)

// 	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
// 	handler.ServeHTTP(s.rr, req)
// 	assert.Equal(s.T(), 200, s.rr.Code)
// }

// // TestRequireTokenACOMatchInvalidToken validates that we return a 404
// // If the caller does not supply the auth data
// func (s *MiddlewareTestSuite) TestRequireTokenACOMatchInvalidToken() {
// 	db := database.Connection

// 	j := models.Job{
// 		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
// 		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
// 		Status:     models.JobStatusFailed,
// 	}

// 	postgrestest.CreateJobs(s.T(), db, &j)
// 	jobID := strconv.Itoa(int(j.ID))

// 	rctx := chi.NewRouteContext()
// 	rctx.URLParams.Add("jobID", jobID)

// 	req, err := http.NewRequest("GET", s.server.URL, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	handler := auth.RequireTokenJobMatch(mockHandler)

// 	tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String()
// 	tokenString, err := auth.TokenStringWithIDs(tokenID, acoID)
// 	assert.Nil(s.T(), err)

// 	token, err := auth.GetProvider().VerifyToken(tokenString)
// 	assert.Nil(s.T(), err)
// 	assert.NotNil(s.T(), token)
// 	token.Claims = nil

// 	ctx := req.Context()
// 	ctx = context.WithValue(ctx, auth.TokenContextKey, token)
// 	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
// 	handler.ServeHTTP(s.rr, req)
// 	assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)
// }

// func (s *MiddlewareTestSuite) TestCheckBlacklist() {
// 	blacklisted := testUtils.RandomHexID()[0:4]
// 	notBlacklisted := testUtils.RandomHexID()[0:4]

// 	handler := auth.CheckBlacklist(mockHandler)
// 	tests := []struct {
// 		name            string
// 		ad              *auth.AuthData
// 		expectedCode    int
// 		expectedMessage string
// 	}{
// 		{"No auth data found", nil, http.StatusNotFound, "AuthData not found"},
// 		{"Blacklisted ACO", &auth.AuthData{CMSID: blacklisted, Blacklisted: true}, http.StatusForbidden,
// 			fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", blacklisted)},
// 		{"Non-blacklisted ACO", &auth.AuthData{CMSID: notBlacklisted, Blacklisted: false}, http.StatusOK, ""},
// 	}

// 	for _, tt := range tests {
// 		s.T().Run(tt.name, func(t *testing.T) {
// 			rr := httptest.NewRecorder()
// 			ctx := context.Background()
// 			if tt.ad != nil {
// 				ctx = context.WithValue(ctx, auth.AuthDataContextKey, *tt.ad)
// 			}
// 			req, err := http.NewRequestWithContext(ctx, "GET", "", nil)
// 			assert.NoError(t, err)
// 			handler.ServeHTTP(rr, req)

// 			assert.Equal(t, tt.expectedCode, rr.Code)
// 			assert.Contains(t, rr.Body.String(), tt.expectedMessage)
// 		})
// 	}

// }

// func TestMiddlewareTestSuite(t *testing.T) {
// 	suite.Run(t, new(MiddlewareTestSuite))
// }
