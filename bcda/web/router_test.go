package web

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/models"
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

	// Set up
	cmsID := testUtils.RandomHexID()[0:4]
	aco := models.ACO{Name: "TestRegisterSystem", CMSID: &cmsID, UUID: uuid.NewUUID(), ClientID: uuid.New()}
	db := database.Connection

	// Set up a constant token to reference the aco under test
	bearerString := uuid.New()
	token := &jwt.Token{
		Claims: &auth.CommonClaims{
			StandardClaims: jwt.StandardClaims{
				Issuer: "ssas",
			},
			ClientID: uuid.New(),
			SystemID: uuid.New(),
			Data:     fmt.Sprintf(`{"cms_ids":["%s"]}`, cmsID),
		},
		Raw:   uuid.New(),
		Valid: true}

	mock := &auth.MockProvider{}
	mock.On("VerifyToken", bearerString).Return(token, nil)
	mock.On("AuthorizeAccess", token.Raw).Return(nil)
	auth.SetMockProvider(s.T(), mock)

	postgrestest.CreateACO(s.T(), db, aco)
	defer postgrestest.DeleteACO(s.T(), db, aco.UUID)

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

	blackListValues := []*models.Termination{
		{
			TerminationDate: time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			CutoffDate:      time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			BlacklistType:   models.Involuntary,
		},
		nil,
	}

	for _, blacklistValue := range blackListValues {
		for _, config := range configs {
			for _, path := range config.paths {
				s.T().Run(fmt.Sprintf("blacklist-value-%v-%s", blacklistValue, path), func(t *testing.T) {
					aco.TerminationDetails = blacklistValue
					fmt.Println(aco.UUID.String())
					postgrestest.UpdateACO(t, db, aco)
					rr := httptest.NewRecorder()
					req, err := http.NewRequest("GET", path, nil)
					assert.NoError(t, err)
					req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", bearerString))
					config.handler.ServeHTTP(rr, req)

					if aco.Blacklisted() {
						assert.Equal(t, http.StatusForbidden, rr.Code)
						assert.Contains(t, rr.Body.String(), fmt.Sprintf("ACO (CMS_ID: %s) is unauthorized", cmsID))
					} else {
						assert.NotEqual(t, http.StatusForbidden, rr.Code)
					}
				})
			}
		}
	}

	mock.AssertExpectations(s.T())
}

// Verifies that we have the rate limiting handlers in place for the correct environments
func (s *RouterTestSuite) TestRateLimitRoutes() {
	// patterns := []string{"/Group/{groupId}/$export", "/Patient/$export"}

	env := conf.GetEnv("DEPLOYMENT_TARGET")
	defer conf.SetEnv(s.T(), "DEPLOYMENT_TARGET", env)

	tests := []struct {
		target       string
		hasRateLimit bool
	}{
		{"dev", false},
		{"prod", true},
	}

	for _, tt := range tests {
		s.T().Run(tt.target, func(t *testing.T) {
			conf.SetEnv(s.T(), "DEPLOYMENT_TARGET", tt.target)
			conf.SetEnv(s.T(), "VERSION_2_ENDPOINT_ACTIVE", "true")
			router := NewAPIRouter().(chi.Router)
			assert.NotNil(s.T(), router)

			v1Router := getRouterForVersion("v1", router)
			assert.NotNil(t, v1Router)
			v2Router := getRouterForVersion("v2", router)
			assert.NotNil(t, v2Router)

			// Test all requests for all versions of the our API
			for _, versionRouter := range []chi.Router{v1Router, v2Router} {
				for _, ep := range []string{"/Group/{groupId}/$export", "/Patient/$export"} {
					middlewares := getMiddlewareForHandler(ep, versionRouter)
					assert.NotNil(t, middlewares)
					var hasRateLimit bool
					for _, mw := range middlewares {
						assert.NotNil(t, mw)
						// Use the pointer values of the middleware to check if we're
						// using the rate limit functions.
						// If the pointer value of the middleware matches the rate limit function, then
						// we know that the middleware function used is the rate limit function
						if reflect.ValueOf(mw) == reflect.ValueOf(middleware.CheckConcurrentJobs) {
							hasRateLimit = true
						}
					}
					assert.Equal(t, tt.hasRateLimit, hasRateLimit)
				}
			}
		})
	}
}

func getMiddlewareForHandler(pattern string, router chi.Router) chi.Middlewares {
	for _, route := range router.Routes() {
		if route.Pattern == pattern {
			return route.Handlers["GET"].(*chi.ChainHandler).Middlewares
		}
		// Go through all of the children
		if route.SubRoutes != nil {
			middleware := getMiddlewareForHandler(pattern, route.SubRoutes.(chi.Router))
			if middleware != nil {
				return middleware
			}
		}
	}
	// No matches
	return nil
}

// getRouterForVersion retrives the underlying router associated with a particular versioned endpoint
func getRouterForVersion(version string, router chi.Router) chi.Router {
	for _, route := range router.Routes() {
		if route.Pattern == fmt.Sprintf("/api/%s/*", version) {
			return route.SubRoutes.(chi.Router)
		}
		// Go through all of the children
		if route.SubRoutes != nil {
			router := getRouterForVersion(version, route.SubRoutes.(chi.Router))
			if router != nil {
				return router
			}
		}
	}

	return nil
}
func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
