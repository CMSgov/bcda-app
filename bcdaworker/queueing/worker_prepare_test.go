package queueing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	cm "github.com/CMSgov/bcda-app/middleware"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	fhirModels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/mock"
)

type PrepareWorkerUnitTestSuite struct {
	suite.Suite
}

func TestCleanupTestSuite(t *testing.T) {
	suite.Run(t, new(PrepareWorkerUnitTestSuite))
}

// func (s *PrepareWorkerUnitTestSuite) TearDownSuite() {

// }

// func (s *PrepareWorkerUnitTestSuite) SetupSuite() {

// }

// func (s *PrepareWorkerUnitTestSuite) SetupDownTest() {

// }

// func (s *PrepareWorkerUnitTestSuite) TearDownTest() {

// }

// func (s *PrepareWorkerUnitTestSuite) TestWork() {

// }

func (s *PrepareWorkerUnitTestSuite) TestPrepare() {
	cfg, err := service.LoadConfig()
	if err != nil {
		log.API.Fatalf("Failed to load service config. Err: %v", err)
	}
	r := &models.MockRepository{}
	svc := service.NewService(r, cfg, "/v1/fhir")

	cmsID := testUtils.RandomHexID()[0:4]
	clientID := uuid.New()
	aco := &models.ACO{Name: "ACO Test Name", CMSID: &cmsID, UUID: uuid.NewUUID(), ClientID: clientID, TerminationDetails: nil}
	r.On("GetACOByCMSID", mock.MatchedBy(func(req context.Context) bool { return true }), "A9999").Return(aco, nil)
	r.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, time.Time{}, models.FileTypeDefault).Return(&models.CCLFFile{
		ID:              1000,
		PerformanceYear: 25,
		CreatedAt:       time.Now(),
	}, nil)
	r.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, mock.Anything).Return([]string{"old"}, nil)
	r.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return([]*models.CCLFBeneficiary{getCCLFBeneficiary()}, nil)

	models.SetMockRepository(s.T(), r)

	c := new(client.MockBlueButtonClient)
	jobArgs := PrepareJobArgs{
		Job:           models.Job{},
		CMSID:         "A9999",
		BFDPath:       "/v1/fhir",
		RequestType:   service.RequestType(1),
		ResourceTypes: []string{"Coverage"},
	}

	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logrus.Fields{"cms_id": "A9999", "transaction_id": uuid.NewRandom().String()})}
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	ctx = middleware.SetRequestParamsCtx(ctx, middleware.RequestParameters{})
	ctx = context.WithValue(ctx, cm.CtxTransactionKey, uuid.New())
	mockCall := c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)

	worker := &PrepareJobWorker{svc: svc, v1Client: c, v2Client: c}

	assert.NotNil(s.T(), mockCall)
	exports, _, _ := worker.prepareExportJobs(ctx, jobArgs)
	assert.NotEmpty(s.T(), exports)

}

func (s *PrepareWorkerUnitTestSuite) TestPrepareWorker() {
	conf.SetEnv(s.T(), "QUEUE_LIBRARY", "river")
	c := new(client.MockBlueButtonClient)
	mockCall := c.On("GetPatient", mock.Anything, "0").Return(&fhirModels.Bundle{}, nil)

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
		v1Client: &client.MockBlueButtonClient{},
		v2Client: &client.MockBlueButtonClient{},
	}
	w := rivertest.NewWorker(s.T(), driver, &river.Config{}, worker)
	d := database.Pgxv5Pool
	tx, err := d.Begin(ctx)
	if err != nil {
		s.T().Log(err)
	}

	result, err := w.Work(ctx, s.T(), tx, j.Args, &river.InsertOpts{})
	assert.NotNil(s.T(), mockCall)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), river.EventKindJobCompleted, result.EventKind)
	assert.Equal(s.T(), rivertype.JobStateCompleted, result.Job.State)

}

func (s *PrepareWorkerUnitTestSuite) TestIntegration() {
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

func getCCLFBeneficiary() *models.CCLFBeneficiary {
	return &models.CCLFBeneficiary{
		ID: func() uint {
			id, err := safecast.ToUint(testUtils.CryptoRandInt63())
			if err != nil {
				panic(err)
			}
			return id
		}(),
		FileID: func() uint {
			id, err := safecast.ToUint(testUtils.CryptoRandInt31())
			if err != nil {
				panic(err)
			}
			return id
		}(),
		MBI:          fmt.Sprintf("MBI%d", testUtils.CryptoRandInt31()),
		BlueButtonID: fmt.Sprintf("BlueButton%d", testUtils.CryptoRandInt31()),
	}
}
