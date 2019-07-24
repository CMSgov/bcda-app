package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pborman/uuid"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx"
	"github.com/newrelic/go-agent"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/encryption"
	"github.com/CMSgov/bcda-app/bcda/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/bgentry/que-go"
)

var (
	qc  *que.Client
	txn newrelic.Transaction
)

type jobEnqueueArgs struct {
	ID             int
	ACOID          string
	UserID         string
	BeneficiaryIDs []string
	ResourceType   string
}

func init() {
	createWorkerDirs()
	log.SetFormatter(&log.JSONFormatter{})
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
	txn = m.Start("processJob", nil, nil)
	defer m.End(txn)

	log.Info("Worker started processing job ", j.ID)

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

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	jobArgs := jobEnqueueArgs{}
	err := json.Unmarshal(j.Args, &jobArgs)
	if err != nil {
		return err
	}

	var exportJob models.Job
	err = db.First(&exportJob, "ID = ?", jobArgs.ID).Error
	if err != nil {
		return err
	}

	var aco models.ACO
	err = db.First(&aco, "uuid = ?", exportJob.ACOID).Error
	if err != nil {
		return err
	}

	err = db.Model(&exportJob).Where("status = ?", "Pending").Update("status", "In Progress").Error
	if err != nil {
		return err
	}

	bb, err := client.NewBlueButtonClient()
	if err != nil {
		log.Error(err)
		return err
	}

	jobID := strconv.Itoa(jobArgs.ID)
	staging := fmt.Sprintf("%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID)
	data := fmt.Sprintf("%s/%s", os.Getenv("FHIR_PAYLOAD_DIR"), jobID)

	if _, err := os.Stat(staging); os.IsNotExist(err) {
		err = os.MkdirAll(staging, os.ModePerm)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	fileUUID, err := writeBBDataToFile(bb, jobArgs.ACOID, *aco.CMSID, jobArgs.BeneficiaryIDs, jobID, jobArgs.ResourceType)
	fileName := fileUUID + ".ndjson"

	// This is only run AFTER completion of all the collection
	if err != nil {

		err = db.Model(&exportJob).Update("status", "Failed").Error
		if err != nil {
			return err
		}

	} else {
		_, err := ioutil.ReadDir(staging)
		if err != nil {
			log.Error(err)
			return err
		}

		oldpath := staging + "/" + fileName
		if _, err := os.Stat(data); os.IsNotExist(err) {
			err = os.Mkdir(data, os.ModePerm)
			if err != nil {
				log.Error(err)
				return err
			}
		}

		var aco models.ACO
		err = db.Model(&exportJob).Association("ACO").Find(&aco).Error
		if err != nil {
			log.Error("error getting ACO for job:", err.Error())
		}
		if aco.PublicKey == "" {
			log.Error("no public key found for ACO", aco.UUID.String())
		}
		publicKey, err := aco.GetPublicKey()
		if err != nil {
			log.Error("error getting public key: ", err.Error(), aco.PublicKey)
		} else {
			err := encryption.EncryptAndMove(staging, data, fileName, publicKey, exportJob.ID)
			if err != nil {
				log.Error(err)
				return err
			}
		}
		err = os.Remove(oldpath)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	_, err = exportJob.CheckCompletedAndCleanup()
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Worker finished processing job ", j.ID)

	return nil
}

func writeBBDataToFile(bb client.APIClient, acoID string, cmsID string, cclfBeneficiaryIDs []string, jobID, t string) (fileUUID string, error error) {
	segment := newrelic.StartSegment(txn, "writeBBDataToFile")

	if bb == nil {
		err := errors.New("Blue Button client is required")
		log.Error(err)
		return "", err
	}

	// TODO: Should this error be returned or written to file?
	var bbFunc client.BeneDataFunc
	switch t {
	case "ExplanationOfBenefit":
		bbFunc = bb.GetExplanationOfBenefitData
	case "Patient":
		bbFunc = bb.GetPatientData
	case "Coverage":
		bbFunc = bb.GetCoverageData
	default:
		err := fmt.Errorf("Invalid resource type requested: %s", t)
		log.Error(err)
		return "", err
	}

	re := regexp.MustCompile("[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}")
	if !re.Match([]byte(acoID)) {
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

	defer f.Close()

	w := bufio.NewWriter(f)
	errorCount := 0
	totalBeneIDs := float64(len(cclfBeneficiaryIDs))
	failThreshold := getFailureThreshold()
	db := database.GetGORMDbConnection()
	defer db.Close()
	for _, cclfBeneficiaryID := range cclfBeneficiaryIDs {
		var cclfBeneficiary models.CCLFBeneficiary
		db.First(&cclfBeneficiary, cclfBeneficiaryID)
		blueButtonID, err := cclfBeneficiary.GetBlueButtonID(bb)
		if err != nil {
			log.Error(err)
			errorCount++
			appendErrorToFile(fileUUID, responseutils.Exception, responseutils.BbErr, fmt.Sprintf("Error retrieving BlueButton ID for cclfBeneficiary %s", cclfBeneficiaryID), jobID)
		} else {
			cclfBeneficiary.BlueButtonID = blueButtonID
			db.Save(&cclfBeneficiary)
			pData, err := bbFunc(blueButtonID, jobID, cmsID)
			if err != nil {
				log.Error(err)
				errorCount++
				appendErrorToFile(fileUUID, responseutils.Exception, responseutils.BbErr, fmt.Sprintf("Error retrieving %s for beneficiary %s in ACO %s", t, blueButtonID, acoID), jobID)
			} else {
				fhirBundleToResourceNDJSON(w, pData, t, cclfBeneficiaryID, acoID, jobID, fileUUID)
			}
		}
		failPct := (float64(errorCount) / totalBeneIDs) * 100
		if failPct >= failThreshold {
			return "", errors.New("number of failed requests has exceeded threshold")
		}
	}

	err = w.Flush()
	if err != nil {
		return "", err
	}

	err = segment.End()
	if err != nil {
		log.Error(err)
	}

	return fileUUID, nil
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

func appendErrorToFile(fileUUID, code, detailsCode, detailsDisplay string, jobID string) {
	segment := newrelic.StartSegment(txn, "appendErrorToFile")

	oo := responseutils.CreateOpOutcome(responseutils.Error, code, detailsCode, detailsDisplay)

	dataDir := os.Getenv("FHIR_STAGING_DIR")
	fileName := fmt.Sprintf("%s/%s/%s-error.ndjson", dataDir, jobID, fileUUID)
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

	if err != nil {
		log.Error(err)
	}

	defer f.Close()

	ooBytes, err := json.Marshal(oo)
	if err != nil {
		log.Error(err)
	}

	if _, err = f.WriteString(string(ooBytes) + "\n"); err != nil {
		log.Error(err)
	}

	err = segment.End()
	if err != nil {
		log.Error(err)
	}
}

func fhirBundleToResourceNDJSON(w *bufio.Writer, jsonData, jsonType, beneficiaryID, acoID, jobID, fileUUID string) {
	segment := newrelic.StartSegment(txn, "fhirBundleToResourceNDJSON")

	var jsonOBJ map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &jsonOBJ)
	if err != nil {
		log.Error(err)
		appendErrorToFile(fileUUID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error unmarshaling %s resources from data for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
		return
	}
	entries := jsonOBJ["entry"]

	// There might be no entries.  If this happens we can't iterate over them.
	if entries != nil {
		for _, entry := range entries.([]interface{}) {
			entrymap := entry.(map[string]interface{})
			if len(entrymap) != 0 {
				entryJSON, err := json.Marshal(entrymap["resource"])
				// This is unlikely to happen because we just unmarshalled this data a few lines above.
				if err != nil {
					log.Error(err)
					appendErrorToFile(fileUUID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error marshaling %s to JSON for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
					continue
				}
				_, err = w.WriteString(string(entryJSON) + "\n")
				if err != nil {
					log.Error(err)
					appendErrorToFile(fileUUID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error writing %s to file for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
				}
			}
		}
	}

	err = segment.End()
	if err != nil {
		log.Error(err)
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

	var workerPoolSize int
	if len(os.Getenv("WORKER_POOL_SIZE")) == 0 {
		workerPoolSize = 2
	} else {
		workerPoolSize, err = strconv.Atoi(os.Getenv("WORKER_POOL_SIZE"))
		if err != nil {
			log.Fatal(err)
		}
	}

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
