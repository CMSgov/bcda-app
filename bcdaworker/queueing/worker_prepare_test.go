package queueing

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/log"
	cm "github.com/CMSgov/bcda-app/middleware"
	"github.com/go-testfixtures/testfixtures/v3"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	fhirModels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/mock"
)

type PrepareWorkerIntegrationTestSuite struct {
	suite.Suite
	r    models.Repository
	db   *sql.DB
	pool *pgxv5Pool.Pool
	ctx  context.Context
}

func TestCleanupTestSuite(t *testing.T) {
	suite.Run(t, new(PrepareWorkerIntegrationTestSuite))
}

func (s *PrepareWorkerIntegrationTestSuite) SetupTest() {
	s.db, _ = databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	s.pool = database.GetPool()
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
	s.r = postgres.NewRepository(s.db)

	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logrus.Fields{"cms_id": "A9999", "transaction_id": uuid.NewRandom().String()})}
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	ctx = middleware.SetRequestParamsCtx(ctx, middleware.RequestParameters{})
	s.ctx = context.WithValue(ctx, cm.CtxTransactionKey, uuid.New())

}

func (s *PrepareWorkerIntegrationTestSuite) TestPrepareExportJobsDatabase_Integration() {
	tests := []struct {
		name            string
		expectedErr     bool
		exportJobsLen   int
		parentJobStatus string
		qErr            bool
		bfdErr          bool
	}{
		{"Happy path", false, 1, string(models.JobStatusPending), false, false},
		{"getBundleLastUpdated failed", true, 0, string(models.JobStatusFailed), false, true},
		{"getQueueJobs failed", true, 0, string(models.JobStatusFailed), true, false},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			svc := service.NewMockService(t)
			c := new(client.MockBlueButtonClient)

			worker := &PrepareJobWorker{svc: svc, v1Client: c, v2Client: c, r: s.r}
			aco, err := s.r.GetACOByCMSID(context.Background(), "A0003")
			if err != nil {
				s.T().Log("failed to get job")
				s.T().FailNow()
			}
			j := models.Job{Status: models.JobStatusPending, ACOID: aco.UUID, RequestURL: "/foo/bar"}
			id, _ := s.r.CreateJob(context.Background(), j)
			j.ID = id
			jobArgs := worker_types.PrepareJobArgs{
				Job:                    j,
				CMSID:                  "A0003",
				BFDPath:                "/v1/fhir",
				RequestType:            constants.DataRequestType(1),
				ComplexDataRequestType: constants.GetNewAndExistingBenes,
				CCLFFileNewID:          uint(1),
				CCLFFileOldID:          uint(2),
				ResourceTypes:          []string{"Coverage"},
			}

			if tt.bfdErr {
				// code returns before GetQueJobs
			} else if tt.qErr {
				svc.On("GetQueJobs", testUtils.CtxMatcher, mock.Anything).Return([]*worker_types.JobEnqueueArgs{}, errors.New("an error occurred"))
			} else {
				svc.On("GetQueJobs", testUtils.CtxMatcher, mock.Anything).Return([]*worker_types.JobEnqueueArgs{{ID: 52}}, nil)
			}

			if tt.bfdErr {
				c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, errors.New("an error occurred"))
			} else {
				c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)
			}

			exports, _, err := worker.prepareExportJobs(s.ctx, jobArgs)
			if tt.expectedErr {
				assert.NotNil(s.T(), err)
			} else {
				assert.Nil(s.T(), err)
			}

			assert.Len(s.T(), exports, tt.exportJobsLen)
		})
	}
}

