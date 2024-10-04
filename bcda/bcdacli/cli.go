package bcdacli

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/ccoveille/go-safecast"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/CMSgov/bcda-app/bcda/alr/csv"
	"github.com/CMSgov/bcda-app/bcda/alr/gen"
	"github.com/CMSgov/bcda-app/bcda/auth"
	authclient "github.com/CMSgov/bcda-app/bcda/auth/client"

	"github.com/CMSgov/bcda-app/bcda/cclf"
	cclfUtils "github.com/CMSgov/bcda-app/bcda/cclf/utils"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/suppression"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/optout"
	"github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// App Name and usage.  Edit them here to prevent breaking tests
const Name = "bcda"
const Usage = "Beneficiary Claims Data API CLI"

var (
	db *sql.DB
	r  models.Repository
)

func GetApp() *cli.App {
	return setUpApp()
}

func setUpApp() *cli.App {
	app := cli.NewApp()
	app.Name = Name
	app.Usage = Usage
	app.Version = constants.Version
	app.Before = func(c *cli.Context) error {
		db = database.Connection
		r = postgres.NewRepository(db)
		return nil
	}
	var hours, err = safecast.ToUint(utils.GetEnvInt("FILE_ARCHIVE_THRESHOLD_HR", 72))
	if err != nil {
		fmt.Println("Error converting FILE_ARCHIVE_THRESHOLD_HR to uint", err)
	}
	var acoName, acoCMSID, acoID, accessToken, acoSize, filePath, fileSource, s3Endpoint, assumeRoleArn, environment, groupID, groupName, ips, fileType, alrFile string
	var thresholdHr int
	var httpPort, httpsPort int
	app.Commands = []cli.Command{
		{
			Name:  "start-api",
			Usage: "Start the API",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "http-port",
					Usage:       "Port to use for http",
					Destination: &httpPort,
				},
				cli.IntFlag{
					Name:        "https-port",
					Usage:       "Port to use for http",
					Destination: &httpsPort,
				},
			},
			Action: func(c *cli.Context) error {
				fmt.Fprintf(app.Writer, "%s\n", "Starting bcda...")

				var httpAddr, httpsAddr string
				if httpPort != 0 {
					httpAddr = fmt.Sprintf(":%d", httpPort)
				} else {
					httpAddr = ":3001"
				}
				if httpsPort != 0 {
					httpsAddr = fmt.Sprintf(":%d", httpsPort)
				} else {
					httpsAddr = ":3000"
				}

				// Accepts and redirects HTTP requests to HTTPS
				srv := &http.Server{
					Handler:           web.NewHTTPRouter(),
					Addr:              httpAddr,
					ReadTimeout:       5 * time.Second,
					WriteTimeout:      5 * time.Second,
					ReadHeaderTimeout: 2 * time.Second,
				}

				go func() { log.API.Fatal(srv.ListenAndServe()) }()

				auth := &http.Server{
					Handler:           web.NewAuthRouter(),
					ReadTimeout:       time.Duration(utils.GetEnvInt("API_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout:      time.Duration(utils.GetEnvInt("API_WRITE_TIMEOUT", 20)) * time.Second,
					IdleTimeout:       time.Duration(utils.GetEnvInt("API_IDLE_TIMEOUT", 120)) * time.Second,
					ReadHeaderTimeout: 2 * time.Second,
				}

				api := &http.Server{
					Handler:           web.NewAPIRouter(),
					ReadTimeout:       time.Duration(utils.GetEnvInt("API_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout:      time.Duration(utils.GetEnvInt("API_WRITE_TIMEOUT", 20)) * time.Second,
					IdleTimeout:       time.Duration(utils.GetEnvInt("API_IDLE_TIMEOUT", 120)) * time.Second,
					ReadHeaderTimeout: 2 * time.Second,
				}

				fileserver := &http.Server{
					Handler:           web.NewDataRouter(),
					ReadTimeout:       time.Duration(utils.GetEnvInt("FILESERVER_READ_TIMEOUT", 10)) * time.Second,
					WriteTimeout:      time.Duration(utils.GetEnvInt("FILESERVER_WRITE_TIMEOUT", 360)) * time.Second,
					IdleTimeout:       time.Duration(utils.GetEnvInt("FILESERVER_IDLE_TIMEOUT", 120)) * time.Second,
					ReadHeaderTimeout: 2 * time.Second,
				}

				smux := servicemux.New(httpsAddr)
				smux.AddServer(fileserver, "/data")
				smux.AddServer(auth, "/auth")
				smux.AddServer(api, "")
				smux.Serve()

				return nil
			},
		},
		{
			Name:     "create-group",
			Category: constants.CliAuthToolsCategory,
			Usage:    "Create a group (SSAS)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "id",
					Usage:       "ID of group",
					Destination: &groupID,
				},
				cli.StringFlag{
					Name:        "name",
					Usage:       "Name of group",
					Destination: &groupName,
				},
				cli.StringFlag{
					Name:        "aco-id",
					Usage:       "CMS ID or UUID of ACO associated with group",
					Destination: &acoID,
				},
			},
			Action: func(c *cli.Context) error {
				ssasID, err := createGroup(groupID, groupName, acoID)
				if err != nil {
					return err
				}
				fmt.Fprint(app.Writer, fmt.Sprint(ssasID))
				return nil
			},
		},
		{
			Name:     "create-aco",
			Category: constants.CliAuthToolsCategory,
			Usage:    "Create an ACO",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "name",
					Usage:       "Name of ACO",
					Destination: &acoName,
				},
				cli.StringFlag{
					Name:        constants.CliCMSIDArg,
					Usage:       constants.CliCMSIDDesc,
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				acoUUID, err := createACO(acoName, acoCMSID)
				if err != nil {
					return err
				}
				fmt.Fprintf(app.Writer, "%s\n", acoUUID)
				return nil
			},
		},
		// FYI, save-public-cred deprecated
		{
			Name:     "revoke-token",
			Category: constants.CliAuthToolsCategory,
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
			Name:     "generate-client-credentials",
			Category: constants.CliAuthToolsCategory,
			Usage:    "Register a system and generate credentials for client specified by ACO CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        constants.CliCMSIDArg,
					Usage:       constants.CliCMSIDDesc,
					Destination: &acoCMSID,
				},
				cli.StringFlag{
					Name:        "ips",
					Usage:       "Comma separated list of IPs associated with the ACO",
					Destination: &ips,
				},
			},
			Action: func(c *cli.Context) error {
				if acoCMSID == "" {
					return errors.New("ACO CMS ID (--cms-id) is required")
				}
				var ipAddr []string
				if len(ips) > 0 {
					ipAddr = strings.Split(ips, ",")
				}
				msg, err := generateClientCredentials(acoCMSID, ipAddr)
				if err != nil {
					return err
				}
				fmt.Fprintln(app.Writer, msg)
				return nil
			},
		},
		{
			Name:     "reset-client-credentials",
			Category: constants.CliAuthToolsCategory,
			Usage:    "Generate a new secret for a client specified by ACO CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        constants.CliCMSIDArg,
					Usage:       constants.CliCMSIDDesc,
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				aco, err := r.GetACOByCMSID(context.Background(), acoCMSID)
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
			Name:     "archive-job-files",
			Category: "Cleanup",
			Usage:    "Update job statuses and move files to an inaccessible location",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "threshold",
					Value:       24,
					Usage:       constants.CliArchDesc,
					EnvVar:      "ARCHIVE_THRESHOLD_HR",
					Destination: &thresholdHr,
				},
			},
			Action: func(c *cli.Context) error {
				cutoff := time.Now().Add(-time.Hour * time.Duration(thresholdHr))
				return archiveExpiring(cutoff)
			},
		},
		{
			Name:     constants.CleanupArchArg,
			Category: "Cleanup",
			Usage:    constants.CliRemoveArchDesc,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "threshold",
					Usage:       constants.CliArchDesc,
					Destination: &thresholdHr,
				},
			},
			Action: func(c *cli.Context) error {
				cutoff := time.Now().Add(-time.Hour * time.Duration(thresholdHr))
				return cleanupJob(cutoff, models.JobStatusArchived, models.JobStatusExpired,
					conf.GetEnv("FHIR_ARCHIVE_DIR"), conf.GetEnv("FHIR_STAGING_DIR"))
			},
		},
		{
			Name:     "cleanup-failed",
			Category: "Cleanup",
			Usage:    constants.CliRemoveArchDesc,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "threshold",
					Usage:       constants.CliArchDesc,
					Destination: &thresholdHr,
				},
			},
			Action: func(c *cli.Context) error {
				cutoff := time.Now().Add(-(time.Hour * time.Duration(thresholdHr)))
				return cleanupJob(cutoff, models.JobStatusFailed, models.JobStatusFailedExpired,
					conf.GetEnv("FHIR_STAGING_DIR"), conf.GetEnv("FHIR_PAYLOAD_DIR"))
			},
		},
		{
			Name:     "cleanup-cancelled",
			Category: "Cleanup",
			Usage:    constants.CliRemoveArchDesc,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "threshold",
					Usage:       constants.CliRemoveArchDesc,
					Destination: &thresholdHr,
				},
			},
			Action: func(c *cli.Context) error {
				cutoff := time.Now().Add(-(time.Hour * time.Duration(thresholdHr)))
				return cleanupJob(cutoff, models.JobStatusCancelled, models.JobStatusCancelledExpired,
					conf.GetEnv("FHIR_STAGING_DIR"), conf.GetEnv("FHIR_PAYLOAD_DIR"))
			},
		},
		{
			Name:     "import-cclf-directory",
			Category: constants.CliDataImpCategory,
			Usage:    "Import all CCLF files from the specified directory",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "directory",
					Usage:       "Directory where CCLF files are located",
					Destination: &filePath,
				},
				cli.StringFlag{
					Name:        "filesource",
					Usage:       "Source of files. Must be one of 'local', 's3'. Defaults to 'local'",
					Destination: &fileSource,
				},
				cli.StringFlag{
					Name:        "s3endpoint",
					Usage:       "Custom S3 endpoint",
					Destination: &s3Endpoint,
				},
				cli.StringFlag{
					Name:        "assume-role-arn",
					Usage:       "Optional IAM role ARN to assume for S3",
					Destination: &assumeRoleArn,
				},
			},
			Action: func(c *cli.Context) error {
				ignoreSignals()
				var file_processor cclf.CclfFileProcessor

				if fileSource == "s3" {
					file_processor = &cclf.S3FileProcessor{
						Handler: optout.S3FileHandler{
							Logger:        log.API,
							Endpoint:      s3Endpoint,
							AssumeRoleArn: assumeRoleArn,
						},
					}
				} else {
					file_processor = &cclf.LocalFileProcessor{
						Handler: optout.LocalFileHandler{
							Logger:                 log.API,
							PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
							FileArchiveThresholdHr: hours,
						},
					}
				}

				importer := cclf.CclfImporter{
					Logger:        log.API,
					FileProcessor: file_processor,
				}

				success, failure, skipped, err := importer.ImportCCLFDirectory(filePath)
				if err != nil {
					log.API.Error("error returned from ImportCCLFDirectory: ", err)
					return err

				}
				if failure > 0 || skipped > 0 {
					log.API.Errorf("Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped, err)
					err = errors.New("Files skipped or failed import. See logs for more details.")
					return err

				}
				log.API.Infof("Completed CCLF import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
				fmt.Fprintf(app.Writer, "Completed CCLF import.  Successfully imported %v files.  Failed to import %v files.  Skipped %v files.  See logs for more details.", success, failure, skipped)
				return err
			},
		},
		{
			Name:     "generate-cclf-runout-files",
			Category: constants.CliDataImpCategory,
			Usage:    "Clone CCLF files and rename them as runout files",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "directory",
					Usage:       "Directory where CCLF files are located",
					Destination: &filePath,
				},
			},
			Action: func(c *cli.Context) error {
				rc, err := cloneCCLFZips(filePath)
				if err != nil {
					fmt.Fprintf(app.Writer, "%s\n", err)
					return err
				}
				fmt.Fprintf(app.Writer, "Completed CCLF runout file generation. Generated %d zip files.", rc)
				return nil
			},
		},
		{
			Name:     "generate-synthetic-alr-data",
			Category: constants.CliDataImpCategory,
			Usage:    "Generate and ingest synthetic ALR data associated with a particular ACO",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        constants.CliCMSIDArg,
					Usage:       constants.CliCMSIDDesc,
					Destination: &acoCMSID,
				},
				cli.StringFlag{
					Name:        "alr-template-file",
					Usage:       "File path for ALR template file",
					Destination: &alrFile,
				},
			},
			Action: func(c *cli.Context) error {
				if alrFile == "" {
					return errors.New("alr template file must be specified")
				}

				file, err := r.GetLatestCCLFFile(context.Background(), acoCMSID, 8, "Completed",
					time.Time{}, time.Time{}, models.FileTypeDefault)
				if err != nil {
					return err
				}
				if file == nil {
					return fmt.Errorf("no CCLF8 file found for CMS ID %s", acoCMSID)
				}

				tempFile, err := os.CreateTemp("", "*")
				if err != nil {
					return err
				}
				in, err := os.Open(filepath.Clean(alrFile))
				if err != nil {
					return err
				}
				defer utils.CloseFileAndLogError(in)

				if _, err := io.Copy(tempFile, in); err != nil {
					return err
				}

				mbiSupplier := func() ([]string, error) {
					return r.GetCCLFBeneficiaryMBIs(context.Background(), file.ID)
				}
				if err := gen.UpdateCSV(tempFile.Name(), mbiSupplier); err != nil {
					return err
				}

				entries, err := csv.ToALR(tempFile.Name())
				if err != nil {
					return err
				}

				alrRepo := postgres.NewAlrRepo(database.Connection)
				alrs := make([]models.Alr, 0, len(entries))
				for idx := range entries {
					alrs = append(alrs, *entries[idx])
				}
				return alrRepo.AddAlr(context.Background(), acoCMSID, time.Now(), alrs)
			},
		},
		{
			Name:     "import-suppression-directory",
			Category: constants.CliDataImpCategory,
			Usage:    "Import all 1-800-MEDICARE suppression data files from the specified directory",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "directory",
					Usage:       "Directory where suppression files are located",
					Destination: &filePath,
				},
				cli.StringFlag{
					Name:        "filesource",
					Usage:       "Source of files. Must be one of 'local', 's3'. Defaults to 'local'",
					Destination: &fileSource,
				},
				cli.StringFlag{
					Name:        "s3endpoint",
					Usage:       "Custom S3 endpoint",
					Destination: &s3Endpoint,
				},
				cli.StringFlag{
					Name:        "assume-role-arn",
					Usage:       "Optional IAM role ARN to assume for S3",
					Destination: &assumeRoleArn,
				},
			},
			Action: func(c *cli.Context) error {
				ignoreSignals()
				db := database.Connection
				r := postgres.NewRepository(db)

				var file_handler optout.OptOutFileHandler

				if fileSource == "s3" {
					file_handler = &optout.S3FileHandler{
						Logger:        log.API,
						Endpoint:      s3Endpoint,
						AssumeRoleArn: assumeRoleArn,
					}
				} else {
					file_handler = &optout.LocalFileHandler{
						Logger:                 log.API,
						PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
						FileArchiveThresholdHr: hours,
					}
				}

				importer := suppression.OptOutImporter{
					FileHandler: file_handler,
					Saver: &suppression.BCDASaver{
						Repo: r,
					},
					Logger:               log.API,
					ImportStatusInterval: utils.GetEnvInt("SUPPRESS_IMPORT_STATUS_RECORDS_INTERVAL", 1000),
				}
				s, f, sk, err := importer.ImportSuppressionDirectory(filePath)
				fmt.Fprintf(app.Writer, "Completed 1-800-MEDICARE suppression data import.\nFiles imported: %v\nFiles failed: %v\nFiles skipped: %v\n", s, f, sk)
				return err
			},
		},
		{
			Name:     "import-synthetic-cclf-package",
			Category: constants.CliDataImpCategory,
			Usage:    "Import a package of synthetic CCLF files",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "acoSize",
					Usage:       "Set the size of the ACO.  Must be one of 'dev', 'dev-auth', 'dev-cec', 'dev-cec-auth', 'dev-ng', 'dev-ng-auth', 'small', 'medium', 'large', or 'extra-large'",
					Destination: &acoSize,
				},
				cli.StringFlag{
					Name:        "environment",
					Usage:       "Which set of files to use.",
					Destination: &environment,
				},
				cli.StringFlag{
					Name:        "fileType",
					Usage:       "Type of CCLF File to generate. Must be one of 'default', 'runout'. Defaults to 'default'",
					Destination: &fileType,
				},
			},
			Action: func(c *cli.Context) error {
				ft := models.FileTypeDefault
				if fileType != "" {
					switch fileType {
					case "runout":
						ft = models.FileTypeRunout
					default:
						return errors.New("Unsupported file type.")
					}
				}
				err := cclfUtils.ImportCCLFPackage(acoSize, environment, ft)
				return err
			},
		},
		{
			Name:     "denylist-aco",
			Category: constants.CliAuthToolsCategory,
			Usage:    "Denylists an ACO by their CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        constants.CliCMSIDArg,
					Usage:       constants.CliCMSIDDesc,
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				td := &models.Termination{
					TerminationDate: time.Now(),
					CutoffDate:      time.Now(),
					DenylistType:    models.Involuntary,
				}
				return setDenylistState(acoCMSID, td)
			},
		},
		{
			Name:     "undenylist-aco",
			Category: constants.CliAuthToolsCategory,
			Usage:    "Undenylists an ACO by their CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        constants.CliCMSIDArg,
					Usage:       constants.CliCMSIDDesc,
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				return setDenylistState(acoCMSID, nil)
			},
		},
	}
	return app
}

