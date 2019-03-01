package main

import (
	"bufio"
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
	// TODO(rnagle): remove `Encrypt` when file encryption functionality is ready for release
	Encrypt bool
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

	fileName, err := writeBBDataToFile(bb, jobArgs.ACOID, jobArgs.BeneficiaryIDs, jobID, jobArgs.ResourceType)

	// THis is only run AFTER completion of all the collection
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
		newpath := data + "/" + fileName
		if _, err := os.Stat(data); os.IsNotExist(err) {
			err = os.Mkdir(data, os.ModePerm)
			if err != nil {
				log.Error(err)
				return err
			}
		}

		// TODO (knollfear): Remove this too when we stop supporting unencrypted files
		if !jobArgs.Encrypt {
			db := database.GetGORMDbConnection()
			defer database.Close(db)
			err = db.Create(&models.JobKey{JobID: uint(jobArgs.ID), EncryptedKey: []byte("NO_ENCRYPTION"), FileName: fileName}).Error
			if err != nil {
				log.Error(err)
				return err
			}

			err := os.Rename(oldpath, newpath)
			if err != nil {
				log.Error(err)
				return err
			}

		} else {
			// this will be the only code path after ATO
			publicKey := exportJob.ACO.GetPublicKey()
			if publicKey == nil {
				fmt.Println("NO KEY EXISTS  THIS IS BAD")
			} else {
				err := encryption.EncryptAndMove(staging, data, fileName, exportJob.ACO.GetPublicKey(), exportJob.ID)
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

	}

	_, err = exportJob.CheckCompleted()
	if err != nil {
		log.Error(err)
		return err
	}

	log.Info("Worker finished processing job ", j.ID)

	return nil
}

func writeBBDataToFile(bb client.APIClient, acoID string, beneficiaryIDs []string, jobID, t string) (fileName string, error error) {
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
	fileName = fmt.Sprintf("%s.ndjson", uuid.NewRandom().String())
	f, err := os.Create(fmt.Sprintf("%s/%s/%s", dataDir, jobID, fileName))
	if err != nil {
		log.Error(err)
		return "", err
	}

	defer f.Close()

	w := bufio.NewWriter(f)
	errorCount := 0
	totalBeneIDs := float64(len(beneficiaryIDs))
	failThreshold := getFailureThreshold()

	for _, beneficiaryID := range beneficiaryIDs {
		pData, err := bbFunc(beneficiaryID, jobID)
		if err != nil {
			log.Error(err)
			errorCount++
			appendErrorToFile(acoID, responseutils.Exception, responseutils.BbErr, fmt.Sprintf("Error retrieving %s for beneficiary %s in ACO %s", t, beneficiaryID, acoID), jobID)
		} else {
			fhirBundleToResourceNDJSON(w, pData, t, beneficiaryID, acoID, jobID)
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

	return fileName, nil
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

func appendErrorToFile(acoID, code, detailsCode, detailsDisplay string, jobID string) {
	segment := newrelic.StartSegment(txn, "appendErrorToFile")

	oo := responseutils.CreateOpOutcome(responseutils.Error, code, detailsCode, detailsDisplay)

	dataDir := os.Getenv("FHIR_STAGING_DIR")
	fileName := fmt.Sprintf("%s/%s/%s-error.ndjson", dataDir, jobID, acoID)
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

func fhirBundleToResourceNDJSON(w *bufio.Writer, jsonData, jsonType, beneficiaryID, acoID, jobID string) {
	segment := newrelic.StartSegment(txn, "fhirBundleToResourceNDJSON")

	var jsonOBJ map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &jsonOBJ)
	if err != nil {
		log.Error(err)
		appendErrorToFile(acoID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error UnMarshaling %s resources from data for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
		return
	}

	entries := jsonOBJ["entry"]

	// There might be no entries.  If this happens we can't iterate over them.
	if entries != nil {

		for _, entry := range entries.([]interface{}) {
			entryJSON, err := json.Marshal(entry)
			// This is unlikely to happen because we just unmarshalled this data a few lines above.
			if err != nil {
				log.Error(err)
				appendErrorToFile(acoID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error Marshaling %s to Json for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
				continue
			}
			_, err = w.WriteString(string(entryJSON) + "\n")
			if err != nil {
				log.Error(err)
				appendErrorToFile(acoID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error writing %s to file for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
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