func (s *PrepareWorkerIntegrationTestSuite) TestPrepareExportJobs_Integration() {
	cfg, err := service.LoadConfig()
	if err != nil {
		log.API.Fatalf("Failed to load service config. Err: %v", err)
	}
	svc := service.NewService(s.r, cfg, "/v1/fhir")

	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)

	aco, err := s.r.GetACOByCMSID(context.Background(), "A0002")
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	j := models.Job{Status: models.JobStatusPending, ACOID: aco.UUID, RequestURL: "/foo/bar"}
	id, _ := s.r.CreateJob(context.Background(), j)
	j.ID = id
	jobArgs := worker_types.PrepareJobArgs{
		Job:                    j,
		ACOID:                  aco.UUID,
		CMSID:                  "A0002",
		BFDPath:                "/v1/fhir",
		RequestType:            constants.DataRequestType(1),
		ComplexDataRequestType: constants.GetNewAndExistingBenes,
		CCLFFileNewID:          uint(1),
		CCLFFileOldID:          uint(2),
		ResourceTypes:          []string{"Coverage"},
	}

	worker := &PrepareJobWorker{svc: svc, v1Client: c, v2Client: c, r: s.r}
	exports, _, err := worker.prepareExportJobs(s.ctx, jobArgs)

	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), exports)
	result, err := s.r.GetJobByID(s.ctx, id)
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	assert.Equal(s.T(), result.Status, models.JobStatusPending)
	assert.Equal(s.T(), result.JobCount, len(exports))

	exportData := *exports[0]
	assert.NotEmpty(s.T(), exportData.ACOID)
	assert.NotEmpty(s.T(), exportData.ID)
	assert.NotEmpty(s.T(), exportData.BBBasePath)
	assert.NotEmpty(s.T(), exportData.BeneficiaryIDs)
	assert.NotEmpty(s.T(), exportData.CMSID)
	assert.NotNil(s.T(), exportData.ClaimsWindow)
	assert.NotNil(s.T(), result)

}

func (s *PrepareWorkerIntegrationTestSuite) TestPrepareWorkerWork() {
	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)

	r := &models.MockRepository{}
	r.On("UpdateJob", mock.Anything, mock.Anything).Return(nil)
	models.SetMockRepository(s.T(), r)

	svc := &service.MockService{}
	cmsID := testUtils.RandomHexID()[0:4]
	clientID := uuid.New()
	aco := &models.ACO{Name: "ACO Test Name", CMSID: &cmsID, UUID: uuid.NewUUID(), ClientID: clientID, TerminationDetails: nil}
	svc.On("GetACOByCMSID", mock.Anything, mock.Anything).Return(aco, nil)
	svc.On("GetQueJobs", mock.Anything, mock.Anything).Return([]*worker_types.JobEnqueueArgs{{ID: 2}}, nil)
	svc.On("GetJobPriority", mock.Anything, mock.Anything, mock.Anything).Return(int16(1))

	j := &river.Job[worker_types.PrepareJobArgs]{
		Args: worker_types.PrepareJobArgs{
			Job:                    models.Job{},
			CMSID:                  "A9999",
			BFDPath:                "/v1/fhir",
			RequestType:            constants.DataRequestType(1),
			ComplexDataRequestType: constants.GetNewAndExistingBenes,
			CCLFFileNewID:          uint(1),
			CCLFFileOldID:          uint(2),
			ResourceTypes:          []string{"Claim"},
		},
	}

	driver := riverpgxv5.New(s.pool)
	_, err := driver.GetExecutor().Exec(context.Background(), `delete from river_job`)
	if err != nil {
		s.T().Log(err)
	}
	worker := &PrepareJobWorker{
		svc:      svc,
		v1Client: c,
		v2Client: &client.MockBlueButtonClient{},
		r:        r,
	}
	w := rivertest.NewWorker(s.T(), driver, &river.Config{}, worker)
	d := s.pool
	tx, err := d.Begin(s.ctx)
	if err != nil {
		s.T().Log(err)
	}

	result, err := w.Work(s.ctx, s.T(), tx, j.Args, &river.InsertOpts{})
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), river.EventKindJobCompleted, result.EventKind)
	assert.Equal(s.T(), rivertype.JobStateCompleted, result.Job.State)

}