func createGroup(id, name, acoID string) (string, error) {
	if id == "" || name == "" || acoID == "" {
		return "", errors.New("ID (--id), name (--name), and ACO ID (--aco-id) are required")
	}

	var (
		aco *models.ACO
		err error
	)

	if match := service.IsSupportedACO(acoID); match {
		aco, err = r.GetACOByCMSID(context.Background(), acoID)
		if err != nil {
			return "", err
		}
	} else if match, err := regexp.MatchString("[0-9a-f]{6}-([0-9a-f]{4}-){3}[0-9a-f]{12}", acoID); err == nil && match {
		aco, err = r.GetACOByUUID(context.Background(), uuid.Parse(acoID))
		if err != nil {
			return "", err
		}
	} else {
		return "", errors.New("ACO ID (--aco-id) must be a supported CMS ID or UUID")
	}

	ssas, err := authclient.NewSSASClient()
	if err != nil {
		return "", err
	}

	b, err := ssas.CreateGroup(id, name, *aco.CMSID)
	if err != nil {
		return "", err
	}

	var g map[string]interface{}
	err = json.Unmarshal(b, &g)
	if err != nil {
		return "", err
	}

	ssasID := g["group_id"].(string)
	if aco.UUID != nil {
		aco.GroupID = ssasID

		err := r.UpdateACO(context.Background(), aco.UUID,
			map[string]interface{}{"group_id": ssasID})
		if err != nil {
			return ssasID, errors.Wrapf(err, "group %s was created, but ACO could not be updated", ssasID)
		}
	}

	return ssasID, nil
}

