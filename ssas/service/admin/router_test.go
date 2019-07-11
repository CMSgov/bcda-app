package admin

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"net/http"
	"net/http/httptest"
	"testing"
)

type RouterTestSuite struct {
	suite.Suite
	router  http.Handler
}

func (s *RouterTestSuite) SetupTest() {
	s.router = NewRouter()
}

func (s *RouterTestSuite) TestPostSystemRoute() {
	req := httptest.NewRequest("POST", "/system", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}