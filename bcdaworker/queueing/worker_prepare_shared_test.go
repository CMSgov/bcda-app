package queueing

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/api/request_types"
	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/fhir"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/web/middleware"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	workerRepo "github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/db"
	"github.com/CMSgov/bcda-app/log"
	authMw "github.com/CMSgov/bcda-app/middleware"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/pborman/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/riverqueue/river/rivertype"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
)

var mbis = []string{"1S00EU7JJ00", "1S00EU7JJ11", "1S00EU7JJ18", "1S00EU8EX75", "1S00EU8EY44"}
var adjQuery = request_types.SharedExportQuery{
	DataTypes:     []string{"adjudicated"},
	MBIs:          mbis,
	ResourceTypes: []string{"Patient"},
	SourceQuery:   "https://test.test.cms.gov/v3/Patient&params=test",
	ParsedQuery: request_types.SharedExportParams{
		Headers: map[string]string{},
		Parameters: request_types.SharedExportParameters{
			Since: "2026-08-01T08:00:00-05:00",
			TypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "Patient",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "service-date",
						Value: "gt2026-08-01",
					},
					{
						Name:  "_tag",
						Value: "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem",
					},
				},
			},
		},
	},
	Version: "v3",
}
var partAdjQuery = request_types.SharedExportQuery{
	DataTypes:     []string{"partially-adjudicated"},
	MBIs:          mbis,
	ResourceTypes: []string{"Claim"},
	SourceQuery:   "https://test.test.cms.gov/v3/Patient&params=test",
	ParsedQuery: request_types.SharedExportParams{
		Headers: map[string]string{},
		Parameters: request_types.SharedExportParameters{
			Since: "2023-08-01T08:00:00-05:00",
			TypeFilter: fhir.TypeFilterSubquery{
				ResourceType: "Claim",
				QueryParameters: []fhir.TypeFilterSubqueryParam{
					{
						Name:  "service-date",
						Value: "gt2023-08-01",
					},
					{
						Name:  "_tag",
						Value: "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory",
					},
				},
			},
		},
	},
	Version: "v3",
}

type PrepareSharedWorkerTestSuite struct {
	suite.Suite
	r           models.Repository
	db          *sql.DB
	pool        *pgxv5Pool.Pool
	ctx         context.Context
	dbContainer db.TestDatabaseContainer
}

func TestCleanupPrepareSharedWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(PrepareSharedWorkerTestSuite))
}

func (s *PrepareSharedWorkerTestSuite) SetupSuite() {
	var err error
	s.dbContainer, err = db.NewTestDatabaseContainer()
	require.NoError(s.T(), err)
}

func (s *PrepareSharedWorkerTestSuite) TearDownSuite() {
	defer func() {
		if err := testcontainers.TerminateContainer(s.dbContainer.Container); err != nil {
			s.T().Log(fmt.Errorf("failed to terminate container: %w", err))
		}
	}()
}

func (s *PrepareSharedWorkerTestSuite) SetupTest() {
	var err error
	err = s.dbContainer.ExecuteDir("testdata/")
	require.NoError(s.T(), err)
	s.db, err = s.dbContainer.NewSqlDbConnection()
	require.NoError(s.T(), err)
	s.pool, err = s.dbContainer.NewPgxPoolConnection()
	require.NoError(s.T(), err)
	s.r = postgres.NewRepository(s.db)
	require.NoError(s.T(), err)

	var lggr logrus.Logger
	newLogEntry := &log.StructuredLoggerEntry{Logger: lggr.WithFields(logrus.Fields{"cms_id": "A9999", "transaction_id": uuid.NewRandom().String()})}
	ctx := context.WithValue(context.Background(), log.CtxLoggerKey, newLogEntry)
	ctx = middleware.SetRequestParamsCtx(ctx, request_types.RequestParameters{})
	s.ctx = context.WithValue(ctx, authMw.CtxTransactionKey, uuid.New())
}

func (s *PrepareSharedWorkerTestSuite) TearDownTest() {
	var err error
	err = s.db.Close()
	require.NoError(s.T(), err)
	err = s.dbContainer.RestoreSnapshot("Base")
	require.NoError(s.T(), err)
}

func (s *PrepareSharedWorkerTestSuite) TestNewPrepareSharedJobWorker() {
	w, err := NewPrepareSharedJobWorker(s.db, s.pool, service.Config{})
	assert.Nil(s.T(), err)
	assert.NotEmpty(s.T(), w)
}

