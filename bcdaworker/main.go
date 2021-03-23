package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/manager"
	"github.com/CMSgov/bcda-app/conf"
	log "github.com/sirupsen/logrus"
)

func init() {
	createWorkerDirs()
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
	filePath := conf.GetEnv("BCDA_WORKER_ERROR_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to open worker error log file; using default stderr")
	}
}

func createWorkerDirs() {
	staging := conf.GetEnv("FHIR_STAGING_DIR")
	err := os.MkdirAll(staging, 0744)
	if err != nil {
		log.Fatal(err)
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
	queue := manager.StartQue(log.StandardLogger(), utils.GetEnvInt("WORKER_POOL_SIZE", 2))
	defer queue.StopQue()

	if hInt, err := strconv.Atoi(conf.GetEnv("WORKER_HEALTH_INT_SEC")); err == nil {
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
