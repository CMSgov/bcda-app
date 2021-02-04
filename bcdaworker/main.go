package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	fhirmodels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"

	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/repository"
	"github.com/CMSgov/bcda-app/bcdaworker/repository/postgres"
	"github.com/CMSgov/bcda-app/conf"
)

var (
	qc *que.Client
)

func init() {
	createWorkerDirs()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
	filePath := conf.GetEnv("BCDA_WORKER_ERROR_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to open worker error log file; using default stderr")
	}
}

func createWorkerDirs() {
	staging := conf.GetEnv("FHIR_STAGING_DIR")
	err := os.MkdirAll(staging, 0744)
	if err != nil {
		log.Fatal(err)
	}
}

func processJob(j *que.Job) error {
	m := monitoring.GetMonitor()
	txn := m.Start("processJob", nil, nil)
	ctx := newrelic.NewContext(context.Background(), txn)
	defer m.End(txn)

	log.Info("Worker started processing job ", j.ID)

	// Update the Cloudwatch Metric for job queue count
	updateJobQueueCountCloudwatchMetric()

	db := database.GetDbConnection()
	defer db.Close()
	r := postgres.NewRepository(db)

	jobArgs := models.JobEnqueueArgs{}
	err := json.Unmarshal(j.Args, &jobArgs)
	if err != nil {
		return err
	}

	// Verify Jobs have a BB base path
	if len(jobArgs.BBBasePath) == 0 {
		err = errors.New("empty BBBasePath: Must be set")
		log.Error(err)
		return err
	}

	exportJob, err := r.GetJobByID(ctx, uint(jobArgs.ID))
	if goerrors.Is(err, repository.ErrJobNotFound) {
		// Based on the current backoff delay (j.ErrorCount^4 + 3 seconds), this should've given
		// us plenty of headroom to ensure that the parent job will never be found.
		maxNotFoundRetries := int32(utils.GetEnvInt("BCDA_WORKER_MAX_JOB_NOT_FOUND_RETRIES", 3))
		if j.ErrorCount >= maxNotFoundRetries {
			log.Errorf("No job found for ID: %d acoID: %s. Retries exhausted. Removing job from queue.", jobArgs.ID,
				jobArgs.ACOID)
			// By returning a nil error response, we're singaling to que-go to remove this job from the jobqueue.
			return nil
		}

		log.Warnf("No job found for ID %d acoID: %s. Will retry.", jobArgs.ID, jobArgs.ACOID)
		return errors.Wrap(repository.ErrJobNotFound, "could not retrieve job from database")
	}

	if err != nil {
		return errors.Wrap(err, "could not retrieve job from database")
	}

	aco, err := r.GetACOByUUID(ctx, exportJob.ACOID)
	if err != nil {
		return errors.Wrap(err, "could not retrieve ACO from database")
	}

	err = r.UpdateJobStatusCheckStatus(ctx, exportJob.ID, models.JobStatusPending, models.JobStatusInProgress)
	if goerrors.Is(err, repository.ErrJobNotUpdated) {
		log.Warnf("Failed to update job. Assume job already updated. Continuing. %s", err.Error())
	} else if err != nil {
		return errors.Wrap(err, "could not update job status in database")
	}

	bb, err := client.NewBlueButtonClient(client.NewConfig(jobArgs.BBBasePath))
	if err != nil {
		err = errors.Wrap(err, "could not create Blue Button client")
		log.Error(err)
		return err
	}

	jobID := strconv.Itoa(jobArgs.ID)
	stagingPath := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_STAGING_DIR"), jobID)
	payloadPath := fmt.Sprintf("%s/%s", conf.GetEnv("FHIR_PAYLOAD_DIR"), jobID)

	if err = createDir(stagingPath); err != nil {
		log.Error(err)
		return err
	}

	// Create directory for job results.
	// This will be used in the clean up later to move over processed files.
	if err = createDir(payloadPath); err != nil {
		log.Error(err)
		return err
	}

	fileUUID, fileSize, err := writeBBDataToFile(ctx, r, bb, *aco.CMSID, jobArgs)
	fileName := fileUUID + ".ndjson"

	// This is only run AFTER completion of all the collection
	if err != nil {
		err = r.UpdateJobStatus(ctx, exportJob.ID, models.JobStatusFailed)
		if err != nil {
			return err
		}
	} else {
		if fileSize == 0 {
			log.Warn("Empty file found in request: ", fileName)
			fileName = models.BlankFileName
		}

		err = addJobFileName(ctx, r, fileName, jobArgs.ResourceType, *exportJob)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	_, err = checkJobCompleteAndCleanup(ctx, r, exportJob.ID)
	if err != nil {
		log.Error(err)
		return err
	}

	updateJobStats(ctx, r, exportJob.ID)

	log.Info("Worker finished processing job ", j.ID)

	return nil
}

func createDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err = os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func writeBBDataToFile(ctx context.Context, r repository.Repository, bb client.APIClient,
	cmsID string, jobArgs models.JobEnqueueArgs) (fileUUID string, size int64, err error) {
	segment := getSegment(ctx, "writeBBDataToFile")
	defer func() {
		segment.End()
	}()

	var bundleFunc func(bbID string) (*fhirmodels.Bundle, error)
	switch jobArgs.ResourceType {
	case "Coverage":
		bundleFunc = func(bbID string) (*fhirmodels.Bundle, error) {
			return bb.GetCoverage(bbID, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime)
		}
	case "ExplanationOfBenefit":
		bundleFunc = func(bbID string) (*fhirmodels.Bundle, error) {
			return bb.GetExplanationOfBenefit(bbID, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime, jobArgs.ServiceDate)
		}
	case "Patient":
		bundleFunc = func(bbID string) (*fhirmodels.Bundle, error) {
			return bb.GetPatient(bbID, strconv.Itoa(jobArgs.ID), cmsID, jobArgs.Since, jobArgs.TransactionTime)
		}
	default:
		return "", 0, fmt.Errorf("unsupported resource type %s", jobArgs.ResourceType)
	}

	dataDir := conf.GetEnv("FHIR_STAGING_DIR")
	fileUUID = uuid.New()
	f, err := os.Create(fmt.Sprintf("%s/%d/%s.ndjson", dataDir, jobArgs.ID, fileUUID))
	if err != nil {
		log.Error(err)
		return "", 0, err
	}

	defer utils.CloseFileAndLogError(f)

	w := bufio.NewWriter(f)
	errorCount := 0
	totalBeneIDs := float64(len(jobArgs.BeneficiaryIDs))
	failThreshold := getFailureThreshold()
	failed := false

	for _, beneID := range jobArgs.BeneficiaryIDs {
		errMsg, err := func() (string, error) {
			id, err := strconv.ParseUint(beneID, 10, 64)
			if err != nil {
				return fmt.Sprintf("Error failed to convert %s to uint", beneID), err
			}

			bene, err := getBeneficiary(ctx, r, uint(id), bb)
			if err != nil {
				return fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary MBI %s", bene.MBI), err
			}
			b, err := bundleFunc(bene.BlueButtonID)
			if err != nil {
				return fmt.Sprintf("Error retrieving %s for beneficiary MBI %s in ACO %s", jobArgs.ResourceType, bene.MBI, jobArgs.ACOID), err
			}
			fhirBundleToResourceNDJSON(ctx, w, b, jobArgs.ResourceType, beneID, cmsID, fileUUID, jobArgs.ID)
			return "", nil
		}()

		if err != nil {
			log.Error(err)
			errorCount++
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_EXCEPTION, responseutils.BbErr, errMsg, jobArgs.ID)
		}

		failPct := (float64(errorCount) / totalBeneIDs) * 100
		if failPct >= failThreshold {
			failed = true
			break
		}
	}

	if err = w.Flush(); err != nil {
		return "", 0, err
	}

	if failed {
		return "", 0, errors.New("number of failed requests has exceeded threshold")
	}

	fstat, err := f.Stat()
	if err != nil {
		return "", 0, err
	}

	return fileUUID, fstat.Size(), nil
}