func (s *PrepareSharedWorkerTestSuite) TestPrepareSharedWorkerWork_Integration() {
	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhir.Bundle{}, nil)

	aco, err := s.r.GetACOByCMSID(context.Background(), "A0003")
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	j := models.Job{
		Status:     models.JobStatusPending,
		ACOID:      aco.UUID,
		RequestURL: "/foo/bar",
	}
	id, _ := s.r.CreateJob(context.Background(), j)
	j.ID = id

	jobArgs := &river.Job[worker_types.PrepareSharedJobArgs]{
		Args: worker_types.PrepareSharedJobArgs{
			ACOID:         aco.UUID,
			BFDPath:       constants.BFDV3Path,
			CMSID:         "A0003",
			CreationTime:  time.Now(),
			Job:           j,
			Queries:       []request_types.SharedExportQuery{adjQuery, partAdjQuery},
			TransactionID: "txn_123",
		},
	}

	cfg := service.Config{
		ACOConfigs: []service.ACOConfig{
			{SharedExport: true, Pattern: "A0003"},
		},
	}

	worker := &PrepareSharedJobWorker{appRepo: s.r, workerRepo: *workerRepo.NewRepository(s.db), fhirClient: c, pool: s.pool, cfg: cfg}
	driver := riverpgxv5.New(s.pool)
	err = driver.GetExecutor().Exec(context.Background(), `delete from river_job`)
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

	// verify file was created
	payloadDir := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
	s.T().Cleanup(func() { _ = os.RemoveAll(payloadDir) })
}

func (s *PrepareSharedWorkerTestSuite) TestPrepareSharedExportJobs() {
	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhir.Bundle{}, nil)

	now := time.Now()
	txTime := now.Add(-1 * time.Hour)
	transactionUUID := uuid.NewRandom().String()
	ctx := context.WithValue(s.T().Context(), authMw.CtxTransactionKey, transactionUUID)

	worker := &PrepareSharedJobWorker{appRepo: s.r, workerRepo: *workerRepo.NewRepository(s.db), fhirClient: c, pool: s.pool}
	aco, err := s.r.GetACOByCMSID(context.Background(), "A0003")
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	j := models.Job{
		Status:          models.JobStatusPending,
		ACOID:           aco.UUID,
		RequestURL:      "/foo/bar",
		TransactionTime: txTime,
	}
	id, _ := s.r.CreateJob(context.Background(), j)
	j.ID = id

	jobArgs := worker_types.PrepareSharedJobArgs{
		ACOID:         aco.UUID,
		BFDPath:       constants.BFDV3Path,
		CMSID:         "A0003",
		CreationTime:  now,
		Job:           j,
		Queries:       []request_types.SharedExportQuery{adjQuery, partAdjQuery},
		TransactionID: transactionUUID,
	}

	resultSubjobs, err := worker.prepareSharedExportJobs(ctx, jobArgs)
	assert.Nil(s.T(), err)
	require.Len(s.T(), resultSubjobs, 2)

	// match first subjob data
	assert.Equal(s.T(), aco.UUID.String(), resultSubjobs[0].ACOID)
	assert.Equal(s.T(), "A0003", resultSubjobs[0].CMSID)
	assert.Len(s.T(), resultSubjobs[0].BeneficiaryIDs, len(mbis))
	for _, mbi := range mbis {
		assert.Contains(s.T(), resultSubjobs[0].BeneficiaryIDs, mbi)
	}
	assert.Equal(s.T(), "Patient", resultSubjobs[0].ResourceType)
	assert.Equal(s.T(), "Patient", resultSubjobs[0].TypeFilter.ResourceType)
	assert.Equal(s.T(), constants.BFDV3Path, resultSubjobs[0].BBBasePath)
	assert.Equal(s.T(), "adjudicated", resultSubjobs[0].DataType)
	assert.Equal(s.T(), "gt2026-08-01T08:00:00-05:00", resultSubjobs[0].Since)
	require.Len(s.T(), resultSubjobs[0].TypeFilter.QueryParameters, 2)
	assert.Equal(s.T(), "service-date", resultSubjobs[0].TypeFilter.QueryParameters[0].Name)
	assert.Equal(s.T(), "gt2026-08-01", resultSubjobs[0].TypeFilter.QueryParameters[0].Value)
	assert.Equal(s.T(), "_tag", resultSubjobs[0].TypeFilter.QueryParameters[1].Name)
	assert.Equal(s.T(), "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem", resultSubjobs[0].TypeFilter.QueryParameters[1].Value)
	assert.Equal(s.T(), transactionUUID, resultSubjobs[0].TransactionID)
	assert.WithinDuration(s.T(), now, resultSubjobs[0].TransactionTime, time.Second*5)

	// match second subjob data
	assert.Equal(s.T(), aco.UUID.String(), resultSubjobs[1].ACOID)
	assert.Equal(s.T(), "A0003", resultSubjobs[1].CMSID)
	assert.Len(s.T(), resultSubjobs[1].BeneficiaryIDs, len(mbis))
	for _, mbi := range mbis {
		assert.Contains(s.T(), resultSubjobs[1].BeneficiaryIDs, mbi)
	}
	assert.Equal(s.T(), "Claim", resultSubjobs[1].ResourceType)
	assert.Equal(s.T(), "Claim", resultSubjobs[1].TypeFilter.ResourceType)
	assert.Equal(s.T(), constants.BFDV3Path, resultSubjobs[1].BBBasePath)
	assert.Equal(s.T(), "partially-adjudicated", resultSubjobs[1].DataType)
	assert.Equal(s.T(), "gt2023-08-01T08:00:00-05:00", resultSubjobs[1].Since)
	require.Len(s.T(), resultSubjobs[1].TypeFilter.QueryParameters, 2)
	assert.Equal(s.T(), "service-date", resultSubjobs[1].TypeFilter.QueryParameters[0].Name)
	assert.Equal(s.T(), "gt2023-08-01", resultSubjobs[1].TypeFilter.QueryParameters[0].Value)
	assert.Equal(s.T(), "_tag", resultSubjobs[1].TypeFilter.QueryParameters[1].Name)
	assert.Equal(s.T(), "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory", resultSubjobs[1].TypeFilter.QueryParameters[1].Value)
	assert.Equal(s.T(), transactionUUID, resultSubjobs[1].TransactionID)
	assert.Equal(s.T(), now, resultSubjobs[1].TransactionTime)

	// make sure job updated properly
	dbresult, err := s.r.GetJobByID(s.ctx, id)
	if err != nil {
		s.T().Log("failed to get job")
		s.T().FailNow()
	}
	assert.Equal(s.T(), dbresult.Status, models.JobStatusPending)
	assert.Equal(s.T(), dbresult.JobCount, 2)
	assert.Equal(s.T(), dbresult.BenesAttributedToACO, len(mbis)*2)
}

