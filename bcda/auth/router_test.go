package auth

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type SSASRouterTestSuite struct {
	suite.Suite
	authRouter http.Handler
}

func (s *SSASRouterTestSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
	s.authRouter = NewAuthRouter()
}

func (s *SSASRouterTestSuite) reqAuthRoute(verb string, route string, body io.Reader) *http.Response {
	req := httptest.NewRequest(strings.ToUpper(verb), route, body)
	rr := httptest.NewRecorder()
	s.authRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *SSASRouterTestSuite) TestAuthTokenRoute() {
	res := s.reqAuthRoute("POST", "/auth/token", nil)
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func TestSSASRouterTestSuite(t *testing.T) {
	suite.Run(t, new(SSASRouterTestSuite))
}