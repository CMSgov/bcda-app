package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/models"

	"github.com/bgentry/que-go"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"

	_ "github.com/lib/pq"
)

var (
	qc *que.Client
	db *sql.DB
)

type jobEnqueueArgs struct {
	AcoID  string
	UserID string
}

func claimsFromToken(token *jwt.Token) (jwt.MapClaims, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return jwt.MapClaims{}, errors.New("Error determining token claims")
}

func bulkRequest(w http.ResponseWriter, r *http.Request) {
	var (
		claims jwt.MapClaims
		err    error
	)

	t := r.Context().Value("token")
	if token, ok := t.(*jwt.Token); ok && token.Valid {
		claims, err = claimsFromToken(token)
		if err != nil {
			log.Fatal(err)
		}
	}

	acoId, _ := claims["aco"].(string)
	userId, _ := claims["sub"].(string)

	newJob := models.Job{
		AcoID:    uuid.Parse(acoId),
		UserID:   uuid.Parse(userId),
		Location: "",
		Status:   "started",
	}
	if err := newJob.Insert(db); err != nil {
		log.Fatal(err)
	}

	args, err := json.Marshal(jobEnqueueArgs{
		AcoID:  acoId,
		UserID: userId,
	})
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_, err = w.Write([]byte(jsonData))
	if err != nil {
		log.Fatal(err)
	}
}

func getToken(w http.ResponseWriter, r *http.Request) {
	authBackend := auth.InitAuthBackend()

	// Generates a token for fake user and ACO combination
	token, err := authBackend.GenerateToken(
		"82503A18-BF3B-436D-BA7B-BAE09B7FFD2F",
		"DBBD1CE1-AE24-435C-807D-ED45953077D3",
	)
	if err != nil {
		log.Fatal(err)
	}
	_, err = w.Write([]byte(token))
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
