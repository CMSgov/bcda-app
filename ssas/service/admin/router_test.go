package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	router http.Handler
}

func (s *RouterTestSuite) SetupTest() {
	s.router = Routes()
}

func (s *RouterTestSuite) TestPostGroupRoute() {
	req := httptest.NewRequest("POST", "/group", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestPostSystemRoute() {
	req := httptest.NewRequest("POST", "/system", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestDeactivateSystemCredentials() {
	req := httptest.NewRequest("DELETE", "/system/1/credentials", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	// TODO Something else here? Not a useful test atm.
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
