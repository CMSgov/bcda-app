package cli

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/DataDog/dd-trace-go/v2/profiler"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/pborman/uuid"
)

const Name = "bcdaworker"
const Usage = "Beneficiary Claims Data API Worker CLI"

var (
	db *sql.DB
)

func GetApp() *cli.App {
	return setUpApp()
}

func setUpApp() *cli.App {
	app := cli.NewApp()
	app.Name = Name
	app.Usage = Usage
	app.Before = func(c *cli.Context) error {
		log.SetupLoggers()
		client.SetLogger(log.BFDWorker)
		db = database.Connect()
		return nil
	}
	app.Commands = []cli.Command{
		{
			Name:  "start-worker",
			Usage: "Start the worker",
			Action: func(c *cli.Context) error {
				startWorker()
				return nil
			},
		},
		{
			Name:  "health",
			Usage: "Check the worker health",
			Action: func(c *cli.Context) error {
				healthChecker := health.NewHealthChecker(db)
				healthy := checkHealth(healthChecker)
				if healthy {
					return nil
				} else {
					return cli.NewExitError("Worker is unhealthy", 1)
				}
			},
		},
	}
	return app
}

func startWorker() {

	logger := log.NewSlogLogger("worker")
	logger.Info("Starting bcdaworker...")

	err := profiler.Start(
		profiler.WithService("BCDA"),
		profiler.WithEnv(os.Getenv("ENV")),
		profiler.WithVersion(constants.Version),
	)
	if err != nil {
		log.Worker.Warn(err)
	}
	defer profiler.Stop()

	createWorkerDirs()

	ctx := context.Background()
	signalCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	riverClient := queueing.CreateRiverClient(logger, db, utils.GetEnvInt("WORKER_POOL_SIZE", 4))
	if err := riverClient.Start(ctx); err != nil {
		logger.Error("failed to start river client", "error", err)
		panic(err)
	}

	<-signalCtx.Done()
	stop()
	logger.Info("Received exit signal; initiating soft stop (waiting for cancelled jobs to finish)")
	<-riverClient.Stopped()
}

func createWorkerDirs() {
	staging := conf.GetEnv("FHIR_STAGING_DIR")
	err := os.MkdirAll(staging, 0744)
	if err != nil {
		log.Worker.Fatal(err)
	}
	localTemp := conf.GetEnv("FHIR_TEMP_DIR")
	err = os.MkdirAll(localTemp, 0744)
	if err != nil {
		log.Worker.Fatal(err)
	}
	err = clearTempDirectory(localTemp)
	if err != nil {
		log.Worker.Fatal(err)
	}
}

func clearTempDirectory(tempDir string) error {
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == tempDir {
			return nil
		}
		if info.IsDir() {
			return os.RemoveAll(path)
		}
		return os.Remove(path)
	})

	if err != nil {
		return err
	}
	return nil
}

func checkHealth(healthChecker health.HealthChecker) bool {
	entry := log.Health

	logFields := logrus.Fields{}
	logFields["type"] = "health"
	logFields["id"] = uuid.NewRandom()

	_, dbOk := healthChecker.IsWorkerDatabaseOK()
	if dbOk {
		logFields["db"] = "ok"
	} else {
		logFields["db"] = "error"
	}

	bbOk := healthChecker.IsBlueButtonOK()
	if bbOk {
		logFields["bb"] = "ok"
	} else {
		logFields["bb"] = "error"
	}

	entry.WithFields(logFields).Info()
	return dbOk && bbOk
}
