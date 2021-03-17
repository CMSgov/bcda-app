package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	apiRouter  http.Handler
	dataRouter http.Handler
}

func (s *RouterTestSuite) SetupTest() {
	conf.SetEnv(s.T(), "DEBUG", "true")
	s.apiRouter = NewAPIRouter()
	s.dataRouter = NewDataRouter()
}

func (s *RouterTestSuite) getAPIRoute(route string) *http.Response {
	req := httptest.NewRequest("GET", route, nil)
	rr := httptest.NewRecorder()
	s.apiRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *RouterTestSuite) deleteAPIRoute(route string) *http.Response {
	req := httptest.NewRequest("DELETE", route, nil)
	rr := httptest.NewRecorder()
	s.apiRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *RouterTestSuite) getDataRoute(route string) *http.Response {
	req := httptest.NewRequest("GET", route, nil)
	rr := httptest.NewRecorder()
	s.dataRouter.ServeHTTP(rr, req)
	return rr.Result()
}

func (s *RouterTestSuite) TestDefaultRoute() {
	res := s.getAPIRoute("/")
	assert.Equal(s.T(), http.StatusMovedPermanently, res.StatusCode)
}

func (s *RouterTestSuite) TestUGRoute() {
	res := s.getAPIRoute("/user_guide.html")
	assert.Equal(s.T(), http.StatusMovedPermanently, res.StatusCode)
}

func (s *RouterTestSuite) TestDefaultProdRoute() {
	err := conf.SetEnv(s.T(), "DEPLOYMENT_TARGET", "prod")
	if err != nil {
		s.FailNow("err in setting env var", err)
	}
	// Need a new router because the one in the test setup does not use the environment variable set in this test.
	s.apiRouter = NewAPIRouter()
	res := s.getAPIRoute("/")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	err = conf.UnsetEnv(s.T(), "DEPLOYMENT_TARGET")
	if err != nil {
		s.FailNow("err in setting env var", err)
	}
}

func (s *RouterTestSuite) TestDataRoute() {
	res := s.getDataRoute("/data/test/test.ndjson")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *RouterTestSuite) TestMetadataRoute() {
	res := s.getAPIRoute("/api/v1/metadata")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)

	bytes, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	assert.Nil(s.T(), err)
	var obj map[string]interface{}
	assert.NoError(s.T(), json.Unmarshal(bytes, &obj))
	assert.Equal(s.T(), "CapabilityStatement", obj["resourceType"].(string))
}

