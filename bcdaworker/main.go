package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/CMSgov/bcda-app/bcda/responseutils"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
)

var (
	qc *que.Client
)

type jobEnqueueArgs struct {
	ID             int
	AcoID          string
	UserID         string
	BeneficiaryIDs []string
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	filePath := os.Getenv("BCDA_WORKER_ERROR_LOG")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file; using default stderr")
	}
}

func processJob(j *que.Job) error {
	log.Info("Worker started processing job ID ", j.ID)

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

	err = writeEOBDataToFile(bb, jobArgs.AcoID, jobArgs.BeneficiaryIDs)

	if err != nil {
		exportJob.Status = "Failed"
	} else {
		exportJob.Status = "Completed"
	}

	err = db.Save(exportJob).Error
	if err != nil {
		return err
	}

	return nil
}

func writeEOBDataToFile(bb client.APIClient, acoID string, beneficiaryIDs []string) error {
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

	dataDir := os.Getenv("FHIR_PAYLOAD_DIR")
	f, err := os.Create(fmt.Sprintf("%s/%s.ndjson", dataDir, acoID))
	if err != nil {
		log.Error(err)
		return err
	}

	defer f.Close()

	w := bufio.NewWriter(f)

	pData, err := bb.GetExplanationOfBenefitData(beneficiaryIDs[0])
	if err != nil {
		log.Error(err)
		appendErrorToFile(acoID, responseutils.Exception, responseutils.BbErr, "Error retrieving ExplanationOfBenefit")
	} else {
		// Append newline because we'll be writing multiple entries per file later
		_, err := w.WriteString(pData + "\n")
		if err != nil {
			log.Error(err)
		}
	}

	w.Flush()

	return nil
}

func appendErrorToFile(acoID, code, detailsCode, detailsDisplay string) {
	oo := responseutils.CreateOpOutcome(responseutils.Error, code, detailsCode, detailsDisplay)

	dataDir := os.Getenv("FHIR_PAYLOAD_DIR")
	fileName := fmt.Sprintf("%s/%s-error.ndjson", dataDir, acoID)
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

func main() {
	fmt.Println("Starting bcdaworker...")

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
	defer pgxpool.Close()

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

	waitForSig()
}
