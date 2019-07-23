package admin

import (
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

func (s *RouterTestSuite) TestDeleteGroup() {
	req := httptest.NewRequest("DELETE", "/group/101", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestPostSystem() {
	req := httptest.NewRequest("POST", "/system", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusBadRequest, res.StatusCode)
}

func (s *RouterTestSuite) TestDeactivateSystemCredentials() {
	db := ssas.GetGORMDbConnection()
	group := ssas.Group{GroupID: "delete-system-credentials-test-group"}
	db.Create(&group)
	system := ssas.System{GroupID: group.GroupID, ClientID: "delete-system-credentials-test-system"}
	db.Create(&system)
	systemID := strconv.FormatUint(uint64(system.ID), 10)

	req := httptest.NewRequest("DELETE", "/system/"+systemID+"/credentials", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	_ = ssas.CleanDatabase(group)
}

func (s *RouterTestSuite) TestPutSystemCredentials() {
	db := ssas.GetGORMDbConnection()
	group := ssas.Group{GroupID: "put-system-credentials-test-group"}
	db.Create(&group)
	system := ssas.System{GroupID: group.GroupID, ClientID: "put-system-credentials-test-system"}
	db.Create(&system)
	systemID := strconv.FormatUint(uint64(system.ID), 10)

	req := httptest.NewRequest("PUT", "/system/"+systemID+"/credentials", nil)
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	res := rr.Result()
	assert.Equal(s.T(), http.StatusCreated, res.StatusCode)

	_ = ssas.CleanDatabase(group)
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
