package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/monitoring"
	"github.com/CMSgov/bcda-app/bcda/servicemux"

	que "github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
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
	ACOID          string
	UserID         string
	BeneficiaryIDs []string
	ResourceType   string
	// TODO: remove `Encrypt` when file encryption disable functionality is ready to be deprecated
	Encrypt bool
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
	// Errors encountered during processing
	// collection format: csv
	Errors []fileItem        `json:"error"`
	KeyMap map[string]string `json:"KeyMap"`
	JobID  uint
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	filePath := os.Getenv("BCDA_ERROR_LOG")

	/* #nosec -- 0640 permissions required for Splunk ingestion */
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)

	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to open error log file; using default stderr")
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
	var acoName, acoID, userName, userEmail, userID, accessToken, ttl, threshold, acoSize string
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

				fmt.Fprintf(app.Writer, "%s\n", "Starting bcda...")
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

				smux := servicemux.New(":3000")
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
				fmt.Fprintf(app.Writer, "%s\n", acoUUID)
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
				fmt.Fprintf(app.Writer, "%s\n", userUUID)
				return nil
			},
		},
		{
			Name:     "create-token",
			Category: "Authentication tools",
			Usage:    "Create an access token",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "user-id",
					Usage:       "UUID of user",
					Destination: &userID,
				},
			},
			Action: func(c *cli.Context) error {
				accessToken, err := createAccessToken(userID)
				if err != nil {
					return err
				}
				fmt.Fprintf(app.Writer, "%s\n", accessToken)
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
				fmt.Fprintf(app.Writer, "%s\n", "Access token has been deactivated")
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
				cli.StringFlag{
					Name:        "size",
					Usage:       "Set the size of the ACO.  Must be one of 'Dev', 'Small', 'Medium', or 'Large'",
					Destination: &acoSize,
				},
			},
			Action: func(c *cli.Context) error {
				if ttl == "" {
					ttl = os.Getenv("JWT_EXPIRATION_DELTA")
					if ttl == "" {
						ttl = "72"
					}
				}
				ttlInt, err := validateAlphaTokenInputs(ttl, acoSize)
				if err != nil {
					return err
				}
				accessToken, err := createAlphaToken(ttlInt, acoSize)
				if err != nil {
					return err
				}
				fmt.Fprintf(app.Writer, "%s\n", accessToken)
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
			Name:     "archive-job-files",
			Category: "Archive files for jobs that are expiring",
			Usage:    "Updates job statuses and moves files to an inaccessible location",
			Action: func(c *cli.Context) error {
				threshold := getEnvInt("ARCHIVE_THRESHOLD_HR", 24)
				return archiveExpiring(threshold)
			},
		},
		{
			Name:     "cleanup-archive",
			Category: "Cleanup archive for jobs that have expired",
			Usage:    "Removes job directory and files from archive and updates job status to Expired",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "threshold",
					Usage:       "How long files should wait in archive before deletion",
					Destination: &threshold,
				},
			},
			Action: func(c *cli.Context) error {
				th, err := strconv.Atoi(threshold)
				if err != nil {
					return err
				}
				return cleanupArchive(th)
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

	acoUUID, err := models.CreateACO(name)
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

	user, err := models.CreateUser(name, email, acoUUID)
	if err != nil {
		return userUUID, err
	}

	return user.UUID.String(), nil
}

func createAccessToken(userID string) (string, error) {
	errMsgs := []string{}
	var userUUID uuid.UUID

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

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	var user models.User

	if db.First(&user, "UUID = ?", userID).RecordNotFound() {
		return "", fmt.Errorf("unable to locate User with id of %s", userID)
	}

	params := fmt.Sprintf(`{"clientID" : "%s", "ttl" : %d}`, user.ACOID.String(), 72)
	token, err := auth.GetAuthProvider().RequestAccessToken([]byte(params))
	if err != nil {
		return "", err
	}

	return token.TokenString, nil
}

func revokeAccessToken(accessToken string) error {
	if accessToken == "" {
		return errors.New("Access token (--access-token) must be provided")
	}

	authProvider := auth.GetAuthProvider()

	return authProvider.RevokeAccessToken(accessToken)
}

func validateAlphaTokenInputs(ttl, acoSize string) (int, error) {
	i, err := strconv.Atoi(ttl)
	if err != nil || i <= 0 {
		return i, fmt.Errorf("invalid argument '%v' for --ttl; should be an integer > 0", ttl)
	}
	switch strings.ToLower(acoSize) {
	case
		"dev",
		"small",
		"medium",
		"large":
		return i, nil
	default:
		return i, errors.New("invalid argument for --size.  Please use 'Dev', 'Small', 'Medium', or 'Large'")
	}
}

