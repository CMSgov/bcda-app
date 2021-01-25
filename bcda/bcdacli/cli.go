package bcdacli

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	authclient "github.com/CMSgov/bcda-app/bcda/auth/client"
	"github.com/CMSgov/bcda-app/bcda/auth/rsautils"
	"github.com/CMSgov/bcda-app/bcda/cclf"
	cclfUtils "github.com/CMSgov/bcda-app/bcda/cclf/testutils"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/servicemux"
	"github.com/CMSgov/bcda-app/bcda/suppression"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/web"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
		db = database.GetDbConnection()
		r = postgres.NewRepository(db)
		return nil
	}
	app.After = func(c *cli.Context) error {
		return db.Close()
	}
	var acoName, acoCMSID, acoID, accessToken, acoSize, filePath, dirToDelete, environment, groupID, groupName, ips, fileType string
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
					Handler:      web.NewHTTPRouter(),
					Addr:         httpAddr,
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
			Category: "Authentication tools",
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
				acoUUID, err := createACO(acoName, acoCMSID)
				if err != nil {
					return err
				}
				fmt.Fprintf(app.Writer, "%s\n", acoUUID)
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

				aco, err := r.GetACOByCMSID(context.Background(), acoCMSID)
				if err != nil {
					fmt.Fprintf(app.Writer, "Unable to find ACO %s: %s\n", acoCMSID, err.Error())
					return err
				}

				f, err := os.Open(filepath.Clean(filePath))
				if err != nil {
					fmt.Fprintf(app.Writer, "Unable to open file %s: %s\n", filePath, err.Error())
					return err
				}

				aco.PublicKey, err = genPublicKey(bufio.NewReader(f))
				if err != nil {
					fmt.Fprintf(app.Writer, "Unable to generate public key for ACO %s: %s\n", acoCMSID, err.Error())
					return err
				}

				err = r.UpdateACO(context.Background(), aco.UUID,
					map[string]interface{}{"public_key": aco.PublicKey})
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
			Name:     "generate-client-credentials",
			Category: "Authentication tools",
			Usage:    "Register a system and generate credentials for client specified by ACO CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
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
			Category: "Authentication tools",
			Usage:    "Generate a new secret for a client specified by ACO CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
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
					Usage:       "How long files should wait in archive before deletion",
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
			Name:     "cleanup-archive",
			Category: "Cleanup",
			Usage:    "Remove job directory and files from archive and update job status to Expired",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "threshold",
					Usage:       "How long files should wait in archive before deletion",
					Destination: &thresholdHr,
				},
			},
			Action: func(c *cli.Context) error {
				cutoff := time.Now().Add(-time.Hour * time.Duration(thresholdHr))
				return cleanupJob(cutoff, models.JobStatusArchived, models.JobStatusExpired,
					os.Getenv("FHIR_ARCHIVE_DIR"))
			},
		},
		{
			Name:     "cleanup-failed",
			Category: "Cleanup",
			Usage:    "Remove job directory and files from archive and update job status to Expired",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:        "threshold",
					Usage:       "How long files should wait in archive before deletion",
					Destination: &thresholdHr,
				},
			},
			Action: func(c *cli.Context) error {
				cutoff := time.Now().Add(-(time.Hour * time.Duration(thresholdHr)))
				return cleanupJob(cutoff, models.JobStatusFailed, models.JobStatusFailedExpired,
					os.Getenv("FHIR_STAGING_DIR"), os.Getenv("FHIR_PAYLOAD_DIR"))
			},
		},
		{
			Name:     "import-cclf-directory",
			Category: "Data import",
			Usage:    "Import all CCLF files from the specified directory",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "directory",
					Usage:       "Directory where CCLF files are located",
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
			Name:     "generate-cclf-runout-files",
			Category: "Data import",
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
			Name:     "import-suppression-directory",
			Category: "Data import",
			Usage:    "Import all 1-800-MEDICARE suppression data files from the specified directory",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "directory",
					Usage:       "Directory where suppression files are located",
					Destination: &filePath,
				},
			},
			Action: func(c *cli.Context) error {
				s, f, sk, err := suppression.ImportSuppressionDirectory(filePath)
				fmt.Fprintf(app.Writer, "Completed 1-800-MEDICARE suppression data import.\nFiles imported: %v\nFiles failed: %v\nFiles skipped: %v\n", s, f, sk)
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
				filesDeleted, err := utils.DeleteDirectoryContents(dirToDelete)
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
			Name:     "blacklist-aco",
			Category: "Authentication tools",
			Usage:    "Blacklists an ACO by their CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				return setBlacklistState(acoCMSID, true)
			},
		},
		{
			Name:     "unblacklist-aco",
			Category: "Authentication tools",
			Usage:    "Unblacklists an ACO by their CMS ID",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "cms-id",
					Usage:       "CMS ID of ACO",
					Destination: &acoCMSID,
				},
			},
			Action: func(c *cli.Context) error {
				return setBlacklistState(acoCMSID, false)
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

	if match := models.IsSupportedACO(acoID); match {
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
		match := models.IsSupportedACO(cmsID)
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
	log.Info("Archiving expiring job files...")

	jobs, err := r.GetJobsByUpdateTimeAndStatus(context.Background(),
		time.Time{}, maxDate, models.JobStatusCompleted)
	if err != nil {
		log.Error(err)
		return err
	}

	var lastJobError error
	for _, j := range jobs {
		id := j.ID
		jobPayloadDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_PAYLOAD_DIR"), id)
		_, err = os.Stat(jobPayloadDir)
		jobPayloadDirExist := err == nil
		jobArchiveDir := fmt.Sprintf("%s/%d", os.Getenv("FHIR_ARCHIVE_DIR"), id)

		if jobPayloadDirExist {
			err = os.Rename(jobPayloadDir, jobArchiveDir)
			if err != nil {
				log.Error(err)
				lastJobError = err
				continue
			}
		}

		j.Status = models.JobStatusArchived
		err = r.UpdateJob(context.Background(), *j)
		if err != nil {
			log.Error(err)
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
		log.Infof("No %s job files to clean", currentStatus)
		return nil
	}

	for _, job := range jobs {
		if err := cleanupJobData(job.ID, rootDirsToClean...); err != nil {
			log.Errorf("Unable to cleanup directories %s", err)
			continue
		}

		job.Status = newStatus
		err = r.UpdateJob(context.Background(), *job)
		if err != nil {
			log.Errorf("Failed to update job status to %s %s", newStatus, err)
			continue
		}

		log.WithFields(log.Fields{
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

func setBlacklistState(cmsID string, blacklistState bool) error {
	ctx := context.Background()
	aco, err := r.GetACOByCMSID(ctx, cmsID)
	if err != nil {
		return err
	}
	return r.UpdateACO(context.Background(), aco.UUID,
		map[string]interface{}{"blacklisted": blacklistState})
}

// CCLF file name pattern and regex
const cclfPattern = `((?:T|P).*\.ZC[A-B0-9]*)Y(\d{2}\.D\d{6}\.T\d{7})`

var cclfregex = regexp.MustCompile(cclfPattern)

func renameCCLF(name string) string {
	return cclfregex.ReplaceAllString(name, "${1}R${2}")
}

func cloneCCLFZips(path string) (int, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return 0, err
	}

	rcount := 0 // Track the number of runout files that are created
	// Iterate through all cclf zip files in provided directory
	for _, f := range files {
		// Make sure to not clone non CCLF files in case the wrong directory is given
		if !cclfregex.MatchString(f.Name()) {
			log.Infof("Skipping file `%s`, does not match expected name... ", f.Name())
			continue
		}
		fn := renameCCLF(f.Name())
		err := cloneCCLFZip(filepath.Join(path, f.Name()), filepath.Join(path, fn))
		if err != nil {
			return rcount, err
		}
		rcount++
		log.Infof("Created runout file: %s", fn)
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
	newf, err := os.Create(dst)
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

func genPublicKey(publicKey io.Reader) (string, error) {
	k, err := ioutil.ReadAll(publicKey)
	if err != nil {
		return "", errors.Wrap(err, "cannot read public key")
	}

	key, err := rsautils.ReadPublicKey(string(k))
	if err != nil || key == nil {
		return "", errors.Wrap(err, "invalid public key")
	}
	return string(k), nil
}