func createACO(name, cmsID string) (string, error) {
	if name == "" {
		return "", errors.New("ACO name (--name) must be provided")
	}

	var cmsIDPt *string
	if cmsID != "" {
		match := service.IsSupportedACO(cmsID)
		if !match {
			return "", errors.New("ACO CMS ID (--cms-id) is invalid")
		}
		cmsIDPt = &cmsID
	}

	id := uuid.NewRandom()
	aco := models.ACO{Name: name, CMSID: cmsIDPt, UUID: id, ClientID: id.String()}

	err := r.CreateACO(context.Background(), aco)
	if err != nil {
		return "", err
	}

	return aco.UUID.String(), nil
}

func generateClientCredentials(acoCMSID string, ips []string) (string, error) {
	aco, err := r.GetACOByCMSID(context.Background(), acoCMSID)
	if err != nil {
		return "", err
	}

	// The public key is optional for SSAS, and not used by the ACO API
	creds, err := auth.GetProvider().RegisterSystem(aco.UUID.String(), "", aco.GroupID, ips...)
	if err != nil {
		return "", errors.Wrapf(err, "could not register system for %s", acoCMSID)
	}

	msg := fmt.Sprintf("%s\n%s\n%s", creds.ClientName, creds.ClientID, creds.ClientSecret)

	return msg, nil
}

