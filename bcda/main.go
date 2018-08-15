package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/models"
	"github.com/urfave/cli"

	"github.com/bgentry/que-go"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/chi"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"
)

var (
	qc *que.Client
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

	db := database.GetDbConnection()
	defer db.Close()

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
	db := database.GetDbConnection()
	defer db.Close()

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
	app := cli.NewApp()
	app.Name = "bcda"
	app.Usage = "Beneficiary Claims Data API CLI"
	var acoName, acoID, userName, userEmail, userID, accessToken string
	app.Commands = []cli.Command{
		{
			Name:  "start-api",
			Usage: "Start the API",
			Action: func(c *cli.Context) error {
				// Worker queue connection
				queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
				pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
				if err != nil {
					return err
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

				fmt.Println("Starting bcda...")
				err = http.ListenAndServe(":3000", NewRouter())
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:     "create-aco",
			Category: "Authentication tools",
			Usage:    "Create an ACO",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "name",
					Usage:       "Name of ACO",
					Destination: &acoName,
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println(createACO(acoName))
				return nil
			},
		},
		{
			Name:     "create-user",
			Category: "Authentication tools",
			Usage:    "Create a user",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "aco-id",
					Usage:       "UUID of user's ACO",
					Destination: &acoID,
				},
				cli.StringFlag{
					Name:        "name",
					Usage:       "Name of user",
					Destination: &userName,
				},
				cli.StringFlag{
					Name:        "email",
					Usage:       "Email address of user",
					Destination: &userEmail,
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println(createUser(acoID, userName, userEmail))
				return nil
			},
		},
		{
			Name:     "create-token",
			Category: "Authentication tools",
			Usage:    "Create an access token",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "aco-id",
					Usage:       "UUID of ACO",
					Destination: &acoID,
				},
				cli.StringFlag{
					Name:        "user-id",
					Usage:       "UUID of user",
					Destination: &userID,
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Println(createAccessToken(acoID, userID))
				return nil
			},
		},
		{
			Name:     "revoke-token",
			Category: "Authentication tools",
			Usage:    "Revoke an access token",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "access-token",
					Usage:       "Access token",
					Destination: &accessToken,
				},
			},
			Action: func(c *cli.Context) error {
				revokeAccessToken(accessToken)
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func createACO(name string) string {
	if name == "" {
		fmt.Println("ACO name (--name) must be provided")
		return ""
	}

	authBackend := auth.InitAuthBackend()
	acoUUID, err := authBackend.CreateACO(name)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return acoUUID.String()
}

func createUser(acoID, name, email string) string {
	errMsgs := []string{}
	var acoUUID uuid.UUID

	if acoID == "" {
		errMsgs = append(errMsgs, "ACO ID (--aco-id) must be provided")
	} else {
		acoUUID = uuid.Parse(acoID)
		if acoUUID == nil {
			errMsgs = append(errMsgs, "ACO ID must be a UUID")
		}
	}
	if name == "" {
		errMsgs = append(errMsgs, "Name (--name) must be provided")
	}
	if email == "" {
		errMsgs = append(errMsgs, "Email address (--email) must be provided")
	}

	if len(errMsgs) > 0 {
		for _, errMsg := range errMsgs {
			fmt.Println(errMsg)
		}
		return ""
	}

	authBackend := auth.InitAuthBackend()
	userUUID, err := authBackend.CreateUser(name, email, acoUUID)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return userUUID.String()
}

func createAccessToken(acoID, userID string) string {
	errMsgs := []string{}
	var acoUUID, userUUID uuid.UUID

	if acoID == "" {
		errMsgs = append(errMsgs, "ACO ID (--aco-id) must be provided")
	} else {
		acoUUID = uuid.Parse(acoID)
		if acoUUID == nil {
			errMsgs = append(errMsgs, "ACO ID must be a UUID")
		}
	}
	if userID == "" {
		errMsgs = append(errMsgs, "User ID (--user-id) must be provided")
	} else {
		userUUID = uuid.Parse(userID)
		if userUUID == nil {
			errMsgs = append(errMsgs, "User ID must be a UUID")
		}
	}

	if len(errMsgs) > 0 {
		for _, errMsg := range errMsgs {
			fmt.Println(errMsg)
		}
		return ""
	}

	authBackend := auth.InitAuthBackend()

	token, err := authBackend.GenerateToken(userID, acoID)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return token
}

func revokeAccessToken(accessToken string) {
	if accessToken == "" {
		fmt.Println("Access token (--access-token) must be provided")
		return
	}

	authBackend := auth.InitAuthBackend()

	success := authBackend.RevokeToken(accessToken)
	if success {
		fmt.Println("Access token has been deactivated")
	}
}
