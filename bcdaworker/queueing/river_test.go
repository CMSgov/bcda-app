package queueing

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/ccoveille/go-safecast"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

// These are set in que_test.go
// var logger = logrus.New()
// var logHook = test.NewLocal(logger)

// TestProcessJob acts as an end-to-end verification of the process.
// It uses the postgres/que-go backed implementations
func TestProcessJob_Integration(t *testing.T) {
	// Set up the logger since we're using the real client
	client.SetLogger(logger)

	// Reset our environment once we've finished with the test
	defer func(origEnqueuer string) {
		conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
	}(conf.GetEnv("QUEUE_LIBRARY"))

	conf.SetEnv(t, "QUEUE_LIBRARY", "river")

	defer func(payload, staging string) {
		conf.SetEnv(t, "FHIR_PAYLOAD_DIR", payload)
		conf.SetEnv(t, "FHIR_STAGING_DIR", staging)
	}(conf.GetEnv("FHIR_PAYLOAD_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))

	defer func(cert, key, ca string) {
		conf.SetEnv(t, "BB_CLIENT_CERT_FILE", cert)
		conf.SetEnv(t, "BB_CLIENT_KEY_FILE", key)
		conf.SetEnv(t, "BB_CLIENT_CA_FILE", ca)
	}(conf.GetEnv("BB_CLIENT_CERT_FILE"), conf.GetEnv("BB_CLIENT_KEY_FILE"), conf.GetEnv("BB_CLIENT_CA_FILE"))

	conf.SetEnv(t, "BB_CLIENT_CERT_FILE", "../../shared_files/decrypted/bfd-dev-test-cert.pem")
	conf.SetEnv(t, "BB_CLIENT_KEY_FILE", "../../shared_files/decrypted/bfd-dev-test-key.pem")
	conf.SetEnv(t, "BB_CLIENT_CA_FILE", "../../shared_files/localhost.crt")

	// Ensure we do not clutter our working directory with any data
	tempDir1, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	tempDir2, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err.Error())
	}
	conf.SetEnv(t, "FHIR_PAYLOAD_DIR", tempDir1)
	conf.SetEnv(t, "FHIR_STAGING_DIR", tempDir2)

	db := database.Connection

	cmsID := testUtils.RandomHexID()[0:4]
	aco := models.ACO{UUID: uuid.NewRandom(), CMSID: &cmsID}
	postgrestest.CreateACO(t, db, aco)
	job := models.Job{ACOID: aco.UUID, Status: models.JobStatusPending}
	postgrestest.CreateJobs(t, db, &job)

	defer postgrestest.DeleteACO(t, db, aco.UUID)

	q := StartRiver(1)
	defer q.StopRiver()

	id, _ := safecast.ToInt(job.ID)
	jobArgs := models.JobEnqueueArgs{ID: id, ACOID: cmsID, BBBasePath: uuid.New()}

	enqueuer := NewEnqueuer()
	assert.NoError(t, enqueuer.AddJob(context.Background(), jobArgs, 1))

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatal("Job not completed in alloted time.")
			return
		default:
			currentJob := postgrestest.GetJobByID(t, db, job.ID)
			// don't wait for a job if it has a terminal status
			if isTerminalStatus(currentJob.Status) {
				return
			}
			logger.Infof("Waiting on job to be completed. Current status %s.", job.Status)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Runs 100k (very simple) jobs to try to test performance, DB connections, etc
// Commented out as something we probably only want to run very occassionally
// func TestProcessJobPerformance_Integration(t *testing.T) {
// 	defer func(origEnqueuer string) {
// 		conf.SetEnv(t, "QUEUE_LIBRARY", origEnqueuer)
// 	}(conf.GetEnv("QUEUE_LIBRARY"))

// 	conf.SetEnv(t, "QUEUE_LIBRARY", "river")

// 	q := StartRiver(1)
// 	defer q.StopRiver()

// 	db := database.Connection

// 	cmsID := testUtils.RandomHexID()[0:4]
// 	aco := models.ACO{UUID: uuid.NewRandom(), CMSID: &cmsID}
// 	postgrestest.CreateACO(t, db, aco)
// 	job := models.Job{ACOID: aco.UUID, Status: models.JobStatusPending}
// 	postgrestest.CreateJobs(t, db, &job)
// 	bbPath := uuid.New()

// 	defer postgrestest.DeleteACO(t, db, aco.UUID)

// 	jobID, _ := safecast.ToInt(job.ID)

// 	enqueuer := NewEnqueuer()

// 	for i := 0; i <= 100_000; i++ {
// 		jobArgs := models.JobEnqueueArgs{ID: jobID, ACOID: cmsID, BBBasePath: bbPath}
// 		err := enqueuer.AddJob(context.Background(), jobArgs, 1)
// 		assert.NoError(t, err)
// 	}
// }