func revokeAccessToken(accessToken string) error {
	if accessToken == "" {
		return errors.New("Access token (--access-token) must be provided")
	}

	return auth.GetProvider().RevokeAccessToken(accessToken)
}

func archiveExpiring(maxDate time.Time) error {
	log.API.Info("Archiving expiring job files...")

	jobs, err := r.GetJobsByUpdateTimeAndStatus(context.Background(),
		time.Time{}, maxDate, models.JobStatusCompleted)
	if err != nil {
		log.API.Error(err)
		return err
	}

	var lastJobError error
	for _, j := range jobs {
		id := j.ID
		jobPayloadDir := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_PAYLOAD_DIR"), id)
		_, err = os.Stat(jobPayloadDir)
		jobPayloadDirExist := err == nil
		jobArchiveDir := fmt.Sprintf("%s/%d", conf.GetEnv("FHIR_ARCHIVE_DIR"), id)

		if jobPayloadDirExist {
			err = os.Rename(jobPayloadDir, jobArchiveDir)
			if err != nil {
				log.API.Error(err)
				lastJobError = err
				continue
			}
		}

		j.Status = models.JobStatusArchived
		err = r.UpdateJob(context.Background(), *j)
		if err != nil {
			log.API.Error(err)
			lastJobError = err
		}
	}

	return lastJobError
}

