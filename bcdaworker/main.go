package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/quego"
	log "github.com/sirupsen/logrus"
)

// var (
// 	qc *que.Client
// )

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

// func processJob(j *que.Job) error {
// 	m := monitoring.GetMonitor()
// 	txn := m.Start("processJob", nil, nil)
// 	ctx := newrelic.NewContext(context.Background(), txn)
// 	defer m.End(txn)

// 	log.Info("Worker started processing job ", j.ID)

// 	// Update the Cloudwatch Metric for job queue count
// 	updateJobQueueCountCloudwatchMetric()

// 	db := database.GetDbConnection()
// 	defer db.Close()
// 	r := postgres.NewRepository(db)

// 	jobArgs := models.JobEnqueueArgs{}
// 	err := json.Unmarshal(j.Args, &jobArgs)
// 	if err != nil {
// 		return err
// 	}

// 	// Verify Jobs have a BB base path
// 	if len(jobArgs.BBBasePath) == 0 {
// 		err = errors.New("empty BBBasePath: Must be set")
// 		log.Error(err)
// 		return err
// 	}

// 	exportJob, err := r.GetJobByID(ctx, uint(jobArgs.ID))
// 	if goerrors.Is(err, repository.ErrJobNotFound) {

// 	}

// 	if err != nil {
// 		return errors.Wrap(err, "could not retrieve job from database")
// 	}

// 	aco, err := r.GetACOByUUID(ctx, exportJob.ACOID)
// 	if err != nil {
// 		return errors.Wrap(err, "could not retrieve ACO from database")
// 	}

// 	err = r.UpdateJobStatusCheckStatus(ctx, exportJob.ID, models.JobStatusPending, models.JobStatusInProgress)
// 	if goerrors.Is(err, repository.ErrJobNotUpdated) {
// 		log.Warnf("Failed to update job. Assume job already updated. Continuing. %s", err.Error())
// 	} else if err != nil {
// 		return errors.Wrap(err, "could not update job status in database")
// 	}

// 	bb, err := client.NewBlueButtonClient(client.NewConfig(jobArgs.BBBasePath))
// 	if err != nil {
// 		err = errors.Wrap(err, "could not create Blue Button client")
// 		log.Error(err)
// 		return err
// 	}

// 	jobID := strconv.Itoa(jobArgs.ID)
// 	stagingPath := fmt.Sprintf("%s/%s", os.Getenv("FHIR_STAGING_DIR"), jobID)
// 	payloadPath := fmt.Sprintf("%s/%s", os.Getenv("FHIR_PAYLOAD_DIR"), jobID)

// 	if err = createDir(stagingPath); err != nil {
// 		log.Error(err)
// 		return err
// 	}

// 	// Create directory for job results.
// 	// This will be used in the clean up later to move over processed files.
// 	if err = createDir(payloadPath); err != nil {
// 		log.Error(err)
// 		return err
// 	}

// 	fileUUID, fileSize, err := writeBBDataToFile(ctx, r, bb, *aco.CMSID, jobArgs)
// 	fileName := fileUUID + ".ndjson"

// 	// This is only run AFTER completion of all the collection
// 	if err != nil {
// 		err = r.UpdateJobStatus(ctx, exportJob.ID, models.JobStatusFailed)
// 		if err != nil {
// 			return err
// 		}
// 	} else {
// 		if fileSize == 0 {
// 			log.Warn("Empty file found in request: ", fileName)
// 			fileName = models.BlankFileName
// 		}

// 		err = addJobFileName(ctx, r, fileName, jobArgs.ResourceType, *exportJob)
// 		if err != nil {
// 			log.Error(err)
// 			return err
// 		}
// 	}

// 	_, err = checkJobCompleteAndCleanup(ctx, r, exportJob.ID)
// 	if err != nil {
// 		log.Error(err)
// 		return err
// 	}

// 	updateJobStats(ctx, r, exportJob.ID)

// 	log.Info("Worker finished processing job ", j.ID)

// 	return nil
// }

// func createDir(path string) error {
// 	if _, err := os.Stat(path); os.IsNotExist(err) {
// 		if err = os.MkdirAll(path, os.ModePerm); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

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

// func setupQueue() *pgx.ConnPool {
// 	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
// 	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
// 		ConnConfig:   pgxcfg,
// 		AfterConnect: que.PrepareStatements,
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	qc = que.NewClient(pgxpool)
// 	wm := que.WorkMap{
// 		"ProcessJob": processJob,
// 	}

// 	workerPoolSize := utils.GetEnvInt("WORKER_POOL_SIZE", 2)
// 	workers := que.NewWorkerPool(qc, wm, workerPoolSize)
// 	go workers.Start()

// 	return pgxpool
// }

// func getQueueJobCount() float64 {
// 	databaseURL := os.Getenv("QUEUE_DATABASE_URL")
// 	db, err := sql.Open("postgres", databaseURL)
// 	if err != nil {
// 		log.Error(err)
// 	}

// 	pingErr := db.Ping()
// 	if pingErr != nil {
// 		log.Error(pingErr)
// 	}
// 	defer db.Close()

// 	row := db.QueryRow(`select count(*) from que_jobs;`)

// 	var count int
// 	if err := row.Scan(&count); err != nil {
// 		log.Error(err)
// 	}

// 	return float64(count)
// }

// func updateJobStats(ctx context.Context, r repository.Repository, jobID uint) {
// 	updateJobQueueCountCloudwatchMetric()

// 	// Not critical since we use the job_keys count as the authoritative list of completed jobs.
// 	// CompletedJobCount is purely information and can be off.
// 	if err := r.IncrementCompletedJobCount(ctx, jobID); err != nil {
// 		log.Warnf("Failed to update completed job count for job %d. Will continue. %s", jobID, err.Error())
// 	}
// }

// func addJobFileName(ctx context.Context, r repository.Repository, fileName, resourceType string, exportJob models.Job) error {
// 	if err := r.CreateJobKey(ctx, models.JobKey{JobID: exportJob.ID, FileName: fileName, ResourceType: resourceType}); err != nil {
// 		log.Error(err)
// 		return err
// 	}
// 	return nil
// }

// func updateJobQueueCountCloudwatchMetric() {

// 	// Update the Cloudwatch Metric for job queue count
// 	env := os.Getenv("DEPLOYMENT_TARGET")
// 	if env != "" {
// 		sampler, err := metrics.NewSampler("BCDA", "Count")
// 		if err != nil {
// 			fmt.Println("Warning: failed to create new metric sampler...")
// 		} else {
// 			err := sampler.PutSample("JobQueueCount", getQueueJobCount(), []metrics.Dimension{
// 				{Name: "Environment", Value: env},
// 			})
// 			if err != nil {
// 				log.Error(err)
// 			}
// 		}
// 	}
// }

// func getSegment(ctx context.Context, name string) newrelic.Segment {
// 	segment := newrelic.Segment{Name: name}
// 	if txn := newrelic.FromContext(ctx); txn != nil {
// 		segment.StartTime = txn.StartSegmentNow()
// 	}
// 	return segment
// }

func main() {
	fmt.Println("Starting bcdaworker...")
	quego.Start(os.Getenv("QUEUE_DATABASE_URL"), utils.GetEnvInt("WORKER_POOL_SIZE", 2))

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