func (s *RouterTestSuite) TestHealthRoute() {
	res := s.getAPIRoute("/_health")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestVersionRoute() {
	res := s.getAPIRoute("/_version")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestGroupEndpointDisabled() {
	err := conf.UnsetEnv(s.T(), "BCDA_ENABLE_NEW_GROUP")
	assert.Nil(s.T(), err)
	res := s.getAPIRoute("/api/v1/Groups/new/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
	err = conf.SetEnv(s.T(), "BCDA_ENABLE_GROUP", "true")
	assert.Nil(s.T(), err)
}

func (s *RouterTestSuite) TestEOBExportRoute() {
	res := s.getAPIRoute("/api/v1/Patient/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Patients/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group all
	res = s.getAPIRoute("/api/v1/Group/all/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/all/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group new
	res = s.getAPIRoute("/api/v1/Group/new/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/new/$export?_type=ExplanationOfBenefit")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
}

func (s *RouterTestSuite) TestPatientExportRoute() {
	res := s.getAPIRoute("/api/v1/Patient/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Patients/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group all
	res = s.getAPIRoute("/api/v1/Group/all/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/all/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group new
	res = s.getAPIRoute("/api/v1/Group/new/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/new/$export?_type=Patient")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
}

func (s *RouterTestSuite) TestCoverageExportRoute() {
	res := s.getAPIRoute("/api/v1/Patient/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Patients/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group all
	res = s.getAPIRoute("/api/v1/Group/all/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/all/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)

	// group all
	res = s.getAPIRoute("/api/v1/Group/new/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)

	res = s.getAPIRoute("/api/v1/Groups/new/$export?_type=Coverage")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
}

func (s *RouterTestSuite) TestV2EndpointsDisabled() {
	// Set the V2 endpoints to be off and restart the router so the test router has the correct configuration
	v2Active := conf.GetEnv("VERSION_2_ENDPOINT_ACTIVE")
	defer conf.SetEnv(s.T(), "VERSION_2_ENDPOINT_ACTIVE", v2Active)
	conf.SetEnv(s.T(), "VERSION_2_ENDPOINT_ACTIVE", "false")
	s.apiRouter = NewAPIRouter()

	res := s.getAPIRoute("/api/v2/Patient/$export")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
	res = s.getAPIRoute("/api/v2/Group/all/$export")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
	res = s.getAPIRoute("/api/v2/metadata")
	assert.Equal(s.T(), http.StatusNotFound, res.StatusCode)
}

func (s *RouterTestSuite) TestV2EndpointsEnabled() {
	// Set the V2 endpoints to be on and restart the router so the test router has the correct configuration
	v2Active := conf.GetEnv("VERSION_2_ENDPOINT_ACTIVE")
	defer conf.SetEnv(s.T(), "VERSION_2_ENDPOINT_ACTIVE", v2Active)
	conf.SetEnv(s.T(), "VERSION_2_ENDPOINT_ACTIVE", "true")
	s.apiRouter = NewAPIRouter()

	res := s.getAPIRoute("/api/v2/Patient/$export")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
	res = s.getAPIRoute("/api/v2/Group/all/$export")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
	res = s.getAPIRoute("/api/v2/metadata")
	assert.Equal(s.T(), http.StatusOK, res.StatusCode)
}

func (s *RouterTestSuite) TestJobStatusRoute() {
	res := s.getAPIRoute("/api/v1/jobs/1")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *RouterTestSuite) TestDeleteJobRoute() {
	res := s.deleteAPIRoute("/api/v1/jobs/1")
	assert.Equal(s.T(), http.StatusUnauthorized, res.StatusCode)
}

func (s *RouterTestSuite) TestHTTPServerRedirect() {
	router := NewHTTPRouter()

	// Redirect GET http requests to https
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res := w.Result()

	assert.Nil(s.T(), err, "redirect GET http to https")
	assert.Equal(s.T(), http.StatusMovedPermanently, res.StatusCode, "http to https redirect return correct status code")
	assert.Equal(s.T(), "close", res.Header.Get("Connection"), "http to https redirect sets 'connection: close' header")
	assert.Contains(s.T(), res.Header.Get("Location"), "https://", "location response header contains 'https://'")

	// Only respond to GET requests
	req, err = http.NewRequest("POST", "/", nil)
	if err != nil {
		log.Fatal(err)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	res = w.Result()

	assert.Nil(s.T(), err, "redirect POST http to https")
	assert.Equal(s.T(), http.StatusMethodNotAllowed, res.StatusCode, "http to https redirect rejects POST requests")
}

// TestBlacklistedACOs ensures that we return 403 FORBIDDEN when a call is made from a blacklisted ACO.
func (s *RouterTestSuite) TestBlacklistedACO() {
	// Use a new router to ensure that v2 endpoints are active
	v2Active := conf.GetEnv("VERSION_2_ENDPOINT_ACTIVE")
	defer conf.SetEnv(s.T(), "VERSION_2_ENDPOINT_ACTIVE", v2Active)
	conf.SetEnv(s.T(), "VERSION_2_ENDPOINT_ACTIVE", "true")
	apiRouter := NewAPIRouter()

	db := database.GetDbConnection()
	defer db.Close()

	p := auth.GetProvider()
	cmsID := testUtils.RandomHexID()[0:4]
	id := uuid.NewRandom()
	aco := &models.ACO{Name: "TestRegisterSystem", CMSID: &cmsID, UUID: id, ClientID: id.String()}
	postgrestest.CreateACO(s.T(), db, *aco)
	acoUUID := aco.UUID

	defer postgrestest.DeleteACO(s.T(), db, acoUUID)
	c, err := p.RegisterSystem(acoUUID.String(), "", "")
	s.NoError(err)
	token, err := p.MakeAccessToken(c)
	s.NoError(err)

	configs := []struct {
		handler http.Handler
		paths   []string
	}{
		{apiRouter, []string{"/api/v1/Patient/$export", "/api/v1/Group/all/$export",
			"/api/v2/Patient/$export", "/api/v2/Group/all/$export",
			"/api/v1/jobs/1"}},
		{s.dataRouter, []string{"/data/test/test.ndjson"}},
		{NewAuthRouter(), []string{"/auth/welcome"}},
	}

	blackListValues := []models.Termination{
		{
			TerminationDate: time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			CutoffDate:      time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			BlacklistType:   0,
		},
		{},
	}

	for _, blacklistValue := range blackListValues {
		for _, config := range configs {
			for _, path := range config.paths {
				s.T().Run(fmt.Sprintf("blacklist-value-%v-%s", blacklistValue, path), func(t *testing.T) {
					aco.TerminationDetails = &blacklistValue
					postgrestest.UpdateACO(t, db, *aco)
					rr := httptest.NewRecorder()
					req, err := http.NewRequest("GET", path, nil)
					assert.NoError(t, err)
					req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
					config.handler.ServeHTTP(rr, req)

					if aco.BlacklistedFunc() {
						assert.Equal(t, http.StatusForbidden, rr.Code)
						assert.Contains(t, rr.Body.String(), fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", cmsID))
					} else {
						assert.NotEqual(t, http.StatusForbidden, rr.Code)
					}
				})
			}
		}
	}
}
func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