func cleanupJob(maxDate time.Time, currentStatus, newStatus models.JobStatus, rootDirsToClean ...string) error {
	jobs, err := r.GetJobsByUpdateTimeAndStatus(context.Background(),
		time.Time{}, maxDate, currentStatus)
	if err != nil {
		return err
	}

	if len(jobs) == 0 {
		log.API.Infof("No %s job files to clean", currentStatus)
		return nil
	}

	for _, job := range jobs {
		if err := cleanupJobData(job.ID, rootDirsToClean...); err != nil {
			log.API.Errorf("Unable to cleanup directories %s", err)
			continue
		}

		job.Status = newStatus
		err = r.UpdateJob(context.Background(), *job)
		if err != nil {
			log.API.Errorf("Failed to update job status to %s %s", newStatus, err)
			continue
		}

		log.API.WithFields(logrus.Fields{
			"job_began":     job.CreatedAt,
			"files_removed": time.Now(),
			"job_id":        job.ID,
		}).Infof("Files cleaned from %s and job status set to %s", rootDirsToClean, newStatus)
	}

	return nil
}

func cleanupJobData(jobID uint, rootDirs ...string) error {
	for _, rootDirToClean := range rootDirs {
		dir := filepath.Join(rootDirToClean, strconv.FormatUint(uint64(jobID), 10))
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to remove %s because %s", dir, err)
		}
	}

	return nil
}

