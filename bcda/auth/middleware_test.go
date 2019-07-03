package auth

import (
	"context"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

var mockHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

type MiddlewareTestSuite struct {
	suite.Suite
	server *httptest.Server
	rr     *httptest.ResponseRecorder
	token  string
	ad     AuthData
}

func (s *MiddlewareTestSuite) CreateRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(ParseToken)
	router.With(RequireTokenAuth).Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test router"))
		if err != nil {
			log.Fatal(err)
		}
	})

	return router
}

func (s *MiddlewareTestSuite) SetupSuite() {
	models.InitializeGormModels()
}

func (s *MiddlewareTestSuite) SetupTest() {
	userID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	acoID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	tokenID := "d63205a8-d923-456b-a01b-0992fcb40968"
	s.token, _ = TokenStringWithIDs(tokenID, acoID)
	s.ad = AuthData{
		TokenID: tokenID,
		UserID:  userID,
		ACOID:   acoID,
	}
	s.server = httptest.NewServer(s.CreateRouter())
	s.rr = httptest.NewRecorder()
}

func (s *MiddlewareTestSuite) TearDownTest() {
	s.server.Close()
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthWithInvalidSignature() {
	client := s.server.Client()
	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", badToken))
	resp, err := client.Do(req)

	assert.NotNil(s.T(), resp)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 401, resp.StatusCode)
	assert.Nil(s.T(), err)
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthWithInvalidToken() {
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := RequireTokenAuth(mockHandler)

	tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String()
	tokenString, err := TokenStringWithIDs(tokenID, acoID)
	assert.Nil(s.T(), err)

	token, err := GetProvider().DecodeJWT(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	token.Valid = false

	ad := AuthData{
		ACOID:   acoID,
		TokenID: tokenID,
	}

	ctx := req.Context()
	ctx = context.WithValue(ctx, "token", token)
	ctx = context.WithValue(ctx, "ad", ad)
	req = req.WithContext(ctx)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 401, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthWithValidToken() {
	client := s.server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.token))

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(s.T(), 200, resp.StatusCode)
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthWithEmptyToken() {
	client := s.server.Client()

	// Valid token should return a 200 response
	req, err := http.NewRequest("GET", s.server.URL, nil)
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

func (s *MiddlewareTestSuite) TestRequireTokenJobMatchWithWrongACO() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Failed",
	}

	db.Save(&j)
	jobID := strconv.Itoa(int(j.ID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := RequireTokenJobMatch(mockHandler)

	tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String()
	tokenString, err := TokenStringWithIDs(tokenID, acoID)
	assert.Nil(s.T(), err)

	token, err := GetProvider().DecodeJWT(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	ctx := req.Context()
	// ctx = context.WithValue(ctx, "token", token)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestRequireTokenJobMatchWithRightACO() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Failed",
	}

	db.Save(&j)
	jobID := strconv.Itoa(int(j.ID))
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)

	handler := RequireTokenJobMatch(mockHandler)

	acoID, tokenID := "DBBD1CE1-AE24-435C-807D-ED45953077D3", uuid.NewRandom().String()
	tokenString, err := TokenStringWithIDs(tokenID, acoID)
	assert.Nil(s.T(), err)

	token, err := GetProvider().DecodeJWT(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	ctx := context.WithValue(req.Context(), "token", token)
	ad := AuthData{
		ACOID:   acoID,
		TokenID: tokenID,
	}
	ctx = context.WithValue(ctx, "ad", ad)

	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 200, s.rr.Code)
}

// what is this testing? always returns 404 invalid token?
func (s *MiddlewareTestSuite) TestRequireTokenACOMatchInvalidToken() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Failed",
	}

	db.Save(&j)
	jobID := strconv.Itoa(int(j.ID))

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobID", jobID)

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := RequireTokenJobMatch(mockHandler)

	tokenID, acoID := uuid.NewRandom().String(), uuid.NewRandom().String()
	tokenString, err := TokenStringWithIDs(tokenID, acoID)
	assert.Nil(s.T(), err)

	token, err := GetProvider().DecodeJWT(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	token.Claims = nil

	ctx := req.Context()
	ctx = context.WithValue(ctx, "token", token)
	req = req.WithContext(context.WithValue(ctx, chi.RouteCtxKey, rctx))
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), http.StatusNotFound, s.rr.Code)
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
