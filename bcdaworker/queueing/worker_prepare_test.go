package queueing

import (
	"context"
	"errors"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	cm "github.com/CMSgov/bcda-app/middleware"
	"github.com/go-testfixtures/testfixtures/v3"
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

type PrepareWorkerUnitTestSuite struct {
	suite.Suite
	r models.Repository
}

func TestCleanupTestSuite(t *testing.T) {
	suite.Run(t, new(PrepareWorkerUnitTestSuite))
}

func (s *PrepareWorkerUnitTestSuite) SetupTest() {
	db, _ := databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	tf, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("testdata/"),
	)
	if err != nil {
		assert.FailNowf(s.T(), "Failed to setup test fixtures", err.Error())
	}
	if err := tf.Load(); err != nil {
		assert.FailNowf(s.T(), "Failed to load test fixtures", err.Error())
	}
	s.r = postgres.NewRepository(db)
}

func (s *PrepareWorkerUnitTestSuite) TestPrepareExportJobsDatabase_Integration() {
	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logrus.Fields{"cms_id": "A9999", "transaction_id": uuid.NewRandom().String()})}
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	ctx = middleware.SetRequestParamsCtx(ctx, middleware.RequestParameters{})
	ctx = context.WithValue(ctx, cm.CtxTransactionKey, uuid.New())

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
		{"getQueueJobs failed", true, 0, string(models.JobStatusFailed), false, true},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			svc := &service.MockService{}
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
			jobArgs := PrepareJobArgs{
				Job:           j,
				CMSID:         "A0003",
				BFDPath:       "/v1/fhir",
				RequestType:   service.RequestType(1),
				ResourceTypes: []string{"Coverage"},
			}

			if tt.qErr {
				svc.On("GetQueJobs", testUtils.CtxMatcher, mock.Anything).Return([]*models.JobEnqueueArgs{}, errors.New("an error occurred"))
			} else {
				svc.On("GetQueJobs", testUtils.CtxMatcher, mock.Anything).Return([]*models.JobEnqueueArgs{{ID: 1}}, nil)
			}

			if tt.bfdErr {
				c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, errors.New("an error occurred"))
			} else {
				c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)
			}

			exports, _, err := worker.prepareExportJobs(ctx, jobArgs)
			if tt.expectedErr {
				assert.NotNil(s.T(), err)
			} else {
				assert.Nil(s.T(), err)
			}

			assert.Len(s.T(), exports, tt.exportJobsLen)
		})
	}
}

func (s *PrepareWorkerUnitTestSuite) TestPrepareExportJobs_Integration() {
	db, _ := databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	tf, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory("testdata/"),
	)
	if err != nil {
		assert.FailNowf(s.T(), "Failed to setup test fixtures", err.Error())
	}
	if err := tf.Load(); err != nil {
		assert.FailNowf(s.T(), "Failed to load test fixtures", err.Error())
	}
	cfg, err := service.LoadConfig()
	if err != nil {
		log.API.Fatalf("Failed to load service config. Err: %v", err)
	}
	r := postgres.NewRepository(db)
	svc := service.NewService(r, cfg, "/v1/fhir")

	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)

	aco, err := r.GetACOByCMSID(context.Background(), "A0003")
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	j := models.Job{Status: models.JobStatusPending, ACOID: aco.UUID, RequestURL: "/foo/bar"}
	id, _ := r.CreateJob(context.Background(), j)
	j.ID = id
	jobArgs := PrepareJobArgs{
		Job:           j,
		CMSID:         "A0003",
		BFDPath:       "/v1/fhir",
		RequestType:   service.RequestType(1),
		ResourceTypes: []string{"Coverage"},
	}

	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logrus.Fields{"cms_id": "A9999", "transaction_id": uuid.NewRandom().String()})}
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	ctx = middleware.SetRequestParamsCtx(ctx, middleware.RequestParameters{})
	ctx = context.WithValue(ctx, cm.CtxTransactionKey, uuid.New())

	worker := &PrepareJobWorker{svc: svc, v1Client: c, v2Client: c, r: r}
	exports, _, err := worker.prepareExportJobs(ctx, jobArgs)

	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), exports)
	result, err := r.GetJobByID(ctx, id)
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	assert.Equal(s.T(), result.Status, models.JobStatusPending)
	assert.Equal(s.T(), result.JobCount, len(exports))

}

func (s *PrepareWorkerUnitTestSuite) TestPrepareWorkerWork() {
	conf.SetEnv(s.T(), "QUEUE_LIBRARY", "river")
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
	svc.On("GetQueJobs", mock.Anything, mock.Anything).Return([]*models.JobEnqueueArgs{{ID: 1}}, nil)
	svc.On("GetJobPriority", mock.Anything, mock.Anything, mock.Anything).Return(int16(1))

	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logrus.Fields{"cms_id": "A9999", "transaction_id": uuid.NewRandom().String()})}
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	ctx = middleware.SetRequestParamsCtx(ctx, middleware.RequestParameters{})
	ctx = context.WithValue(ctx, cm.CtxTransactionKey, uuid.New())

	j := &river.Job[PrepareJobArgs]{
		Args: PrepareJobArgs{
			Job:           models.Job{},
			CMSID:         "A9999",
			BFDPath:       "/v1/fhir",
			RequestType:   service.RequestType(1),
			ResourceTypes: []string{"Coverage"},
		},
	}

	driver := riverpgxv5.New(database.Pgxv5Pool)

	worker := &PrepareJobWorker{
		svc:      svc,
		v1Client: c,
		v2Client: &client.MockBlueButtonClient{},
		r:        r,
	}
	w := rivertest.NewWorker(s.T(), driver, &river.Config{}, worker)
	d := database.Pgxv5Pool
	tx, err := d.Begin(ctx)
	if err != nil {
		s.T().Log(err)
	}

	result, err := w.Work(ctx, s.T(), tx, j.Args, &river.InsertOpts{})
	//assert.NotNil(s.T(), mockCall)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), river.EventKindJobCompleted, result.EventKind)
	assert.Equal(s.T(), rivertype.JobStateCompleted, result.Job.State)
	// assert

}

func (s *PrepareWorkerUnitTestSuite) TestWorkerWork_Integration() {
	/*
		- setup a new handler
		- call bulk requests
		- verify job gets put on queue
		- run through work
		- verify job gets put on other queue
		- verify job gets processed
	*/

	//h := NewHandler()

	//tx := pgxv5Pool.Tx{}

	// tests the execution of an existing job:
	//job := client.InsertTx(ctx, tx, args, nil)
	// ...
	//result, err := testWorker.WorkJob(ctx, t, tx, job.JobRow)
}

func (s *PrepareWorkerUnitTestSuite) TestPrepareWorker() {

}
func (s *PrepareWorkerUnitTestSuite) TestGetPatient() {

}

func (s *PrepareWorkerUnitTestSuite) TestQueueExportJobs() {
}

// func getCCLFBeneficiary() *models.CCLFBeneficiary {
// 	return &models.CCLFBeneficiary{
// 		ID: func() uint {
// 			id, err := safecast.ToUint(testUtils.CryptoRandInt63())
// 			if err != nil {
// 				panic(err)
// 			}
// 			return id
// 		}(),
// 		FileID: func() uint {
// 			id, err := safecast.ToUint(testUtils.CryptoRandInt31())
// 			if err != nil {
// 				panic(err)
// 			}
// 			return id
// 		}(),
// 		MBI:          fmt.Sprintf("MBI%d", testUtils.CryptoRandInt31()),
// 		BlueButtonID: fmt.Sprintf("BlueButton%d", testUtils.CryptoRandInt31()),
// 	}
// }