func (s *PrepareWorkerIntegrationTestSuite) TestPrepareWorkerWork_Integration() {

	cfg, err := service.LoadConfig()
	if err != nil {
		log.API.Fatalf("Failed to load service config. Err: %v", err)
	}

	svc := service.NewService(s.r, cfg, "/v1/fhir")

	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)

	aco, err := s.r.GetACOByCMSID(context.Background(), "A0003")
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	j := models.Job{Status: models.JobStatusPending, ACOID: aco.UUID, RequestURL: "/foo/bar"}
	id, _ := s.r.CreateJob(context.Background(), j)
	j.ID = id

	jobArgs := &river.Job[worker_types.PrepareJobArgs]{
		Args: worker_types.PrepareJobArgs{
			Job:                    j,
			CMSID:                  "A0003",
			BFDPath:                "/v1/fhir",
			RequestType:            constants.DataRequestType(1),
			ComplexDataRequestType: constants.GetNewAndExistingBenes,
			CCLFFileNewID:          uint(1),
			CCLFFileOldID:          uint(2),
			ResourceTypes:          []string{"Coverage"},
		},
	}

	worker := &PrepareJobWorker{svc: svc, v1Client: c, v2Client: c, r: s.r}
	driver := riverpgxv5.New(s.pool)
	_, err = driver.GetExecutor().Exec(context.Background(), `delete from river_job`)
	if err != nil {
		s.T().Log(err)
	}
	w := rivertest.NewWorker(s.T(), driver, &river.Config{}, worker)
	d := s.pool
	tx, err := d.Begin(s.ctx)
	if err != nil {
		s.T().Log(err)
	}
	result, err := w.Work(s.ctx, s.T(), tx, jobArgs.Args, &river.InsertOpts{})

	assert.Nil(s.T(), err)
	dbresult, err := s.r.GetJobByID(s.ctx, id)
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	assert.Equal(s.T(), dbresult.Status, models.JobStatusPending)
	assert.NotEqual(s.T(), dbresult.JobCount, 0)
	assert.Equal(s.T(), result.EventKind, river.EventKindJobCompleted)
	assert.Equal(s.T(), rivertype.JobStateCompleted, result.Job.State)

}

func (s *PrepareWorkerIntegrationTestSuite) TestPrepareWorker() {
	w, err := NewPrepareJobWorker(s.db)
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), w)
}

func (s *PrepareWorkerIntegrationTestSuite) TestGetBundleLastUpdated() {
	basepath := "/v1/fhir"
	svc := &service.MockService{}
	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)
	worker := &PrepareJobWorker{svc: svc, v1Client: c, v2Client: c, r: s.r}
	_, err := worker.GetBundleLastUpdated(basepath, worker_types.JobEnqueueArgs{})
	assert.Nil(s.T(), err)
}

func (s *PrepareWorkerIntegrationTestSuite) TestQueueExportJobs() {
	prepArgs := worker_types.PrepareJobArgs{}
	ms := &service.MockService{}
	ms.On("GetJobPriority", mock.Anything, mock.Anything, mock.Anything).Return(int16(1))

	worker := &PrepareJobWorker{svc: ms, v1Client: &client.MockBlueButtonClient{}, v2Client: &client.MockBlueButtonClient{}, r: s.r}
	q := NewEnqueuer(s.db, database.GetPool())
	a := &worker_types.JobEnqueueArgs{
		ID: 33,
	}

	driver := riverpgxv5.New(s.pool)
	_, err := driver.GetExecutor().Exec(context.Background(), `delete from river_job`)
	assert.Nil(s.T(), err)

	err = worker.queueExportJobs(context.Background(), q, prepArgs, []*worker_types.JobEnqueueArgs{a}, time.Time{})
	assert.Nil(s.T(), err)
	re := rivertest.RequireInserted(s.ctx, s.T(), driver, worker_types.JobEnqueueArgs{}, nil)
	assert.Equal(s.T(), re.State, rivertype.JobState("available"))

	// Cleanup the queue data
	_, err = driver.GetExecutor().Exec(context.Background(), `delete from river_job`)
	assert.Nil(s.T(), err)
}
