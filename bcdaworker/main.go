package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

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
	beneficiaryIds := jobArgs.BeneficiaryIDs

	bb, err := client.NewBlueButtonClient()
	if err != nil {
		log.Error(err)
		return err
	}

	f, err := os.Create(fmt.Sprintf("data/%s.ndjson", jobArgs.AcoID))
	if err != nil {
		log.Error(err)
	}

	defer f.Close()

	w := bufio.NewWriter(f)

	pData, err := bb.GetExplanationOfBenefitData(beneficiaryIds[0])
	if err != nil {
		log.Error(err)
	} else {
		_, err := w.WriteString(pData + "\n")
		if err != nil {
			log.Error(err)
		}
	}

	w.Flush()

	exportJob.Status = "Completed"
	err = db.Save(exportJob).Error
	if err != nil {
		return err
	}

	return nil
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
