package bcdacli

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	cclfUtils "github.com/CMSgov/bcda-app/bcda/cclf/testutils"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web"
	"github.com/bgentry/que-go"
	"github.com/jackc/pgx"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// App Name and usage.  Edit them here to prevent breaking tests
const Name = "bcda"
const Usage = "Beneficiary Claims Data API CLI"

var qc *que.Client

func GetApp() *cli.App {
	return setUpApp()
}

func setUpApp() *cli.App {
	app := cli.NewApp()
	app.Name = Name
	app.Usage = Usage
	app.Version = constants.Version
	var acoName, acoCMSID, acoID, userName, userEmail, accessToken, ttl, threshold, acoSize, filePath, dirToDelete, environment string
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

				web.SetQC(qc)

				fmt.Fprintf(app.Writer, "%s\n", "Starting bcda...")
				if os.Getenv("DEBUG") == "true" {
					autoMigrate()
				}

				// Accepts and redirects HTTP requests to HTTPS
				srv := &http.Server{
					Handler:      web.NewHTTPRouter(),
					Addr:         ":3001",
					ReadTimeout:  5 * time.Second,
					WriteTimeout: 5 * time.Second,
				}
				go func() { log.Fatal(srv.ListenAndServe()) }()

				auth := &http.Server{
					Handler:      web.NewAuthRouter(),
					ReadTimeout:  time.Duration(utils.GetEnvInt("API_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout: time.Duration(utils.GetEnvInt("API_WRITE_TIMEOUT", 20)) * time.Second,
					IdleTimeout:  time.Duration(utils.GetEnvInt("API_IDLE_TIMEOUT", 120)) * time.Second,
				}

				api := &http.Server{
					Handler:      web.NewAPIRouter(),
					ReadTimeout:  time.Duration(utils.GetEnvInt("API_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout: time.Duration(utils.GetEnvInt("API_WRITE_TIMEOUT", 20)) * time.Second,
					IdleTimeout:  time.Duration(utils.GetEnvInt("API_IDLE_TIMEOUT", 120)) * time.Second,
				}

				fileserver := &http.Server{
					Handler:      web.NewDataRouter(),
					ReadTimeout:  time.Duration(utils.GetEnvInt("FILESERVER_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout: time.Duration(utils.GetEnvInt("FILESERVER_WRITE_TIMEOUT", 360)) * time.Second,
					IdleTimeout:  time.Duration(utils.GetEnvInt("FILESERVER_IDLE_TIMEOUT", 120)) * time.Second,
				}

				smux := servicemux.New(":3000")
				smux.AddServer(fileserver, "/data")
				smux.AddServer(auth, "/auth")
				smux.AddServer(api, "")
				smux.Serve()

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
				cli.StringFlag{
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				acoUUID, err := cclf.CreateACO(acoName, acoCMSID)
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
			Name:     "save-public-key",
			Category: "Authentication tools",
			Usage:    "Upload an ACO's public key to the database",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
					Destination: &acoCMSID,
				},
				cli.StringFlag{
					Name:        "key-file",
					Usage:       "Location of public key in PEM format",
					Destination: &filePath,
				},
			},
			Action: func(c *cli.Context) error {
				if acoCMSID == "" {
					fmt.Fprintf(app.Writer, "cms-id is required\n")
					return errors.New("cms-id is required")
				}

				if filePath == "" {
					fmt.Fprintf(app.Writer, "key-file is required\n")
					return errors.New("key-file is required")
				}

				aco, err := auth.GetACOByCMSID(acoCMSID)
				if err != nil {
					fmt.Fprintf(app.Writer, "Unable to find ACO %s: %s\n", acoCMSID, err.Error())
					return err
				}

				f, err := os.Open(filepath.Clean(filePath))
				if err != nil {
					fmt.Fprintf(app.Writer, "Unable to open file %s: %s\n", filePath, err.Error())
					return err
				}
				reader := bufio.NewReader(f)

				err = aco.SavePublicKey(reader)
				if err != nil {
					fmt.Fprintf(app.Writer, "Unable to save public key for ACO %s: %s\n", acoCMSID, err.Error())
					return err
				}

				fmt.Fprintf(app.Writer, "Public key saved for ACO %s\n", acoCMSID)
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
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				if ttl == "" {
					ttl = os.Getenv("JWT_EXPIRATION_DELTA")
					if ttl == "" {
						ttl = "72"
					}
				}
				ttlInt, err := validateAlphaTokenInputs(ttl, acoCMSID)
				if err != nil {
					return err
				}
				accessToken, err := auth.CreateAlphaToken(ttlInt, acoCMSID)
				if err != nil {
					return err
				}
				fmt.Fprintf(app.Writer, "%s\n", accessToken)
				return nil
			},
		},
		{
			Name:     "generate-client-credentials",
			Category: "Authentication tools",
			Usage:    "Generate new credentials for an ACO client specified by ACO CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {

				// Get ACO by CMS ID (since ResetSecret interface expects a Client ID)
				aco, err := auth.GetACOByCMSID(acoCMSID)
				if err != nil {
					return err
				}

				// Generate new credentials
				creds, err := auth.GetProvider().ResetSecret(aco.ClientID)
				if err != nil {
					return err
				}
				msg := fmt.Sprintf("%s\n%s\n%s", creds.ClientName, creds.ClientID, creds.ClientSecret)
				fmt.Fprintf(app.Writer, "%s\n", msg)
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
			Category: "Cleanup",
			Usage:    "Updates job statuses and moves files to an inaccessible location",
			Action: func(c *cli.Context) error {
				threshold := utils.GetEnvInt("ARCHIVE_THRESHOLD_HR", 24)
				return archiveExpiring(threshold)
			},
		},
		{
			Name:     "cleanup-archive",
			Category: "Cleanup",
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
		{
			Name:     "import-cclf-directory",
			Category: "Data import",
			Usage:    "Import all CCLF files in the directory",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "directory",
					Usage:       "Directory where CCLF Files are located",
					Destination: &filePath,
				},
			},
			Action: func(c *cli.Context) error {
				success, failure, skipped, err := cclf.ImportCCLFDirectory(filePath)
				fmt.Fprintf(app.Writer, "Completed CCLF import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
				return err
			},
		},
		{
			Name:     "delete-dir-contents",
			Category: "Cleanup",
			Usage:    "Delete all of the files in a directory",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "dirToDelete",
					Usage:       "Name of the directory to delete the files from",
					Destination: &dirToDelete,
				},
			},
			Action: func(c *cli.Context) error {
				fi, err := os.Stat(dirToDelete)
				if err != nil {
					return err
				}
				if !fi.IsDir() {
					return fmt.Errorf("unable to delete Directory Contents because %v does not reference a directory", dirToDelete)
				}
				filesDeleted, err := cclf.DeleteDirectoryContents(dirToDelete)
				if filesDeleted > 0 {
					fmt.Fprintf(app.Writer, "Successfully Deleted %v files from %v", filesDeleted, dirToDelete)
				}
				return err
			},
		},
		{
			Name:     "import-synthetic-cclf-package",
			Category: "Data import",
			Usage:    "Import a package of synthetic CCLF files",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "acoSize",
					Usage:       "Set the size of the ACO.  Must be one of 'dev', 'small', 'medium', 'large', or 'extra-large'",
					Destination: &acoSize,
				},
				cli.StringFlag{
					Name:        "environment",
					Usage:       "Which set of files to use.",
					Destination: &environment,
				},
			},
			Action: func(c *cli.Context) error {
				err := cclfUtils.ImportCCLFPackage(acoSize, environment)
				return err
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

func revokeAccessToken(accessToken string) error {
	if accessToken == "" {
		return errors.New("Access token (--access-token) must be provided")
	}

	return auth.GetProvider().RevokeAccessToken(accessToken)
}

func validateAlphaTokenInputs(ttl, acoID string) (int, error) {
	i, err := strconv.Atoi(ttl)
	if err != nil || i <= 0 {
		return i, fmt.Errorf("invalid argument '%v' for --ttl; should be an integer > 0", ttl)
	}

	acoIDFmt := regexp.MustCompile(`T\d{4}`)
	if !acoIDFmt.MatchString(acoID) {
		return i, errors.New("expected CMS ACO ID format for alpha ACOs is 'T' followed by four digits (e.g., 'T1234')")
	}

	return i, nil
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

	var lastJobError error
	for _, j := range jobs {
		t := j.CreatedAt
		elapsed := time.Since(t).Hours()
		if int(elapsed) >= hrThreshold {

			id := int(j.ID)
			jobDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_PAYLOAD_DIR"), id)
			expDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_ARCHIVE_DIR"), id)

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