func (s *PrepareSharedWorkerTestSuite) TestChunkIntoSubJobs() {
	acoUUID := uuid.NewRandom()
	transactionUUID := "tx-id"
	now := time.Now()
	txTime := now.Add(-1 * time.Hour)

	jobArgs := worker_types.PrepareSharedJobArgs{
		ACOID:        acoUUID,
		BFDPath:      constants.BFDV3Path,
		CMSID:        "A0003",
		CreationTime: now,
		Job: models.Job{
			ACOID:           acoUUID,
			TransactionTime: txTime,
		},
		Queries:       []request_types.SharedExportQuery{adjQuery, partAdjQuery},
		TransactionID: transactionUUID,
	}

	resultSubjobs, err := chunkIntoSubJobs(jobArgs)
	assert.Nil(s.T(), err)
	require.Len(s.T(), resultSubjobs, 2)

	// match first subjob data
	assert.Equal(s.T(), acoUUID.String(), resultSubjobs[0].ACOID)
	assert.Equal(s.T(), "A0003", resultSubjobs[0].CMSID)
	assert.Len(s.T(), resultSubjobs[0].BeneficiaryIDs, len(mbis))
	for _, mbi := range mbis {
		assert.Contains(s.T(), resultSubjobs[0].BeneficiaryIDs, mbi)
	}
	assert.Equal(s.T(), "Patient", resultSubjobs[0].ResourceType)
	assert.Equal(s.T(), "Patient", resultSubjobs[0].TypeFilter.ResourceType)
	assert.Equal(s.T(), constants.BFDV3Path, resultSubjobs[0].BBBasePath)
	assert.Equal(s.T(), "adjudicated", resultSubjobs[0].DataType)
	assert.Equal(s.T(), "gt2026-08-01T08:00:00-05:00", resultSubjobs[0].Since)
	require.Len(s.T(), resultSubjobs[0].TypeFilter.QueryParameters, 2)
	assert.Equal(s.T(), "service-date", resultSubjobs[0].TypeFilter.QueryParameters[0].Name)
	assert.Equal(s.T(), "gt2026-08-01", resultSubjobs[0].TypeFilter.QueryParameters[0].Value)
	assert.Equal(s.T(), "_tag", resultSubjobs[0].TypeFilter.QueryParameters[1].Name)
	assert.Equal(s.T(), "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|SharedSystem", resultSubjobs[0].TypeFilter.QueryParameters[1].Value)
	assert.Equal(s.T(), transactionUUID, resultSubjobs[0].TransactionID)
	assert.Equal(s.T(), txTime, resultSubjobs[0].TransactionTime)

	// match second subjob data
	assert.Equal(s.T(), acoUUID.String(), resultSubjobs[1].ACOID)
	assert.Equal(s.T(), "A0003", resultSubjobs[1].CMSID)
	assert.Len(s.T(), resultSubjobs[1].BeneficiaryIDs, len(mbis))
	for _, mbi := range mbis {
		assert.Contains(s.T(), resultSubjobs[1].BeneficiaryIDs, mbi)
	}
	assert.Equal(s.T(), "Claim", resultSubjobs[1].ResourceType)
	assert.Equal(s.T(), "Claim", resultSubjobs[1].TypeFilter.ResourceType)
	assert.Equal(s.T(), constants.BFDV3Path, resultSubjobs[1].BBBasePath)
	assert.Equal(s.T(), "partially-adjudicated", resultSubjobs[1].DataType)
	assert.Equal(s.T(), "gt2023-08-01T08:00:00-05:00", resultSubjobs[1].Since)
	require.Len(s.T(), resultSubjobs[1].TypeFilter.QueryParameters, 2)
	assert.Equal(s.T(), "service-date", resultSubjobs[1].TypeFilter.QueryParameters[0].Name)
	assert.Equal(s.T(), "gt2023-08-01", resultSubjobs[1].TypeFilter.QueryParameters[0].Value)
	assert.Equal(s.T(), "_tag", resultSubjobs[1].TypeFilter.QueryParameters[1].Name)
	assert.Equal(s.T(), "https://bluebutton.cms.gov/fhir/CodeSystem/System-Type|NationalClaimsHistory", resultSubjobs[1].TypeFilter.QueryParameters[1].Value)
	assert.Equal(s.T(), transactionUUID, resultSubjobs[1].TransactionID)
	assert.Equal(s.T(), now, resultSubjobs[1].TransactionTime)
}

