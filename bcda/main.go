package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/urfave/cli"

	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"
)

// App Name and usage.  Edit them here to prevent breaking tests
const Name = "bcda"
const Usage = "Beneficiary Claims Data API CLI"
const CreateACO = "create-aco"

var (
	qc      *que.Client
	version = "latest"
)

// swagger:ignore
type jobEnqueueArgs struct {
	ID             int
	AcoID          string
	UserID         string
	BeneficiaryIDs []string
}

// swagger:model fileItem
type fileItem struct {
	// KNOLL the type of File returned
	Type string `json:"type"`
	// The URL of the file
	URL string `json:"url"`
}

/*
Bulk Response Body for a completed Bulk Status Request
swagger:response bulkResponseBody
*/
type bulkResponseBody struct {
	// The Time of the Transaction Request
	TransactionTime time.Time `json:"transactionTime"`
	// The URL of the Response
	RequestURL string `json:"request"`
	// Is a token required for this response
	RequiresAccessToken bool `json:"requiresAccessToken"`
	// Files included in the payload
	// collection format: csv
	Files []fileItem `json:"output"`
	// Keys created during encryption of the files for this job
	// These keys are encrypted using the ACO's public key
	Keys []string `json:"keys"`
	// Errors encountered during processing
	// collection format: csv
	Errors []fileItem        `json:"error"`
	KeyMap map[string][]byte `json:"KeyMap"`
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	filePath := os.Getenv("BCDA_ERROR_LOG")
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file; using default stderr")
	}
	monitoring.GetMonitor()
}

func main() {
	app := setUpApp()
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func setUpApp() *cli.App {
	app := cli.NewApp()
	app.Name = Name
	app.Usage = Usage
	app.Version = version
	var acoName, acoID, userName, userEmail, userID, accessToken, ttl string
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
				if os.Getenv("DEBUG") == "true" {
					autoMigrate()
				}

				api := &http.Server{
					Handler:      NewAPIRouter(),
					ReadTimeout:  time.Duration(getEnvInt("API_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout: time.Duration(getEnvInt("API_WRITE_TIMEOUT", 20)) * time.Second,
					IdleTimeout:  time.Duration(getEnvInt("API_IDLE_TIMEOUT", 120)) * time.Second,
				}

				fileserver := &http.Server{
					Handler:      NewDataRouter(),
					ReadTimeout:  time.Duration(getEnvInt("FILESERVER_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout: time.Duration(getEnvInt("FILESERVER_WRITE_TIMEOUT", 360)) * time.Second,
					IdleTimeout:  time.Duration(getEnvInt("FILESERVER_IDLE_TIMEOUT", 120)) * time.Second,
				}

				smux := NewServiceMux(":3000")
				smux.AddServer(fileserver, "/data")
				smux.AddServer(api, "")
				smux.Serve()

				return nil
			},
		},
		{
			Name:     CreateACO,
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
		{
			Name:     "create-alpha-token",
			Category: "Alpha tools",
			Usage:    "Create a disposable alpha participant token",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "ttl",
					Usage:       "Set custom Time To Live in hours",
					Destination: &ttl,
				},
			},
			Action: func(c *cli.Context) error {
				accessToken, err := createAlphaToken(ttl)
				if err != nil {
					return err
				}
				fmt.Println(accessToken)
				return nil
			},
		},
		{
			Name:     "sql-migrate",
			Category: "Database tools",
			Usage:    "Migrate GORM schema changes to the DB",
			Action: func(c *cli.Context) error {
				autoMigrate()
				return nil
			},
		},
		{
			Name:     "remove-expired-jobs",
			Category: "Remove expired jobs",
			Usage:    "Updates job statuses and moves files to an inaccessible location",
			Action: func(c *cli.Context) error {
				threshold := getEnvInt("EXPIRED_THRESHOLD_HR", 24)
				removeExpired(threshold)
				return nil
			},
		},
	}
	return app
}

func autoMigrate() {
	fmt.Println("Initializing Database")
	models.InitializeGormModels()
	auth.InitializeGormModels()
	fmt.Println("Completed Database Initialization")
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
	var userUUID string

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
		return userUUID, errors.New(strings.Join(errMsgs, "\n"))
	}

	authBackend := auth.InitAuthBackend()
	user, err := authBackend.CreateUser(name, email, acoUUID)
	if err != nil {
		return userUUID, err
	}

	return user.UUID.String(), nil
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
	// !todo  This does the wrong thing.  A valid token string is created but not persisted to the db and can't then be reused
	token, err := authBackend.GenerateTokenString(userID, acoID)
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

func createAlphaToken(ttl string) (string, error) {
	authBackend := auth.InitAuthBackend()
	tokenString, err := authBackend.CreateAlphaToken(ttl)
	claims := authBackend.GetJWTClaims(tokenString)

	if claims == nil {
		return "", errors.New("Could not read token claims")
	}

	expiresOn := time.Unix(int64(claims["exp"].(float64)), 0).Format(time.RFC850)
	tokenId := claims["id"].(string)

	return fmt.Sprintf("%s\n%s\n%s", expiresOn, tokenId, tokenString), err
}

func getEnvInt(varName string, defaultVal int) int {
	v := os.Getenv(varName)
	if v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return defaultVal
}

func removeExpired(hrThreshold int) {
	log.Info("Cleaning up expired jobs...")
	db := database.GetGORMDbConnection()
	defer db.Close()

	var jobs []models.Job
	err := db.Find(&jobs, "status = ?", "Completed").Error
	if err != nil {
		log.Error(err)
	}

	expDir := os.Getenv("FHIR_EXPIRED_DIR")
	if _, err = os.Stat(expDir); os.IsNotExist(err) {
		err = os.MkdirAll(expDir, os.ModePerm)
		if err != nil {
			log.Error(err)
		}
	}

	for _, j := range jobs {
		t := j.CreatedAt
		elapsed := time.Since(t).Hours()
		if int(elapsed) >= hrThreshold {

			id := int(j.ID)
			jobDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_PAYLOAD_DIR"), id)
			expDir = fmt.Sprintf("%s/%d", os.Getenv("FHIR_EXPIRED_DIR"), id)

			err = os.Rename(jobDir, expDir)
			if err != nil {
				log.Error(err)
				continue
			}

			j.Status = "Expired"
			err = db.Save(j).Error
			if err != nil {
				log.Error(err)
			}
		}
	}
}
