package api

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	acoID = "DBBD1CE1-AE24-435C-807D-ED45953077D3"
)

type RequestsTestSuite struct {
	suite.Suite

	runoutEnabledEnvVar string
	origSvc             models.Service

	db *gorm.DB
}

func TestRequestsTestSuite(t *testing.T) {
	suite.Run(t, new(RequestsTestSuite))
}

func (s *RequestsTestSuite) SetupSuite() {
	s.db = database.GetGORMDbConnection()

	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		s.T().Error(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		s.T().Error(err)
	}

	qc := que.NewClient(pgxpool)
	SetQC(qc)
}

func (s *RequestsTestSuite) SetupTest() {
	s.runoutEnabledEnvVar = os.Getenv("BCDA_ENABLE_RUNOUT")
	s.origSvc = svc
}

func (s *RequestsTestSuite) TearDownSuite() {
	s.db.Close()
}

func (s *RequestsTestSuite) TearDownTest() {
	os.Setenv("BCDA_ENABLE_RUNOUT", s.runoutEnabledEnvVar)
	svc = s.origSvc
}

func (s *RequestsTestSuite) TestRunoutEnabled() {
	os.Setenv("BCDA_ENABLE_RUNOUT", "true")
	qj := []*que.Job{&que.Job{Type: "ProcessJob"}, &que.Job{Type: "ProcessJob"}}
	tests := []struct {
		name string

		errToReturn error
		respCode    int
	}{
		{"Successful", nil, http.StatusAccepted},
		{"No CCLF file found", models.CCLFNotFoundError{}, http.StatusNotFound},
		{"Some other error", errors.New("Some other error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			mockSvc := &models.MockService{}
			var jobs []*que.Job
			if tt.errToReturn == nil {
				jobs = qj
			}

			mockSvc.On("GetQueJobs", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(jobs, tt.errToReturn)
			svc = mockSvc

			req := s.genGroupRequest("runout")
			w := httptest.NewRecorder()
			BulkGroupRequest(w, req)

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
	os.Setenv("BCDA_ENABLE_RUNOUT", "false")
	req := s.genGroupRequest("runout")
	w := httptest.NewRecorder()
	BulkGroupRequest(w, req)

	resp := w.Result()
	body, err := ioutil.ReadAll(resp.Body)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, resp.StatusCode)
	s.Contains(string(body), "Runouts are not enabled")
}

func (s *RequestsTestSuite) genGroupRequest(groupID string) *http.Request {
	req := httptest.NewRequest("GET", "http://bcda.cms.gov", nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("groupId", groupID)

	var aco models.ACO
	s.db.First(&aco, "uuid = ?", acoID)
	ad := auth.AuthData{ACOID: acoID, CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(context.WithValue(req.Context(), auth.AuthDataContextKey, ad))

	return req
}