// getBeneficiary returns the beneficiary. The bb ID value is retrieved and set in the model.
func getBeneficiary(ctx context.Context, r repository.Repository, beneID uint, bb client.APIClient) (models.CCLFBeneficiary, error) {

	bene, err := r.GetCCLFBeneficiaryByID(ctx, beneID)
	if err != nil {
		return models.CCLFBeneficiary{}, err
	}

	cclfBeneficiary := *bene

	bbID, err := cclfBeneficiary.GetBlueButtonID(bb)
	if err != nil {
		return cclfBeneficiary, err
	}

	cclfBeneficiary.BlueButtonID = bbID
	return cclfBeneficiary, nil
}

func getFailureThreshold() float64 {
	exportFailPctStr := conf.GetEnv("EXPORT_FAIL_PCT")
	exportFailPct, err := strconv.Atoi(exportFailPctStr)
	if err != nil {
		exportFailPct = 50
	} else if exportFailPct < 0 {
		exportFailPct = 0
	} else if exportFailPct > 100 {
		exportFailPct = 100
	}
	return float64(exportFailPct)
}

func appendErrorToFile(ctx context.Context, fileUUID string, 
	code fhircodes.IssueTypeCode_Value,
	detailsCode, detailsDisplay string, jobID int) {
	segment := getSegment(ctx, "appendErrorToFile")
	defer func() {
		segment.End()
	}()

	oo := responseutils.CreateOpOutcome(fhircodes.IssueSeverityCode_ERROR, code, detailsCode, detailsDisplay)

	dataDir := conf.GetEnv("FHIR_STAGING_DIR")
	fileName := fmt.Sprintf("%s/%d/%s-error.ndjson", dataDir, jobID, fileUUID)
	/* #nosec -- opening file defined by variable */
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		log.Error(err)
	}

	defer utils.CloseFileAndLogError(f)

	ooBytes, err := json.Marshal(oo)
	if err != nil {
		log.Error(err)
	}

	if _, err = f.WriteString(string(ooBytes) + "\n"); err != nil {
		log.Error(err)
	}
}

