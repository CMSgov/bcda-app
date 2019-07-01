package api

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