func createAlphaToken(ttl int, acoSize string) (s string, err error) {

	aco, err := createAlphaEntities(acoSize)
	if err != nil {
		return
	}

	authProvider := auth.GetAuthProvider()

	params := fmt.Sprintf(`{"clientID" : "%s"}`, aco.UUID.String())
	result, err := authProvider.RegisterClient([]byte(params))
	if err != nil {
		return "", fmt.Errorf("could not register client for %s (%s) because %s", aco.UUID.String(), aco.Name, err.Error())
	}
	aco.ClientID, err = auth.GetParamString(result, "clientID")
	if err != nil {
		return "", fmt.Errorf("could not register client for %s (%s) because %s", aco.UUID.String(), aco.Name, err.Error())
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err = db.Save(&aco).Error
	if err != nil {
		return "", fmt.Errorf("could not save ClientID %s to ACO %s (%s) because %s", aco.ClientID, aco.UUID.String(), aco.Name, err.Error())
	}

	params = fmt.Sprintf(`{"clientID" : "%s", "ttl" : %d}`, aco.ClientID, ttl)
	token, err := authProvider.RequestAccessToken([]byte(params))
	if err != nil {
		return "", err
	}

	expiresOn := time.Unix(token.ExpiresOn, 0).Format(time.RFC850)
	tokenId := token.UUID.String()
	return fmt.Sprintf("%s\n%s\n%s", expiresOn, tokenId, token.TokenString), err
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

func archiveExpiring(hrThreshold int) error {
	log.Info("Archiving expiring job files...")
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var jobs []models.Job
	err := db.Find(&jobs, "status = ?", "Completed").Error
	if err != nil {
		log.Error(err)
		return err
	}

	expDir := os.Getenv("FHIR_ARCHIVE_DIR")
	if _, err = os.Stat(expDir); os.IsNotExist(err) {
		err = os.MkdirAll(expDir, os.ModePerm)
		if err != nil {
			log.Error(err)
			return err
		}
	}

	var lastJobError error
	for _, j := range jobs {
		t := j.CreatedAt
		elapsed := time.Since(t).Hours()
		if int(elapsed) >= hrThreshold {

			id := int(j.ID)
			jobDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_PAYLOAD_DIR"), id)
			expDir = fmt.Sprintf("%s/%d", os.Getenv("FHIR_ARCHIVE_DIR"), id)

			err = os.Rename(jobDir, expDir)
			if err != nil {
				log.Error(err)
				lastJobError = err
				continue
			}

			j.Status = "Archived"
			err = db.Save(j).Error
			if err != nil {
				log.Error(err)
				lastJobError = err
			}
		}
	}

	return lastJobError
}

func cleanupArchive(hrThreshold int) error {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	expDir := os.Getenv("FHIR_ARCHIVE_DIR")
	if _, err := os.Stat(expDir); os.IsNotExist(err) {
		// nothing to do if no base directory exists.
		return nil
	}

	maxDate := time.Now().Add(-(time.Hour * time.Duration(hrThreshold)))

	var jobs []models.Job
	err := db.Find(&jobs, "status = ? AND updated_at <= ?", "Archived", maxDate).Error
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		log.Info("No archived job files to clean")
		return nil
	}

	for _, job := range jobs {
		t := job.UpdatedAt
		elapsed := time.Since(t).Hours()
		if int(elapsed) >= hrThreshold {

			id := int(job.ID)
			jobArchiveDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_ARCHIVE_DIR"), id)

			err = os.RemoveAll(jobArchiveDir)
			if err != nil {
				e := fmt.Sprintf("Unable to remove %s because %s", jobArchiveDir, err)
				log.Error(e)
				continue
			}

			job.Status = "Expired"
			err = db.Save(job).Error
			if err != nil {
				return err
			}

			log.WithFields(log.Fields{
				"job_began":     job.CreatedAt,
				"files_removed": time.Now(),
				"job_id":        job.ID,
			}).Info("Files cleaned from archive and job status set to Expired")
		}
	}

	return nil
}

func createAlphaEntities(acoSize string) (aco models.ACO, err error) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			if tx.Error != nil {
				tx.Rollback()
			}
			err = fmt.Errorf("createAlphaEntities failed because %s", r)
		}
	}()

	if tx.Error != nil {
		return aco, tx.Error
	}

	aco, err = models.CreateAlphaACO(tx)
	if err != nil {
		tx.Rollback()
		return aco, err
	}

	if err = models.AssignAlphaBeneficiaries(tx, aco, acoSize); err != nil {
		tx.Rollback()
		return aco, err
	}

	if _, err = models.CreateAlphaUser(tx, aco); err != nil {
		tx.Rollback()
		return aco, err
	}

	if tx.Commit().Error != nil {
		tx.Rollback()
		return aco, tx.Error
	}

	return aco, nil
}
