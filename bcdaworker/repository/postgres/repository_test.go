package postgres_test

import (
	"context"
	"database/sql"
	"math/rand"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type RepositoryTestSuite struct {
	suite.Suite

	db         *sql.DB
	repository *postgres.Repository
}

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

func (r *RepositoryTestSuite) SetupSuite() {
	r.db = database.Connection
	r.repository = postgres.NewRepository(r.db)
}

// TestACOMethods validates the CRUD operations associated with the acos table
func (r *RepositoryTestSuite) TestACOMethods() {
	assert := r.Assert()
	ctx := context.Background()

	cmsID := testUtils.RandomHexID()[0:4]
	aco := models.ACO{UUID: uuid.NewRandom(), Name: uuid.New(), CMSID: &cmsID}
	postgrestest.CreateACO(r.T(), r.db, aco)
	defer postgrestest.DeleteACO(r.T(), r.db, aco.UUID)

	aco1, err := r.repository.GetACOByUUID(ctx, aco.UUID)
	assert.NoError(err)
	assert.Equal(cmsID, *aco1.CMSID)
	assert.Equal(aco.Name, aco1.Name)

	other := uuid.NewRandom()
	_, err = r.repository.GetACOByUUID(ctx, other)
	assert.EqualError(err, "no ACO record found for uuid "+other.String())
}

// TestCCLFBeneficiariesMethods validates the CRUD operations associated with the cclf_beneficiaries table
func (r *RepositoryTestSuite) TestCCLFBeneficiariesMethods() {
	assert := r.Assert()
	ctx := context.Background()

	// Since we have a foreign key tie, we need the cclf file to exist before creating associated benes
	cclfFile := &models.CCLFFile{CCLFNum: 8, ACOCMSID: testUtils.RandomHexID()[0:4], Timestamp: time.Now(), PerformanceYear: 19, Name: uuid.New()}
	postgrestest.CreateCCLFFile(r.T(), r.db, cclfFile)
	defer postgrestest.DeleteCCLFFilesByCMSID(r.T(), r.db, cclfFile.ACOCMSID)

	bene := models.CCLFBeneficiary{FileID: cclfFile.ID, MBI: testUtils.RandomMBI(r.T()), BlueButtonID: testUtils.RandomHexID()}
	postgrestest.CreateCCLFBeneficiary(r.T(), r.db, &bene)

	bene1, err := r.repository.GetCCLFBeneficiaryByID(ctx, bene.ID)
	assert.NoError(err)
	assert.Equal(bene, *bene1)

	_, err = r.repository.GetCCLFBeneficiaryByID(ctx, uint(rand.Int31()))
	assert.EqualError(err, "sql: no rows in result set")
}

// TestJobsMethods validates the CRUD operations associated with the jobs table
func (r *RepositoryTestSuite) TestJobsMethods() {
	// Account for time precision in postgres
	now := time.Now().Round(time.Millisecond).UTC()

	assert := r.Assert()
	ctx := context.Background()

	cmsID := testUtils.RandomHexID()[0:4]
	// Since jobs have a foreign key tie to the ACO, we need to ensure the ACO exists
	// before creating the job
	aco := models.ACO{UUID: uuid.NewRandom(), Name: uuid.New(), CMSID: &cmsID}
	postgrestest.CreateACO(r.T(), r.db, aco)
	defer postgrestest.DeleteACO(r.T(), r.db, aco.UUID)

	failed := models.Job{ACOID: aco.UUID, Status: models.JobStatusFailed, CompletedJobCount: 1, TransactionTime: now}
	completed := models.Job{ACOID: aco.UUID, Status: models.JobStatusCompleted, CompletedJobCount: 2, TransactionTime: now}
	postgrestest.CreateJobs(r.T(), r.db, &failed, &completed)

	failed1, err := r.repository.GetJobByID(ctx, failed.ID)
	assert.NoError(err)
	assertJobsEqual(assert, failed, *failed1)

	failed.Status = models.JobStatusArchived
	assert.NoError(r.repository.UpdateJobStatus(ctx, failed.ID, failed.Status))
	afterUpdate, err := r.repository.GetJobByID(ctx, failed.ID)
	assert.NoError(err)

	assert.True(afterUpdate.UpdatedAt.After(failed.UpdatedAt))
	// Allows us to compare all of the Job fields
	failed.UpdatedAt = afterUpdate.UpdatedAt
	assertJobsEqual(assert, failed, *afterUpdate)

	failed.Status = models.JobStatusExpired
	assert.NoError(r.repository.UpdateJobStatusCheckStatus(ctx, failed.ID, models.JobStatusArchived, failed.Status))
	afterUpdate, err = r.repository.GetJobByID(ctx, failed.ID)
	assert.NoError(err)
	assert.True(afterUpdate.UpdatedAt.After(failed.UpdatedAt))
	// Allows us to compare all of the Job fields
	failed.UpdatedAt = afterUpdate.UpdatedAt
	assertJobsEqual(assert, failed, *afterUpdate)

	assert.NoError(r.repository.IncrementCompletedJobCount(ctx, failed.ID))
	afterUpdate, err = r.repository.GetJobByID(ctx, failed.ID)
	assert.NoError(err)
	assert.True(afterUpdate.UpdatedAt.After(failed.UpdatedAt))
	assert.Equal(afterUpdate.CompletedJobCount, failed.CompletedJobCount+1)

	// After all of these updates, the completed job should remain untouched
	completed1, err := r.repository.GetJobByID(ctx, completed.ID)
	assert.NoError(err)
	assertJobsEqual(assert, completed, *completed1)

	// Negative cases
	_, err = r.repository.GetJobByID(ctx, 0)
	assert.EqualError(err, "no job found for given id")

	err = r.repository.UpdateJobStatus(ctx, 0, models.JobStatusCompleted)
	assert.EqualError(err, "job was not updated, no match found")

	// Matching jobID, but mismatching job status
	err = r.repository.UpdateJobStatusCheckStatus(ctx, completed.ID, models.JobStatusFailed, models.JobStatusArchived)
	assert.EqualError(err, "job was not updated, no match found")

	err = r.repository.UpdateJobStatusCheckStatus(ctx, 0, models.JobStatusFailed, models.JobStatusArchived)
	assert.EqualError(err, "job was not updated, no match found")

	err = r.repository.IncrementCompletedJobCount(ctx, 0)
	assert.EqualError(err, "job 0 not updated, no job found")

}

// TestJobKeysMethods validates the CRUD operations associated with the job_keys table
func (r *RepositoryTestSuite) TestJobKeyMethods() {
	assert := r.Assert()
	ctx := context.Background()

	jobID := uint(rand.Int31())
	jk := models.JobKey{JobID: jobID}
	jk1 := models.JobKey{JobID: jobID}
	jk2 := models.JobKey{JobID: jobID}

	otherJobID := models.JobKey{JobID: uint(rand.Int31())}
	defer postgrestest.DeleteJobKeysByJobIDs(r.T(), r.db, jobID, otherJobID.JobID)

	assert.NoError(r.repository.CreateJobKey(ctx, jk))
	assert.NoError(r.repository.CreateJobKey(ctx, jk1))
	assert.NoError(r.repository.CreateJobKey(ctx, jk2))
	assert.NoError(r.repository.CreateJobKey(ctx, otherJobID))

	count, err := r.repository.GetJobKeyCount(ctx, jobID)
	assert.NoError(err)
	assert.Equal(3, count)

	count, err = r.repository.GetJobKeyCount(ctx, otherJobID.JobID)
	assert.NoError(err)
	assert.Equal(1, count)

	count, err = r.repository.GetJobKeyCount(ctx, 0)
	assert.NoError(err)
	assert.Equal(0, count)
}

func assertJobsEqual(assert *assert.Assertions, expected, actual models.Job) {
	expected.TransactionTime, actual.TransactionTime = expected.TransactionTime.UTC(), actual.TransactionTime.UTC()
	assert.Equal(expected, actual)
}
