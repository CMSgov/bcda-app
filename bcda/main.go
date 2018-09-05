package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/urfave/cli"

	"github.com/bgentry/que-go"
	"github.com/dgrijalva/jwt-go"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"
)

var (
	qc *que.Client
)

type jobEnqueueArgs struct {
	ID     int
	AcoID  string
	UserID string
}

type fileItem struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type bulkResponseBody struct {
	TransactionTime     time.Time  `json:"transactionTime"`
	RequestURL          string     `json:"request"`
	RequiresAccessToken bool       `json:"requiresAccessToken"`
	Files               []fileItem `json:"output"`
	Errors              []fileItem `json:"error"`
}

func claimsFromToken(token *jwt.Token) (jwt.MapClaims, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}
	return jwt.MapClaims{}, errors.New("Error determining token claims")
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
				acoUUID, err := createACO(acoName)
				if err != nil {
					return err
				}
				fmt.Println(acoUUID)
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
				userUUID, err := createUser(acoID, userName, userEmail)
				if err != nil {
					return err
				}
				fmt.Println(userUUID)
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
				accessToken, err := createAccessToken(acoID, userID)
				if err != nil {
					return err
				}
				fmt.Println(accessToken)
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
				err := revokeAccessToken(accessToken)
				if err != nil {
					return err
				}
				fmt.Println("Access token has been deactivated")
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

func createACO(name string) (string, error) {
	if name == "" {
		return "", errors.New("ACO name (--name) must be provided")
	}

	authBackend := auth.InitAuthBackend()
	acoUUID, err := authBackend.CreateACO(name)
	if err != nil {
		return "", err
	}

	return acoUUID.String(), nil
}

func createUser(acoID, name, email string) (string, error) {
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
		return "", errors.New(strings.Join(errMsgs, "\n"))
	}

	authBackend := auth.InitAuthBackend()
	userUUID, err := authBackend.CreateUser(name, email, acoUUID)
	if err != nil {
		return "", err
	}

	return userUUID.String(), nil
}

func createAccessToken(acoID, userID string) (string, error) {
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
		return "", errors.New(strings.Join(errMsgs, "\n"))
	}

	authBackend := auth.InitAuthBackend()

	token, err := authBackend.GenerateToken(userID, acoID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func revokeAccessToken(accessToken string) error {
	if accessToken == "" {
		return errors.New("Access token (--access-token) must be provided")
	}

	authBackend := auth.InitAuthBackend()

	return authBackend.RevokeToken(accessToken)
}
