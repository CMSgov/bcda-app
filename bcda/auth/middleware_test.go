package auth_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var mockHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

type MiddlewareTestSuite struct {
	testUtils.AuthTestSuite
	server *httptest.Server
	rr     *httptest.ResponseRecorder
	token  string
}

func (s *MiddlewareTestSuite) CreateRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(auth.ParseToken)
	router.With(auth.RequireTokenAuth).Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("Test router"))
		if err != nil {
			log.Fatal(err)
		}
	})

	return router
}

func (s *MiddlewareTestSuite) SetupTest() {
	s.SetupAuthBackend()
	token, err := s.AuthBackend.GenerateTokenString(
		"82503A18-BF3B-436D-BA7B-BAE09B7FFD2F", "DBBD1CE1-AE24-435C-807D-ED45953077D3")
	if err != nil {
		log.Fatal(err)
	}
	s.token = token
	s.server = httptest.NewServer(s.CreateRouter())
	s.rr = httptest.NewRecorder()
}

func (s *MiddlewareTestSuite) TearDownTest() {
	for _, f := range s.TmpFiles {
		os.Remove(f)
	}
	s.server.Close()
}

func (s *MiddlewareTestSuite) TestParseTokenInvalidSigning() {
	client := s.server.Client()
	badToken := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCIsImtpZCI6ImlUcVhYSTB6YkFuSkNLRGFvYmZoa00xZi02ck1TcFRmeVpNUnBfMnRLSTgifQ.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.cJOP_w-hBqnyTsBm3T6lOE5WpcHaAkLuQGAs1QO-lg2eWs8yyGW8p9WagGjxgvx7h9X72H7pXmXqej3GdlVbFmhuzj45A9SXDOAHZ7bJXwM1VidcPi7ZcrsMSCtP1hiN"
	s.token = badToken

	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.token))
	resp, err := client.Do(req)

	fmt.Println(resp.StatusCode)
	assert.Equal(s.T(), 401, resp.StatusCode)
	assert.Nil(s.T(), err)
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthInvalid() {
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenAuth(mockHandler)

	acoID, userID := uuid.NewRandom().String(), uuid.NewRandom().String()
	tokenString, err := s.AuthBackend.GenerateTokenString(userID, acoID)
	assert.Nil(s.T(), err)

	token, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	token.Valid = false

	ctx := req.Context()
	ctx = context.WithValue(ctx, "token", token)
	req = req.WithContext(ctx)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 401, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthBlackListed() {
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenAuth(mockHandler)

	// Blacklisted Token test
	db := database.GetGORMDbConnection()
	userID := "EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73"
	var user models.User
	if db.Find(&user, "UUID = ?", userID).RecordNotFound() {
		assert.NotNil(s.T(), errors.New("Unable to find User"))
	}
	_, tokenString, err := s.AuthBackend.CreateToken(user)
	assert.Nil(s.T(), err)
	// Convert tokenString to a jwtToken
	jwtToken, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)

	_ = s.AuthBackend.RevokeToken(tokenString)

	// just to be sure it is blacklisted
	blacklisted := s.AuthBackend.IsBlacklisted(jwtToken)
	assert.Nil(s.T(), err)
	assert.True(s.T(), blacklisted)

	// The actual test
	ctx := req.Context()
	ctx = context.WithValue(ctx, "token", jwtToken)
	req = req.WithContext(ctx)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 401, s.rr.Code)

}

func (s *MiddlewareTestSuite) TestRequireTokenAuthValid() {
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

func (s *MiddlewareTestSuite) TestRequireTokenAuthEmpty() {
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

func (s *MiddlewareTestSuite) TestRequireTokenACOMatchNotEqual() {
	req, err := http.NewRequest("GET", s.server.URL+"/data/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson", nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenACOMatch(mockHandler)

	acoID, userID := uuid.NewRandom().String(), uuid.NewRandom().String()
	tokenString, err := s.AuthBackend.GenerateTokenString(userID, acoID)
	assert.Nil(s.T(), err)

	token, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	ctx := req.Context()
	ctx = context.WithValue(ctx, "token", token)
	req = req.WithContext(ctx)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 404, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestRequireTokenACOMatchEqual() {
	req, err := http.NewRequest("GET", s.server.URL+"/data/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson", nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenACOMatch(mockHandler)

	acoID, userID := "DBBD1CE1-AE24-435C-807D-ED45953077D3", uuid.NewRandom().String()
	tokenString, err := s.AuthBackend.GenerateTokenString(userID, acoID)
	assert.Nil(s.T(), err)

	token, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	ctx := req.Context()
	ctx = context.WithValue(ctx, "token", token)
	req = req.WithContext(ctx)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 200, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestRequireTokenACOMatchNoClaims() {
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenACOMatch(mockHandler)

	acoID, userID := uuid.NewRandom().String(), uuid.NewRandom().String()
	tokenString, err := s.AuthBackend.GenerateTokenString(userID, acoID)
	assert.Nil(s.T(), err)

	token, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)
	token.Claims = nil

	ctx := req.Context()
	ctx = context.WithValue(ctx, "token", token)
	req = req.WithContext(ctx)
	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 500, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestRequireTokenACOMatchClaims() {
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	handler := auth.RequireTokenACOMatch(mockHandler)

	handler.ServeHTTP(s.rr, req)
	assert.Equal(s.T(), 401, s.rr.Code)
}

func (s *MiddlewareTestSuite) TestClaimsFromToken() {
	acoID, userID := uuid.NewRandom().String(), uuid.NewRandom().String()
	tokenString, err := s.AuthBackend.GenerateTokenString(userID, acoID)
	assert.Nil(s.T(), err)

	token, err := s.AuthBackend.GetJWToken(tokenString)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), token)

	// Test good claims
	claims, err := auth.ClaimsFromToken(token)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), claims)

	// Test invalid claims
	token.Claims = nil
	badclaims, err := auth.ClaimsFromToken(token)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), jwt.MapClaims{}, badclaims)
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