func (s *PrepareSharedWorkerTestSuite) TestGetBundleLastUpdated() {
	c := new(client.MockBlueButtonClient)
	c.On("GetPatient", mock.Anything, "0").Return(&fhir.Bundle{}, nil)
	worker := &PrepareSharedJobWorker{appRepo: s.r, workerRepo: *workerRepo.NewRepository(s.db), fhirClient: c, pool: s.pool}

	jobArgs := worker_types.PrepareSharedJobArgs{
		BFDPath: constants.BFDV3Path,
		Queries: []request_types.SharedExportQuery{adjQuery, partAdjQuery},
	}

	resultTime, err := worker.getBundleLastUpdated(jobArgs)
	assert.NoError(s.T(), err)
	assert.WithinDuration(s.T(), resultTime, time.Now(), time.Second*5)
}

func (s *PrepareSharedWorkerTestSuite) TestQueueExportJobs() {
	prepArgs := worker_types.PrepareSharedJobArgs{}

	q := NewEnqueuer(s.db, s.pool, service.Config{ACOConfigs: []service.ACOConfig{{Pattern: "test"}}})
	subjob1 := &worker_types.JobEnqueueArgs{ID: 33}
	subjob2 := &worker_types.JobEnqueueArgs{ID: 34}

	driver := riverpgxv5.New(s.pool)
	err := driver.GetExecutor().Exec(s.ctx, `delete from river_job`)
	assert.Nil(s.T(), err)

	tx, err := s.pool.Begin(s.ctx)
	assert.Nil(s.T(), err)
	defer func() {
		if err != nil {
			if err1 := tx.Rollback(s.ctx); err1 != nil {
				s.T().Logf("Failed to rollback pgx transaction: %s", err1.Error())
			}
		}
	}()

	err = queueExportJobs(s.T().Context(), tx, q, prepArgs, []*worker_types.JobEnqueueArgs{subjob1, subjob2})
	assert.Nil(s.T(), err)

	err = tx.Commit(s.ctx)
	assert.Nil(s.T(), err)

	subjobResults := rivertest.RequireManyInserted(s.ctx, s.T(), driver, []rivertest.ExpectedJob{
		{Args: worker_types.JobEnqueueArgs{ID: 33}},
		{Args: worker_types.JobEnqueueArgs{ID: 34}},
	})
	require.Len(s.T(), subjobResults, 2)
	assert.Equal(s.T(), subjobResults[0].State, rivertype.JobState("available"))
	assert.Equal(s.T(), subjobResults[1].State, rivertype.JobState("available"))

	// Cleanup the queue data
	err = driver.GetExecutor().Exec(s.ctx, `delete from river_job`)
	assert.Nil(s.T(), err)
}