func setDenylistState(cmsID string, td *models.Termination) error {
	aco, err := r.GetACOByCMSID(context.Background(), cmsID)
	if err != nil {
		return err
	}
	return r.UpdateACO(context.Background(), aco.UUID,
		map[string]interface{}{"termination_details": td})
}

// CCLF file name pattern and regex
const cclfPattern = `((?:T|P).*\.ZC[A-B0-9]*)Y(\d{2}\.D\d{6}\.T\d{7})`

var cclfregex = regexp.MustCompile(cclfPattern)

func renameCCLF(name string) string {
	return cclfregex.ReplaceAllString(name, "${1}R${2}")
}

func cloneCCLFZips(path string) (int, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	rcount := 0 // Track the number of runout files that are created
	// Iterate through all cclf zip files in provided directory
	for _, f := range files {
		// Make sure to not clone non CCLF files in case the wrong directory is given
		if !cclfregex.MatchString(f.Name()) {
			log.API.Infof("Skipping file `%s`, does not match expected name... ", f.Name())
			continue
		}
		fn := renameCCLF(f.Name())
		err := cloneCCLFZip(filepath.Join(path, f.Name()), filepath.Join(path, fn))
		if err != nil {
			return rcount, err
		}
		rcount++
		log.API.Infof("Created runout file: %s", fn)
	}
	return rcount, nil
}

func cloneCCLFZip(src, dst string) error {
	// Open source zip file for cloning
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()

	// Create destination runout zip file with proper nomenclature
	newf, err := os.Create(path.Clean(dst))
	if err != nil {
		return err
	}
	defer utils.CloseFileAndLogError(newf)

	zw := zip.NewWriter(newf)
	defer zw.Close()

	// For each CCLF file in src zip, rename and write to dst zip
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			return err
		}
		defer r.Close()

		w, err := zw.Create(renameCCLF(f.Name))
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r) // #nosec G110
		if err != nil {
			return err
		}
	}

	return nil
}

func ignoreSignals() chan os.Signal {
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Println("Ignoring SIGTERM/SIGINT to allow work to finish.")

		for range sigs {
			fmt.Println("SIGTERM/SIGINT signal received; ignoring to finish work...")
		}
	}()

	return sigs
}
