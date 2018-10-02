package auth_test

import (
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

var mockHandler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {}

type MiddlewareTestSuite struct {
	testUtils.AuthTestSuite
	server *httptest.Server
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
}

func (s *MiddlewareTestSuite) TearDownTest() {
	for _, f := range s.TmpFiles {
		os.Remove(f)
	}
	s.server.Close()
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthReturnVal() {
	result := auth.RequireTokenAuth(mockHandler)
	assert.IsType(s.T(), result, mockHandler)
}

func (s *MiddlewareTestSuite) TestRequireTokenAuthValidToken() {
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

func (s *MiddlewareTestSuite) TestRequireTokenAuthInvalidToken() {
	client := s.server.Client()

	// Invalid token should result in 401 Unauthorized response
	req, err := http.NewRequest("GET", s.server.URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Authorization", "invalidtoken")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	assert.Equal(s.T(), 401, resp.StatusCode)
}

func TestMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(MiddlewareTestSuite))
}
