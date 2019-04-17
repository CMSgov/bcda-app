package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type AuthRouterTestSuite struct {
	suite.Suite
	authRouter http.Handler
}

func (s *AuthRouterTestSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
	s.authRouter = NewAuthRouter()
}

func (s *AuthRouterTestSuite) reqAuthRoute(verb string, route string, body io.Reader) *http.Response {
	req := httptest.NewRequest(strings.ToUpper(verb), route, body)
	rr := httptest.NewRecorder()
	s.authRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *AuthRouterTestSuite) TestAuthTokenRoute() {
	res := s.reqAuthRoute("POST", "/auth/token", nil)
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *AuthRouterTestSuite) TestGetAuthGroupRoute() {
	res := s.reqAuthRoute("GET", "/auth/group", nil)
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *AuthRouterTestSuite) TestPostAuthGroupRoute() {
	res := s.reqAuthRoute("POST", "/auth/group", nil)
	assert.Equal(s.T(), http.StatusNotImplemented, res.StatusCode)
}

func (s *AuthRouterTestSuite) TestPutAuthGroupRoute() {
	res := s.reqAuthRoute("PUT", "/auth/group", nil)
	assert.Equal(s.T(), http.StatusNotImplemented, res.StatusCode)
}

func (s *AuthRouterTestSuite) TestDeleteAuthGroupRoute() {
	res := s.reqAuthRoute("DELETE", "/auth/group", nil)
	assert.Equal(s.T(), http.StatusNotImplemented, res.StatusCode)
}
