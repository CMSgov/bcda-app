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
	AcoId int
}

func bulkRequest(w http.ResponseWriter, r *http.Request) {
	acoId := chi.URLParam(r, "acoId")

	i, err := strconv.Atoi(fmt.Sprintf("%s", acoId))
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

	args, err := json.Marshal(jobEnqueueArgs{AcoId: newJob.AcoID})
	if err != nil {
		log.Fatal(err)
	}

	j := &que.Job{
		Type: "ProcessJob",
		Args: args,
	}
	if err := qc.Enqueue(j); err != nil {
		log.Fatal(err)
	}

	jsonData, err := json.Marshal(newJob)
	if err != nil {
		log.Fatal(err)
	}
	w.Write([]byte(jsonData))
}

func jobStatus(w http.ResponseWriter, r *http.Request) {
	jobId := chi.URLParam(r, "jobId")

	i, err := strconv.Atoi(fmt.Sprintf("%s", jobId))
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
	w.Write([]byte(jsonData))
}

func main() {
	// Worker queue connection
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

	// API db connection
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err = sql.Open("postgres", databaseUrl)
	defer db.Close()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting bcda...")
	http.ListenAndServe(":3000", NewRouter())
}
