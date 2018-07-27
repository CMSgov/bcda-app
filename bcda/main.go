package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/CMSgov/bcda-app/models"
	"github.com/bgentry/que-go"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx"

	_ "github.com/lib/pq"
)

var (
	qc *que.Client
	db *sql.DB
)

type jobEnqueueArgs struct {
	AcoID int
}

func bulkRequest(w http.ResponseWriter, r *http.Request) {
	acoID := chi.URLParam(r, "acoId")

	i, err := strconv.Atoi(acoID)
	if err != nil {
		log.Fatal(err)
	}

	newJob := models.Job{
		AcoID:    i,
		Location: "",
		Status:   "started",
	}
	if err = newJob.Insert(db); err != nil {
		log.Fatal(err)
	}

	args, err := json.Marshal(jobEnqueueArgs{AcoID: newJob.AcoID})
	if err != nil {
		log.Fatal(err)
	}

	j := &que.Job{
		Type: "ProcessJob",
		Args: args,
	}
	if err = qc.Enqueue(j); err != nil {
		log.Fatal(err)
	}

	jsonData, err := json.Marshal(newJob)
	if err != nil {
		log.Fatal(err)
	}
	_, err = w.Write([]byte(jsonData))
	if err != nil {
		log.Fatal(err)
	}
}

func jobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	i, err := strconv.Atoi(jobID)
	if err != nil {
		log.Fatal(err)
	}

	job, err := models.JobByID(db, i)
	if err != nil {
		log.Fatal(err)
	}

	jsonData, err := json.Marshal(job)
	if err != nil {
		log.Fatal(err)
	}
	_, err = w.Write([]byte(jsonData))
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Worker queue connection
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

	// API db connection
	databaseURL := os.Getenv("DATABASE_URL")
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Starting bcda...")
	err = http.ListenAndServe(":3000", NewRouter())
	if err != nil {
		log.Fatal(err)
	}
}
