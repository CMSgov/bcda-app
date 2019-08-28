package admin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/CMSgov/bcda-app/ssas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	router           http.Handler
	clientID, secret string
}

func (s *RouterTestSuite) SetupTest() {
	s.router = routes()
	s.clientID = "31e029ef-0e97-47f8-873c-0e8b7e7f99bf"
	s.secret = "nbZ5oAnTlzyzeep46bL4qDGGuidXuYxs3xknVWBKjTI=:9s/Tnqvs8M7GN6VjGkLhCgjmS59r6TaVguos8dKV9lGqC1gVG8ywZVEpDMkdwOaj8GoNe4TU3jS+OZsK3kTfEQ=="
}

func (s *RouterTestSuite) TestRevokeToken() {
	req := httptest.NewRequest("DELETE", "/token/abc-123", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestPostGroup() {
	req := httptest.NewRequest("POST", "/group", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestGetGroup() {
	req := httptest.NewRequest("GET", "/group", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestPutGroup() {
	req := httptest.NewRequest("PUT", "/group/1", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestDeleteGroup() {
	req := httptest.NewRequest("DELETE", "/group/101", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestPostSystem() {
	req := httptest.NewRequest("POST", "/system", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestDeactivateSystemCredentials() {
	db := ssas.GetGORMDbConnection()
	defer db.Close()
	group := ssas.Group{GroupID: "delete-system-credentials-test-group"}
	db.Create(&group)
	system := ssas.System{GroupID: group.GroupID, ClientID: "delete-system-credentials-test-system"}
	db.Create(&system)
	systemID := strconv.FormatUint(uint64(system.ID), 10)

	req := httptest.NewRequest("DELETE", "/system/"+systemID+"/credentials", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	err := ssas.CleanDatabase(group)
	assert.Nil(s.T(), err)
}

func (s *RouterTestSuite) TestPutSystemCredentials() {
	db := ssas.GetGORMDbConnection()
	defer db.Close()
	group := ssas.Group{GroupID: "put-system-credentials-test-group"}
	db.Create(&group)
	system := ssas.System{GroupID: group.GroupID, ClientID: "put-system-credentials-test-system"}
	db.Create(&system)
	systemID := strconv.FormatUint(uint64(system.ID), 10)

	req := httptest.NewRequest("PUT", "/system/"+systemID+"/credentials", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s:%s", s.clientID, s.secret))
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusCreated, res.StatusCode)

	err := ssas.CleanDatabase(group)
	assert.Nil(s.T(), err)
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
