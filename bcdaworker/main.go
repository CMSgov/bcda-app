package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/jinzhu/gorm"
	newrelic "github.com/newrelic/go-agent"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	fhirmodels "github.com/CMSgov/bcda-app/bcda/models/fhir"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

var (
	qc *que.Client
)

func init() {
	createWorkerDirs()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
	filePath := os.Getenv("BCDA_WORKER_ERROR_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to open worker error log file; using default stderr")
	}
}

func createWorkerDirs() {
	staging := os.Getenv("FHIR_STAGING_DIR")
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

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	jobArgs := models.JobEnqueueArgs{}
	err := json.Unmarshal(j.Args, &jobArgs)
	if err != nil {
		return err
	}

	var exportJob models.Job
	result := db.First(&exportJob, "ID = ?", jobArgs.ID)

	if result.RecordNotFound() {
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
		return errors.Wrap(gorm.ErrRecordNotFound, "could not retrieve job from database")
	}

	if result.Error != nil {
		return errors.Wrap(result.Error, "could not retrieve job from database")
	}

	var aco models.ACO
	err = db.First(&aco, "uuid = ?", exportJob.ACOID).Error
	if err != nil {
		return errors.Wrap(err, "could not retrieve ACO from database")
	}

	err = db.Model(&exportJob).Where("status = ?", "Pending").Update("status", "In Progress").Error
	if err != nil {
		return errors.Wrap(err, "could not update job status in database")
	}

	bb, err := client.NewBlueButtonClient(client.NewConfig())
	if err != nil {
		err = errors.Wrap(err, "could not create Blue Button client")
		log.Error(err)
		return err
	}

	jobID := strconv.Itoa(jobArgs.ID)
	stagingPath := fmt.Sprintf("%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID)
	payloadPath := fmt.Sprintf("%s/%s", os.Getenv("FHIR_PAYLOAD_DIR"), jobID)

	if err = createDir(stagingPath); err != nil {
		log.Error(err)
		return err
	}

	if err = createDir(payloadPath); err != nil {
		log.Error(err)
		return err
	}

	fileUUID, err := writeBBDataToFile(ctx, bb, db, jobArgs.ACOID, *aco.CMSID, jobArgs.BeneficiaryIDs, jobID, jobArgs.ResourceType, jobArgs.Since, jobArgs.TransactionTime)
	fileName := fileUUID + ".ndjson"

	// This is only run AFTER completion of all the collection
	if err != nil {
		err = db.Model(&exportJob).Update("status", "Failed").Error
		if err != nil {
			return err
		}
	} else {
		err = addJobFileName(fileName, jobArgs.ResourceType, exportJob, db)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	_, err = exportJob.CheckCompletedAndCleanup(db)
	if err != nil {
		log.Error(err)
		return err
	}

	updateJobStats(exportJob.ID, db)

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

func writeBBDataToFile(ctx context.Context, bb client.APIClient, db *gorm.DB, acoID string, acoCMSID string, cclfBeneficiaryIDs []string, jobID, t, since string, transactionTime time.Time) (fileUUID string, error error) {
	segment := getSegment(ctx, "writeBBDataToFile")
	defer func() {
		if err := segment.End(); err != nil {
			log.Error(err)
		}
	}()

	if bb == nil {
		err := errors.New("Blue Button client is required")
		log.Error(err)
		return "", err
	}

	bbFunc := bbFuncByType(bb, t)
	if bbFunc == nil {
		err := fmt.Errorf("Invalid resource type requested: %s", t)
		log.Error(err)
		return "", err
	}

	if !utils.IsUUID(acoID) {
		err := errors.New("Invalid ACO ID")
		log.Error(err)
		return "", err
	}

	dataDir := os.Getenv("FHIR_STAGING_DIR")
	fileUUID = uuid.NewRandom().String()
	f, err := os.Create(fmt.Sprintf("%s/%s/%s.ndjson", dataDir, jobID, fileUUID))
	if err != nil {
		log.Error(err)
		return "", err
	}

	defer utils.CloseFileAndLogError(f)

	w := bufio.NewWriter(f)
	errorCount := 0
	totalBeneIDs := float64(len(cclfBeneficiaryIDs))
	failThreshold := getFailureThreshold()
	failed := false

	for _, cclfBeneficiaryID := range cclfBeneficiaryIDs {
		blueButtonID, err := beneBBID(cclfBeneficiaryID, bb, db)

		if err != nil {
			handleBBError(ctx, err, &errorCount, fileUUID, fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary %s", cclfBeneficiaryID), jobID)
		} else {
			b, err := bbFunc(blueButtonID, jobID, acoCMSID, since, transactionTime)
			if err != nil {
				handleBBError(ctx, err, &errorCount, fileUUID, fmt.Sprintf("Error retrieving %s for beneficiary %s in ACO %s", t, blueButtonID, acoID), jobID)
			} else {
				fhirBundleToResourceNDJSON(ctx, w, b, t, cclfBeneficiaryID, acoCMSID, jobID, fileUUID)
			}
		}
		failPct := (float64(errorCount) / totalBeneIDs) * 100
		if failPct >= failThreshold {
			failed = true
			break
		}
	}

	err = w.Flush()
	if err != nil {
		return "", err
	}

	if failed {
		return "", errors.New("number of failed requests has exceeded threshold")
	}

	return fileUUID, nil
}

func bbFuncByType(bb client.APIClient, t string) client.BeneDataFunc {
	return map[string]client.BeneDataFunc{
		"ExplanationOfBenefit": bb.GetExplanationOfBenefit,
		"Patient":              bb.GetPatient,
		"Coverage":             bb.GetCoverage,
	}[t]
}

// beneBBID returns the beneficiary's Blue Button ID. The ID value is retrieved from BB and saved.
func beneBBID(cclfBeneID string, bb client.APIClient, db *gorm.DB) (string, error) {

	var cclfBeneficiary models.CCLFBeneficiary
	db.First(&cclfBeneficiary, cclfBeneID)
	bbID, err := cclfBeneficiary.GetBlueButtonID(bb)
	if err != nil {
		return "", err
	}

	// Update the value in the DB only if necessary
	if cclfBeneficiary.BlueButtonID != bbID {
		db.Model(&cclfBeneficiary).Update("blue_button_id", bbID)
	}

	return bbID, nil
}

func handleBBError(ctx context.Context, err error, errorCount *int, fileUUID, msg, jobID string) {
	log.Error(err)
	(*errorCount)++
	appendErrorToFile(ctx, fileUUID, responseutils.Exception, responseutils.BbErr, msg, jobID)
}

func getFailureThreshold() float64 {
	exportFailPctStr := os.Getenv("EXPORT_FAIL_PCT")
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

func appendErrorToFile(ctx context.Context, fileUUID, code, detailsCode, detailsDisplay string, jobID string) {
	segment := getSegment(ctx, "appendErrorToFile")
	defer func() {
		if err := segment.End(); err != nil {
			log.Error(err)
		}
	}()

	oo := responseutils.CreateOpOutcome(responseutils.Error, code, detailsCode, detailsDisplay)

	dataDir := os.Getenv("FHIR_STAGING_DIR")
	fileName := fmt.Sprintf("%s/%s/%s-error.ndjson", dataDir, jobID, fileUUID)
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

func fhirBundleToResourceNDJSON(ctx context.Context, w *bufio.Writer, b *fhirmodels.Bundle, jsonType, beneficiaryID, acoID, jobID, fileUUID string) {
	segment := getSegment(ctx, "fhirBundleToResourceNDJSON")
	defer func() {
		if err := segment.End(); err != nil {
			log.Error(err)
		}
	}()

	for _, entry := range b.Entries {
		if entry["resource"] == nil {
			continue
		}

		entryJSON, err := json.Marshal(entry["resource"])
		// This is unlikely to happen because we just unmarshalled this data a few lines above.
		if err != nil {
			log.Error(err)
			appendErrorToFile(ctx, fileUUID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error marshaling %s to JSON for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
			continue
		}
		_, err = w.WriteString(string(entryJSON) + "\n")
		if err != nil {
			log.Error(err)
			appendErrorToFile(ctx, fileUUID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error writing %s to file for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
		}
	}
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
	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
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
	databaseURL := os.Getenv("QUEUE_DATABASE_URL")
	db, err := sql.Open("postgres", databaseURL)
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

func updateJobStats(jID uint, db *gorm.DB) {
	updateJobQueueCountCloudwatchMetric()

	var j models.Job
	if err := db.First(&j, jID).Error; err == nil {
		db.Model(&j).Update(models.Job{CompletedJobCount: j.CompletedJobCount + 1})
	}
}

func addJobFileName(fileName, resourceType string, exportJob models.Job, db *gorm.DB) error {
	err := db.Create(&models.JobKey{JobID: exportJob.ID, FileName: fileName, ResourceType: resourceType}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func updateJobQueueCountCloudwatchMetric() {

	// Update the Cloudwatch Metric for job queue count
	env := os.Getenv("DEPLOYMENT_TARGET")
	if env != "" {
		sampler, err := metrics.NewSampler("BCDA", "Count")
		if err != nil {
			fmt.Println("Warning: failed to create new metric sampler...")
		} else {
			err := sampler.PutSample("JobQueueCount", getQueueJobCount(), []metrics.Dimension{
				metrics.Dimension{Name: "Environment", Value: env},
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

	if hInt, err := strconv.Atoi(os.Getenv("WORKER_HEALTH_INT_SEC")); err == nil {
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
