package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/health"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/pborman/uuid"
	"github.com/sirupsen/logrus"
)

func init() {
	createWorkerDirs()
	client.SetLogger(log.BBWorker)
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

func main() {
	fmt.Println("Starting bcdaworker...")
	queue := queueing.StartRiver(utils.GetEnvInt("WORKER_POOL_SIZE", 4))
	defer queue.StopRiver()

	if hInt, err := strconv.Atoi(conf.GetEnv("WORKER_HEALTH_INT_SEC")); err == nil {
		ticker := time.NewTicker(time.Duration(hInt) * time.Second)
		quit := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					logHealth()
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
	}

	waitForSig()
}

func logHealth() {
	entry := log.Health

	logFields := logrus.Fields{}
	logFields["type"] = "health"
	logFields["id"] = uuid.NewRandom()

	if _, ok := health.IsWorkerDatabaseOK(); ok {
		logFields["db"] = "ok"
	} else {
		logFields["db"] = "error"
	}

	if health.IsBlueButtonOK() {
		logFields["bb"] = "ok"
	} else {
		logFields["bb"] = "error"
	}

	entry.WithFields(logFields).Info()
}
