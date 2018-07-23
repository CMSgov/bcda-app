package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
)

var (
	qc *que.Client
)

func processJob(j *que.Job) error {
	fmt.Printf("Worker started processing job (ID: %d, Args: %s)\n", j.ID, j.Args)
	return nil
}

func waitForSig() {
	signal_chan := make(chan os.Signal, 1)
	defer close(signal_chan)

	signal.Notify(signal_chan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exit_chan := make(chan int)
	defer close(exit_chan)

	go func() {
		for {
			s := <-signal_chan
			switch s {
			case syscall.SIGINT:
				fmt.Println("interrupt")
				exit_chan <- 0
			case syscall.SIGTERM:
				fmt.Println("force stop")
				exit_chan <- 0
			case syscall.SIGQUIT:
				fmt.Println("stop and core dump")
				exit_chan <- 0
			}
		}
	}()

	code := <-exit_chan
	os.Exit(code)
}

func main() {
	fmt.Println("Starting bcdaworker...")

	queueDatabaseUrl := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseUrl)
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
