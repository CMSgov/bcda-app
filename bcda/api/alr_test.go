package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/bcda/service"
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
	// UUID for ACO Dev
	s.acoID = uuid.Parse("0c527d2e-2e8a-4808-b11d-0fa06baf8254")
	s.db = database.Connection

}

func (s *AlrTestSuite) TestAlrRequest() {
	enqueuer := &queueing.MockEnqueuer{}
	enqueuer.On("AddAlrJob", mock.Anything, mock.Anything).Return(nil)

	resourceMap := map[string]service.DataType{
		"Patient":     {Adjudicated: true},
		"Observation": {Adjudicated: true},
	}

	h := newHandler(resourceMap, "v1/fhir", "v1", s.db)
	h.Enq = enqueuer

	// Set up request with the correct context scoped values
	req := httptest.NewRequest("GET",
		"http://bcda.cms.gov/api/v1/alr/$export",
		nil)
	aco := postgrestest.GetACOByUUID(s.T(), s.db, s.acoID)
	ad := auth.AuthData{ACOID: s.acoID.String(), CMSID: *aco.CMSID, TokenID: uuid.NewRandom().String()}

	ctx := context.WithValue(req.Context(), auth.AuthDataContextKey, ad)
	ctx = middleware.NewRequestParametersContext(ctx, middleware.RequestParameters{ResourceTypes: []string{"Patient", "Observation"}})
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.alrRequest(w, req)
	assert.Equal(s.T(), http.StatusAccepted, w.Result().StatusCode)

	assert.True(s.T(), enqueuer.AssertNumberOfCalls(s.T(), "AddAlrJob", 1), "We should've enqueued one ALR jobs")
}