func fhirBundleToResourceNDJSON(ctx context.Context, w *bufio.Writer, b *fhirmodels.Bundle, jsonType, beneficiaryID, acoID, fileUUID string, jobID int) {
	segment := getSegment(ctx, "fhirBundleToResourceNDJSON")
	defer func() {
		segment.End()
	}()

	for _, entry := range b.Entries {
		if entry["resource"] == nil {
			continue
		}

		entryJSON, err := json.Marshal(entry["resource"])
		// This is unlikely to happen because we just unmarshalled this data a few lines above.
		if err != nil {
			log.Error(err)
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_EXCEPTION, responseutils.InternalErr, fmt.Sprintf("Error marshaling %s to JSON for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
			continue
		}
		_, err = w.WriteString(string(entryJSON) + "\n")
		if err != nil {
			log.Error(err)
			appendErrorToFile(ctx, fileUUID, fhircodes.IssueTypeCode_EXCEPTION, responseutils.InternalErr, fmt.Sprintf("Error writing %s to file for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
		}
	}
}

func checkJobCompleteAndCleanup(ctx context.Context, r repository.Repository, jobID uint) (jobCompleted bool, err error) {
	j, err := r.GetJobByID(ctx, jobID)
	if err != nil {
		return false, err
	}

	if j.Status == models.JobStatusCompleted {
		return true, nil
	}

	completedCount, err := r.GetJobKeyCount(ctx, jobID)
	if err != nil {
		return false, err
	}

	if completedCount >= j.JobCount {
		staging := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_STAGING_DIR"), j.ID)
		payload := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)

		files, err := ioutil.ReadDir(staging)
		if err != nil {
			return false, err
		}

		for _, f := range files {
			oldPath := fmt.Sprintf("%s/%s", staging, f.Name())
			newPath := fmt.Sprintf("%s/%s", payload, f.Name())
			err := os.Rename(oldPath, newPath)
			if err != nil {
				return false, err
			}
		}

		if err = os.Remove(staging); err != nil {
			return false, err
		}

		if err := r.UpdateJobStatus(ctx, j.ID, models.JobStatusCompleted); err != nil {
			return false, err
		}

		// Able to mark job as completed
		return true, nil
	}

	// We still have parts of the job that are not complete
	return false, nil
}

func waitForSig() {
	signalChan := make(chan os.Signal, 1)
	defer close(signalChan)

	signal.Notify(signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exitChan := make(chan int)
	defer close(exitChan)

	go func() {
		for {
			s := <-signalChan
			switch s {
			case syscall.SIGINT:
				fmt.Println("interrupt")
				exitChan <- 0
			case syscall.SIGTERM:
				fmt.Println("force stop")
				exitChan <- 0
			case syscall.SIGQUIT:
				fmt.Println("stop and core dump")
				exitChan <- 0
			}
		}
	}()

	code := <-exitChan
	os.Exit(code)
}

func setupQueue() *pgx.ConnPool {
	queueDatabaseURL := conf.GetEnv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		log.Fatal(err)
	}

	qc = que.NewClient(pgxpool)
	wm := que.WorkMap{
		"ProcessJob": processJob,
	}

	workerPoolSize := utils.GetEnvInt("WORKER_POOL_SIZE", 2)
	workers := que.NewWorkerPool(qc, wm, workerPoolSize)
	go workers.Start()

	return pgxpool
}

func getQueueJobCount() float64 {
	databaseURL := conf.GetEnv("QUEUE_DATABASE_URL")
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Error(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Error(pingErr)
	}
	defer db.Close()

	row := db.QueryRow(`select count(*) from que_jobs;`)

	var count int
	if err := row.Scan(&count); err != nil {
		log.Error(err)
	}

	return float64(count)
}

func updateJobStats(ctx context.Context, r repository.Repository, jobID uint) {
	updateJobQueueCountCloudwatchMetric()

	// Not critical since we use the job_keys count as the authoritative list of completed jobs.
	// CompletedJobCount is purely information and can be off.
	if err := r.IncrementCompletedJobCount(ctx, jobID); err != nil {
		log.Warnf("Failed to update completed job count for job %d. Will continue. %s", jobID, err.Error())
	}
}

func addJobFileName(ctx context.Context, r repository.Repository, fileName, resourceType string, exportJob models.Job) error {
	if err := r.CreateJobKey(ctx, models.JobKey{JobID: exportJob.ID, FileName: fileName, ResourceType: resourceType}); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func updateJobQueueCountCloudwatchMetric() {

	// Update the Cloudwatch Metric for job queue count
	env := conf.GetEnv("DEPLOYMENT_TARGET")
	if env != "" {
		sampler, err := metrics.NewSampler("BCDA", "Count")
		if err != nil {
			fmt.Println("Warning: failed to create new metric sampler...")
		} else {
			err := sampler.PutSample("JobQueueCount", getQueueJobCount(), []metrics.Dimension{
				{Name: "Environment", Value: env},
			})
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func getSegment(ctx context.Context, name string) newrelic.Segment {
	segment := newrelic.Segment{Name: name}
	if txn := newrelic.FromContext(ctx); txn != nil {
		segment.StartTime = txn.StartSegmentNow()
	}
	return segment
}

func main() {
	fmt.Println("Starting bcdaworker...")

	workerPool := setupQueue()
	defer workerPool.Close()

	if hInt, err := strconv.Atoi(conf.GetEnv("WORKER_HEALTH_INT_SEC")); err == nil {
		healthLogger := NewHealthLogger()
		ticker := time.NewTicker(time.Duration(hInt) * time.Second)
		quit := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					healthLogger.Log()
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}

	waitForSig()
}
