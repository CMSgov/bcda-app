package main

import (
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/bcdagorm"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	que "github.com/bgentry/que-go"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
)

func bulkRequest(w http.ResponseWriter, r *http.Request) {
	var (
		claims jwt.MapClaims
		err    error
	)

	db := database.GetGORMDbConnection()
	defer db.Close()

	t := r.Context().Value("token")
	if token, ok := t.(*jwt.Token); ok && token.Valid {
		claims, err = auth.ClaimsFromToken(token)
		if err != nil {
			log.Fatal(err)
		}
	}

	acoId, _ := claims["aco"].(string)
	userId, _ := claims["sub"].(string)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	newJob := bcdagorm.Job{
		AcoID:      uuid.Parse(acoId),
		UserID:     uuid.Parse(userId),
		RequestURL: fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		Status:     "Pending",
	}
	if err := db.Save(newJob); err != nil {
		log.Fatal(err)
	}

	args, err := json.Marshal(jobEnqueueArgs{
		ID:     int(newJob.ID),
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

	w.Header().Set("Content-Location", fmt.Sprintf("%s://%s/api/v1/jobs/%d", scheme, r.Host, newJob.ID))
	w.WriteHeader(http.StatusAccepted)
}

func jobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	db := database.GetGORMDbConnection()
	defer db.Close()

	i, err := strconv.Atoi(jobID)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(400), 400)
		return
	}
	var job bcdagorm.Job
	err = db.First(&job, i).Error
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(404), 404)
		return
	}

	switch job.Status {
	case "Pending":
		fallthrough
	case "In Progress":
		w.Header().Set("X-Progress", job.Status)
		w.WriteHeader(http.StatusAccepted)
	case "Failed":
		http.Error(w, http.StatusText(500), 500)
	case "Completed":
		w.Header().Set("Content-Type", "application/json")

		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		fi := fileItem{
			Type: "ExplanationOfBenefit",
			URL:  fmt.Sprintf("%s://%s/data/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson", scheme, r.Host),
		}

		rb := bulkResponseBody{
			TransactionTime:     time.Now(),
			RequestURL:          job.RequestURL,
			RequiresAccessToken: true,
			Files:               []fileItem{fi},
			Errors:              []fileItem{},
		}

		jsonData, err := json.Marshal(rb)
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		_, err = w.Write([]byte(jsonData))
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
}

func serveData(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/DBBD1CE1-AE24-435C-807D-ED45953077D3.ndjson")
}

func getToken(w http.ResponseWriter, r *http.Request) {
	authBackend := auth.InitAuthBackend()

	// Generates a token for fake user and ACO combination
	token, err := authBackend.GenerateTokenString(
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
