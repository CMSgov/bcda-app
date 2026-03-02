package cli

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
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
	fmt.Println("Starting bcdaworker...")
	createWorkerDirs()
	queue := queueing.StartRiver(db, utils.GetEnvInt("WORKER_POOL_SIZE", 4))
	defer queue.StopRiver()
	waitForSig()
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
