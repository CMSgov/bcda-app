package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

    configuration "github.com/CMSgov/bcda-app/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type AuthRouterTestSuite struct {
	suite.Suite
	authRouter http.Handler
}

func (s *AuthRouterTestSuite) SetupTest() {
	configuration.SetEnv(&testing.T{}, "DEBUG", "true")
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

func TestAuthRouterTestSuite(t *testing.T) {
	suite.Run(t, new(AuthRouterTestSuite))
}
