package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APITestSuite struct {
	suite.Suite
	rr *httptest.ResponseRecorder
}

func (s *APITestSuite) SetupTest() {
	s.rr = httptest.NewRecorder()
}

func (s *APITestSuite) TestBulkRequestMissingToken() {
	req, err := http.NewRequest("GET", "/api/v1/Patient/$export", nil)
	assert.Nil(s.T(), err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(bulkRequest)

	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusBadRequest, rr.Code)
}

func (s *APITestSuite) TestJobStatusPending() {
	req, err := http.NewRequest("GET", "/api/v1/jobs/1", nil)
	assert.Nil(s.T(), err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobId", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusAccepted, rr.Code)
	assert.Equal(s.T(), "Pending", rr.Header().Get("X-Progress"))
}

func (s *APITestSuite) TestJobStatusInProgress() {
	req, err := http.NewRequest("GET", "/api/v1/jobs/2", nil)
	assert.Nil(s.T(), err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobId", "2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusAccepted, rr.Code)
	assert.Equal(s.T(), "In Progress", rr.Header().Get("X-Progress"))
}

func (s *APITestSuite) TestJobStatusFailed() {
	req, err := http.NewRequest("GET", "/api/v1/jobs/3", nil)
	assert.Nil(s.T(), err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobId", "3")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusInternalServerError, rr.Code)
}

func (s *APITestSuite) TestJobStatusCompleted() {
	req, err := http.NewRequest("GET", "/api/v1/jobs/4", nil)
	assert.Nil(s.T(), err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(jobStatus)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("jobId", "4")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Code)
	assert.Equal(s.T(), "application/json", rr.Header().Get("Content-Type"))
	assert.Equal(s.T(), `{"transactionTime":"2018-09-18T20:38:22.428453Z","request":"/api/v1/Patient/$export","requiresAccessToken":true,"output":[{"type":"ExplanationOfBenefit","url":"http:///data/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson"}],"error":[]}`, rr.Body.String())
}

func (s *APITestSuite) TestServeData() {
	req, err := http.NewRequest("GET", "/api/v1/data/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson", nil)
	assert.Nil(s.T(), err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(serveData)

	handler.ServeHTTP(rr, req)

	assert.Equal(s.T(), http.StatusOK, rr.Code)
	assert.Contains(s.T(), rr.Body.String(), `{"resourceType": "Bundle", "total": 33, "entry": [{"resource": {"status": "active", "diagnosis": [{"diagnosisCodeableConcept": {"coding": [{"system": "http://hl7.org/fhir/sid/icd-9-cm", "code": "2113"}]},`)
}

func (s *APITestSuite) TestGetToken() {}

func TestAPITestSuite(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}
