package api

import (
	"context"
	"database/sql"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RequestsTestSuite struct {
	suite.Suite

	runoutEnabledEnvVar string

	db *sql.DB

	acoID uuid.UUID
}

func TestRequestsTestSuite(t *testing.T) {
	suite.Run(t, new(RequestsTestSuite))
}

func (s *RequestsTestSuite) SetupSuite() {
	s.db = database.Connection

	// Create an ACO for us to test with
	s.acoID = uuid.NewUUID()
	cmsID := "ZYXWV"

	postgrestest.CreateACO(s.T(), s.db, models.ACO{UUID: s.acoID, CMSID: &cmsID})
}

func (s *RequestsTestSuite) SetupTest() {
	s.runoutEnabledEnvVar = conf.GetEnv("BCDA_ENABLE_RUNOUT")
}

func (s *RequestsTestSuite) TearDownSuite() {
	postgrestest.DeleteACO(s.T(), s.db, s.acoID)
}

func (s *RequestsTestSuite) TearDownTest() {
	conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", s.runoutEnabledEnvVar)
}

func (s *RequestsTestSuite) TestRunoutEnabled() {
	conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", "true")
	qj := []*models.JobEnqueueArgs{}
	tests := []struct {
		name string

		errToReturn error
		respCode    int
	}{
		{"Successful", nil, http.StatusAccepted},
		{"No CCLF file found", service.CCLFNotFoundError{}, http.StatusNotFound},
		{"Some other error", errors.New("Some other error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &service.MockService{}
			var jobs []*models.JobEnqueueArgs
			if tt.errToReturn == nil {
				jobs = qj
			}

			mockSvc.On("GetQueJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(jobs, tt.errToReturn)
			h := NewHandler([]string{"ExplanationOfBenefit", "Coverage", "Patient"}, "/v1/fhir")
			h.Svc = mockSvc

			req := s.genGroupRequest("runout")
			w := httptest.NewRecorder()
			h.BulkGroupRequest(w, req)

			resp := w.Result()
			body, err := ioutil.ReadAll(resp.Body)

			assert.NoError(t, err)
			assert.Equal(t, tt.respCode, resp.StatusCode)
			if tt.errToReturn == nil {
				assert.NotEmpty(t, resp.Header.Get("Content-Location"))
			} else {
				assert.Contains(t, string(body), tt.errToReturn.Error())
			}
		})
	}
}

func (s *RequestsTestSuite) TestRunoutDisabled() {
	conf.SetEnv(s.T(), "BCDA_ENABLE_RUNOUT", "false")
	req := s.genGroupRequest("runout")
	w := httptest.NewRecorder()
	h := &Handler{}
	h.BulkGroupRequest(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Contains(string(body), "Invalid group ID")
}

func (s *RequestsTestSuite) genGroupRequest(groupID string) *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov/api/v1/Group/$export", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("groupId", groupID)

	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.AuthDataContextKey, ad)
	ctx = middleware.NewRequestParametersContext(ctx, middleware.RequestParameters{})

	req = req.WithContext(ctx)

	return req
}
