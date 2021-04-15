package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AlrTestSuite struct {
	suite.Suite

	db    *sql.DB
	acoID uuid.UUID
}

func TestAlrTestSuite(t *testing.T) {
	suite.Run(t, new(AlrTestSuite))
}

func (s *AlrTestSuite) SetupSuite() {
	// See testdata/acos.yml
	s.acoID = uuid.Parse("a7ff9610-0977-4a90-867e-f6b2b4c8b6a8")
	s.db, _ = databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	tf, err := testfixtures.New(
		testfixtures.Database(s.db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("testdata/"),
	)

	if err != nil {
		assert.FailNowf(s.T(), "Failed to setup test fixtures", err.Error())
	}
	if err := tf.Load(); err != nil {
		assert.FailNowf(s.T(), "Failed to load test fixtures", err.Error())
	}

	enqueuer := &queueing.MockEnqueuer{}
	enqueuer.On("AddAlrJob", mock.Anything, mock.Anything).Return(nil)

}

func (s *AlrTestSuite) TestAlrRequest() {
	enqueuer := &queueing.MockEnqueuer{}
	enqueuer.On("AddAlrJob", mock.Anything, mock.Anything).Return(nil)

	h := newHandler([]string{"Patient", "Observation"}, "v1", s.db)
	h.Enq = enqueuer

	// Set up request with the correct context scoped values
	req := httptest.NewRequest("GET",
		"http://bcda.cms.gov/api/v1/Patient/$export?type=Patient,Observation&_typeFilter=Patient?profile=ALR,Observation?profile=ALR",
		nil)
	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)
	ctx = middleware.NewRequestParametersContext(ctx, middleware.RequestParameters{ResourceTypes: []string{"Patient", "Observation"}})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.alrRequest(w, req, service.DefaultRequest)
	assert.Equal(s.T(), http.StatusAccepted, w.Result().StatusCode)

	// Reset the recorder to allow us to verify the runout response
	w = httptest.NewRecorder()
	h.alrRequest(w, req, service.Runout)
	assert.Equal(s.T(), http.StatusAccepted, w.Result().StatusCode)

	assert.True(s.T(), enqueuer.AssertNumberOfCalls(s.T(), "AddAlrJob", 2), "We should've enqueued two ALR jobs")
}

func (s *AlrTestSuite) TestIsALRRequest() {
	tests := []struct {
		qp    string
		isALR bool
	}{
		{"_type=Patient,Observation&_typeFilter=Patient?profile=ALR,Observation?profile=ALR", true},
		{"_type=Observation,Patient&_typeFilter=Patient?profile=ALR,Observation?profile=ALR", true},
		{"_type=Patient,Observation&_typeFilter=Patient?profile=ALR,Observation?profile=ALR", true},
		{"_type=Observation,Patient&_typeFilter=Observation?profile=ALR,Patient?profile=ALR", true},
		{"", false},
		{"_type=Patient", false},
		{"_typeFilter=Patient?profile=BMS", false},
	}

	for _, tt := range tests {
		s.T().Run(tt.qp, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://bcda.com?"+tt.qp, nil)
			assert.Equal(t, tt.isALR, isALRRequest(r))
		})
	}
}
