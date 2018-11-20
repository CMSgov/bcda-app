package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/encryption"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/jackc/pgx"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/responseutils"
	"github.com/bgentry/que-go"
)

var (
	qc *que.Client
)

type jobEnqueueArgs struct {
	ID             int
	AcoID          string
	UserID         string
	BeneficiaryIDs []string
	// TODO(rnagle): remove `Encrypt` when file encryption functionality is ready for release
	Encrypt bool
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	filePath := os.Getenv("BCDA_WORKER_ERROR_LOG")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file; using default stderr")
	}
}

func processJob(j *que.Job) error {
	log.Info("Worker started processing job ", j.ID)

	db := database.GetGORMDbConnection()
	defer db.Close()

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

	exportJob.Status = "In Progress"
	err = db.Save(exportJob).Error
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
	err = writeEOBDataToFile(bb, jobArgs.AcoID, jobArgs.BeneficiaryIDs, jobID)

	if err != nil {
		exportJob.Status = "Failed"
	} else {
		files, err := ioutil.ReadDir(staging)
		if err != nil {
			log.Error(err)
			return err
		}

		for _, f := range files {
			oldpath := staging + "/" + f.Name()
			newpath := data + "/" + f.Name()
			if _, err := os.Stat(data); os.IsNotExist(err) {
				err = os.Mkdir(data, os.ModePerm)
				if err != nil {
					log.Error(err)
					return err
				}
			}

			// TODO(rnagle): this condition should be removed when file encryption is ready for release
			if !jobArgs.Encrypt {
				err := os.Rename(oldpath, newpath)
				if err != nil {
					log.Error(err)
					return err
				}
			} else {
				// this will be the only code path after ATO
				publicKey := exportJob.Aco.GetPublicKey()
				if publicKey == nil {
					fmt.Println("NO KEY EXISTS  THIS IS BAD")
				}
				err := encryption.EncryptAndMove(staging, data, f.Name(), exportJob.Aco.GetPublicKey(), exportJob.ID)
				if err != nil {
					log.Error(err)
					return err
				}
			}
		}
		os.Remove(staging)
		exportJob.Status = "Completed"
	}

	err = db.Save(exportJob).Error
	if err != nil {
		return err
	}

	log.Info("Worker finished processing job ", j.ID)

	return nil
}

func writeEOBDataToFile(bb client.APIClient, acoID string, beneficiaryIDs []string, jobID string) error {
	re := regexp.MustCompile("[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}")
	if !re.Match([]byte(acoID)) {
		err := errors.New("Invalid ACO ID")
		log.Error(err)
		return err
	}

	if bb == nil {
		err := errors.New("Blue Button client is required")
		log.Error(err)
		return err
	}

	dataDir := os.Getenv("FHIR_STAGING_DIR")
	f, err := os.Create(fmt.Sprintf("%s/%s/%s.ndjson", dataDir, jobID, acoID))
	if err != nil {
		log.Error(err)
		return err
	}

	defer f.Close()

	w := bufio.NewWriter(f)

	for _, beneficiaryID := range beneficiaryIDs {
		pData, err := bb.GetExplanationOfBenefitData(beneficiaryID)
		if err != nil {
			log.Error(err)
			appendErrorToFile(acoID, responseutils.Exception, responseutils.BbErr, fmt.Sprintf("Error retrieving ExplanationOfBenefit for beneficiary %s in ACO %s", beneficiaryID, acoID), jobID)
		} else {
			fhirBundleToResourceNDJSON(w, pData, "ExplanationOfBenefits", beneficiaryID, acoID, jobID)
		}
	}

	w.Flush()

	return nil
}

func appendErrorToFile(acoID, code, detailsCode, detailsDisplay string, jobID string) {
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
}

func fhirBundleToResourceNDJSON(w *bufio.Writer, jsonData, jsonType, beneficiaryID, acoID string, jobID string) {
	var jsonOBJ map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &jsonOBJ)
	if err != nil {
		log.Error(err)
		appendErrorToFile(acoID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error UnMarshaling %s from data for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID), jobID)
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
	waitForSig()
}
