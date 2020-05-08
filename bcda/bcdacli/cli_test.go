package bcdacli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

var origDate string

const (
	invlaidFlag                    = "--abcd"
	invalidFlagErrorMessage        = "flag provided but not defined: -abcd"
	incorrectFlagErrorMessage      = "Incorrect Usage: flag provided but not defined"
	cmsIDFlag                      = "--cms-id"
	eobURL                         = "/api/v1/ExplanationOfBenefit/$export"
	fhirArchiveDirKey              = "FHIR_ARCHIVE_DIR"
	bcdaWorkerTestArchiveDir       = "../bcdaworker/data/test/archive"
	idQuery                        = "id = ?"
	archivedStatus                 = "Archived"
	nameFlag                       = "--name"
	importSuppressionDirectoryArg  = "import-suppression-directory"
	completedMedicareImportMessage = "Completed 1-800-MEDICARE suppression data import."
)

type CLITestSuite struct {
	suite.Suite
	testApp       *cli.App
	expectedSizes map[string]int
}

func (s *CLITestSuite) SetupSuite() {
	s.expectedSizes = map[string]int{
		"dev":    50,
		"small":  10,
		"medium": 25,
		"large":  100,
	}
	testUtils.SetUnitTestKeysForAuth()
	auth.InitAlphaBackend() // should be a provider thing ... inside GetProvider()?
	cclfRefDateKey := "CCLF_REF_DATE"
	origDate = os.Getenv(cclfRefDateKey)
	os.Setenv(cclfRefDateKey, "181125")
}

func (s *CLITestSuite) SetupTest() {
	s.testApp = GetApp()
	autoMigrate()
}

func (s *CLITestSuite) TearDownTest() {
	testUtils.PrintSeparator()
}

func (s *CLITestSuite) TearDownSuite() {
	os.Setenv("CCLF_REF_DATE", origDate)
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TestGetEnvInt() {
	const DEFAULT_VALUE = 200
	os.Setenv("TEST_ENV_STRING", "blah")
	os.Setenv("TEST_ENV_INT", "232")

	assert.Equal(s.T(), 232, utils.GetEnvInt("TEST_ENV_INT", DEFAULT_VALUE))
	assert.Equal(s.T(), DEFAULT_VALUE, utils.GetEnvInt("TEST_ENV_STRING", DEFAULT_VALUE))
	assert.Equal(s.T(), DEFAULT_VALUE, utils.GetEnvInt("FAKE_ENV", DEFAULT_VALUE))
}

func (s *CLITestSuite) TestSetup() {
	assert.Equal(s.T(), 1, 1)
	app := setUpApp()
	assert.Equal(s.T(), app.Name, Name)
	assert.Equal(s.T(), app.Usage, Usage)
}

func (s *CLITestSuite) TestAutoMigrate() {
	// Plenty of other tests will rely on this happening
	// Other tests run these lines so as long as this doesn't error it sb fine
	args := []string{"bcda", "sql-migrate"}
	err := s.testApp.Run(args)
	assert.Nil(s.T(), err)
}

func (s *CLITestSuite) TestSavePublicKeyCLI() {
	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf
	assert := assert.New(s.T())

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	cmsID := "A9901"
	savePublicKeyArg := "save-public-key"
	keyFileFlag := "--key-file"
	_, err := models.CreateACO("Public Key Test ACO", &cmsID)
	assert.Nil(err)
	aco, err := auth.GetACOByCMSID(cmsID)
	assert.Nil(err)
	defer db.Delete(&aco)

	// Unexpected flag
	args := []string{"bcda", savePublicKeyArg, invlaidFlag, "efg"}
	err = s.testApp.Run(args)
	assert.Equal(invalidFlagErrorMessage, err.Error())
	assert.Contains(buf.String(), incorrectFlagErrorMessage)
	buf.Reset()

	// Unspecified ACO
	publicKeyFile := "../../shared_files/ATO_public.pem"
	args = []string{"bcda", savePublicKeyArg, keyFileFlag, publicKeyFile}
	err = s.testApp.Run(args)
	assert.Equal("cms-id is required", err.Error())
	assert.Contains(buf.String(), "")

	// Unspecified File
	args = []string{"bcda", savePublicKeyArg, cmsIDFlag, "A9901"}
	err = s.testApp.Run(args)
	assert.Equal("key-file is required", err.Error())
	assert.Contains(buf.String(), "")

	// Non-existent ACO
	args = []string{"bcda", savePublicKeyArg, cmsIDFlag, "ABCDE", keyFileFlag, publicKeyFile}
	err = s.testApp.Run(args)
	assert.EqualError(err, "no ACO record found for ABCDE")
	assert.Contains(buf.String(), "Unable to find ACO")

	// Missing file
	args = []string{"bcda", savePublicKeyArg, cmsIDFlag, "A9901", keyFileFlag, "FILE_DOES_NOT_EXIST"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "open FILE_DOES_NOT_EXIST: no such file or directory")
	assert.Contains(buf.String(), "Unable to open file")

	// Invalid key
	args = []string{"bcda", savePublicKeyArg, cmsIDFlag, "A9901", keyFileFlag, "../../shared_files/ATO_private.pem"}
	err = s.testApp.Run(args)
	assert.Contains(err.Error(), fmt.Sprintf("invalid public key for ACO %s: unable to parse public key: asn1: structure error: tags don't match", aco.UUID))
	assert.Contains(buf.String(), "Unable to save public key for ACO")

	// Success
	args = []string{"bcda", savePublicKeyArg, cmsIDFlag, "A9901", keyFileFlag, publicKeyFile}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Public key saved for ACO")
}

func (s *CLITestSuite) TestGenerateClientCredentials() {
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf
	assert := assert.New(s.T())

	args := []string{"bcda", "generate-client-credentials", cmsIDFlag, "A9994"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(regexp.MustCompile(".+\n.+\n.+"), buf.String())
}

func (s *CLITestSuite) TestGenerateClientCredentials_InvalidID() {
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf
	generateClientCredentialsArg := "generate-client-credentials"
	assert := assert.New(s.T())

	args := []string{"bcda", generateClientCredentialsArg, cmsIDFlag, "9994"}
	err := s.testApp.Run(args)
	assert.EqualError(err, "no ACO record found for 9994")
	assert.Empty(buf)
	buf.Reset()

	args = []string{"bcda", generateClientCredentialsArg, cmsIDFlag, "A6543"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "no ACO record found for A6543")
	assert.Empty(buf)
}

func (s *CLITestSuite) TestResetSecretCLI() {

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf
	assert := assert.New(s.T())

	outputPattern := regexp.MustCompile(`.+\n(.+)\n.+`)
	resetClientCredentialsArg := "reset-client-credentials"

	// execute positive scenarios via CLI
	args := []string{"bcda", resetClientCredentialsArg, cmsIDFlag, "A9994"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())
	buf.Reset()

	// Execute CLI with invalid ACO CMS ID
	args = []string{"bcda", resetClientCredentialsArg, cmsIDFlag, "BLAH"}
	err = s.testApp.Run(args)
	assert.Equal("no ACO record found for BLAH", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Execute CLI with invalid inputs
	args = []string{"bcda", resetClientCredentialsArg, invlaidFlag, "efg"}
	err = s.testApp.Run(args)
	assert.Equal(invalidFlagErrorMessage, err.Error())
	assert.Contains(buf.String(), incorrectFlagErrorMessage)

}

func (s *CLITestSuite) TestCreateAlphaTokenCLI() {
	// Due to the way the resulting token is returned to the user, not all scenarios can be executed via CLI

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	outputPattern := regexp.MustCompile(`.+\n(.+)\n.+`)
	createAlphaTokenArg := "create-alpha-token"

	// execute positive scenarios via CLI
	args := []string{"bcda", createAlphaTokenArg, "--ttl", "720", cmsIDFlag, "T0001"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())

	buf.Reset()

	// ttl is optional when using the CLI
	args = []string{"bcda", createAlphaTokenArg, cmsIDFlag, "T0002"}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())
	matches := outputPattern.FindSubmatch(buf.Bytes())
	clientID := string(matches[1])
	assert.NotEmpty(clientID)
	aco, err := auth.GetACOByClientID(clientID)
	assert.Nil(err)
	assert.NotEmpty(aco.AlphaSecret)
	buf.Reset()

	args = []string{"bcda", createAlphaTokenArg, cmsIDFlag, "T0003"}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())
	buf.Reset()

	// Execute CLI with invalid inputs
	args = []string{"bcda", createAlphaTokenArg}
	err = s.testApp.Run(args)
	assert.Equal("expected CMS ACO ID format for alpha ACOs is 'T' followed by four digits (e.g., 'T1234')", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	args = []string{"bcda", createAlphaTokenArg, "--ttl", "ABCD", cmsIDFlag, "T0001"}
	err = s.testApp.Run(args)
	assert.Equal("invalid argument 'ABCD' for --ttl; should be an integer > 0", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	args = []string{"bcda", createAlphaTokenArg, "--ttl", "720", cmsIDFlag, "ABCD"}
	err = s.testApp.Run(args)
	assert.Equal("expected CMS ACO ID format for alpha ACOs is 'T' followed by four digits (e.g., 'T1234')", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	args = []string{"bcda", createAlphaTokenArg, invlaidFlag, "efg"}
	err = s.testApp.Run(args)
	assert.Equal(invalidFlagErrorMessage, err.Error())
	assert.Contains(buf.String(), incorrectFlagErrorMessage)
}

func (s *CLITestSuite) TestArchiveExpiring() {

	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	assert := assert.New(s.T())

	// condition: no jobs exist
	args := []string{"bcda", "archive-job-files"}
	err := s.testApp.Run(args)
	assert.Nil(err)

	// save a job to our db
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: eobURL,
		Status:     "Completed",
	}
	db.Save(&j)
	assert.NotNil(j.ID)

	fhirPayloadDirKey := "FHIR_PAYLOAD_DIR"
	os.Setenv(fhirPayloadDirKey, "../bcdaworker/data/test")
	os.Setenv(fhirArchiveDirKey, bcdaWorkerTestArchiveDir)
	id := int(j.ID)
	assert.NotNil(id)

	path := fmt.Sprintf("%s/%d/", os.Getenv(fhirPayloadDirKey), id)
	newpath := os.Getenv(fhirArchiveDirKey)

	if _, err = os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	if _, err = os.Stat(newpath); os.IsNotExist(err) {
		err = os.MkdirAll(newpath, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	f, err := os.Create(fmt.Sprintf("%s/fake.ndjson", path))
	if err != nil {
		s.T().Error(err)
	}
	defer f.Close()

	// execute the test case from CLI
	os.Setenv("ARCHIVE_THRESHOLD_HR", "0")
	args = []string{"bcda", "archive-job-files"}
	err = s.testApp.Run(args)
	assert.Nil(err)

	// check that the file has moved to the archive location
	expPath := fmt.Sprintf("%s/%d/fake.ndjson", os.Getenv(fhirArchiveDirKey), id)
	_, err = ioutil.ReadFile(expPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(expPath, "File not Found")

	var testjob models.Job
	db.First(&testjob, idQuery, j.ID)

	// check the status of the job
	assert.Equal(archivedStatus, testjob.Status)

	// clean up
	os.RemoveAll(os.Getenv(fhirArchiveDirKey))
}

func (s *CLITestSuite) TestArchiveExpiringWithThreshold() {

	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	completedStatus := "Completed"
	fhirPayloadDirKey := "FHIR_PAYLOAD_DIR"

	// save a job to our db
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: eobURL,
		Status:     completedStatus,
	}
	db.Save(&j)
	assert.NotNil(s.T(), j.ID)

	os.Setenv(fhirPayloadDirKey, "../bcdaworker/data/test")
	os.Setenv(fhirArchiveDirKey, bcdaWorkerTestArchiveDir)
	id := int(j.ID)
	assert.NotNil(s.T(), id)

	path := fmt.Sprintf("%s/%d/", os.Getenv(fhirPayloadDirKey), id)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			s.T().Error(err)
		}
	}

	f, err := os.Create(fmt.Sprintf("%s/fake.ndjson", path))
	if err != nil {
		s.T().Error(err)
	}
	defer f.Close()

	err = archiveExpiring(1)
	if err != nil {
		s.T().Error(err)
	}

	// check that the file has not moved to the archive location
	dataPath := fmt.Sprintf("%s/%d/fake.ndjson", os.Getenv(fhirPayloadDirKey), id)
	_, err = ioutil.ReadFile(dataPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(s.T(), dataPath, "File not Found")

	var testjob models.Job
	db.First(&testjob, idQuery, j.ID)

	// check the status of the job
	assert.Equal(s.T(), completedStatus, testjob.Status)

	// clean up
	os.Remove(dataPath)
}

func setupArchivedJob(s *CLITestSuite, email string, modified time.Time) int {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	dateFormat := "2006-01-02 15:04:05"
	acoUUID, err := createACO("ACO "+email, "")
	assert.Nil(s.T(), err)

	// save a job to our db
	j := models.Job{
		ACOID:      uuid.Parse(acoUUID),
		RequestURL: eobURL,
		Status:     archivedStatus,
	}
	db.Save(&j)
	db.Exec("UPDATE jobs SET updated_at=? WHERE id = ?", modified.Format(dateFormat), j.ID)
	db.First(&j, idQuery, j.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), j.ID)
	// compare times using formatted strings to avoid differences (like nano seconds) that we don't care about
	assert.Equal(s.T(), modified.Format(dateFormat), j.UpdatedAt.Format(dateFormat), "UpdatedAt should match %v", modified)

	return int(j.ID)
}

func setupJobArchiveFile(s *CLITestSuite, email string, modified time.Time, accessed time.Time) (int, *os.File) {
	// directory structure is FHIR_ARCHIVE_DIR/<JobId>/<datafile>.ndjson
	// for reference, see main.archiveExpiring() and its companion tests above
	jobId := setupArchivedJob(s, email, modified)
	path := fmt.Sprintf("%s/%d", os.Getenv(fhirArchiveDirKey), jobId)

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		s.T().Error(err)
	}
	jobFile, err := os.Create(fmt.Sprintf("%s/%s", path, "fake.ndjson"))
	if err != nil {
		s.T().Error(err)
	}
	defer jobFile.Close()

	if err := os.Chtimes(jobFile.Name(), accessed, modified); err != nil {
		s.T().Error(err)
	}
	return jobId, jobFile
}

func (s *CLITestSuite) TestCleanArchive() {
	// init
	const Threshold = 30
	thresholdArg := "--threshold"
	cleanupArchiveArg := "cleanup-archive"
	now := time.Now()

	assert := assert.New(s.T())

	// condition: FHIR_ARCHIVE_DIR doesn't exist
	os.Unsetenv(fhirArchiveDirKey)
	args := []string{"bcda", cleanupArchiveArg, thresholdArg, strconv.Itoa(Threshold)}
	err := s.testApp.Run(args)
	assert.Nil(err)
	os.Setenv(fhirArchiveDirKey, bcdaWorkerTestArchiveDir)

	// condition: no jobs exist
	args = []string{"bcda", cleanupArchiveArg, thresholdArg, strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)

	// create a file that was last modified before the Threshold, but accessed after it
	modified := now.Add(-(time.Hour * (Threshold + 1)))
	accessed := now.Add(-(time.Hour * (Threshold - 1)))
	beforeJobID, before := setupJobArchiveFile(s, "before@test.com", modified, accessed)
	defer before.Close()

	// create a file that is clearly after the threshold (unless the threshold is 0)
	afterJobID, after := setupJobArchiveFile(s, "after@test.com", now, now)
	defer after.Close()

	// condition: bad threshold value
	args = []string{"bcda", cleanupArchiveArg, thresholdArg, "abcde"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "strconv.Atoi: parsing \"abcde\": invalid syntax")

	// condition: before < Threshold < after <= now
	// a file created before the Threshold should be deleted; one created after should not
	// we use last modified as a proxy for created, because these files should not be changed after creation
	args = []string{"bcda", cleanupArchiveArg, thresholdArg, strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)

	_, err = os.Stat(before.Name())

	if err == nil {
		assert.Fail("%s was not removed; it should have been", before.Name())
	} else {
		assert.True(os.IsNotExist(err), "%s should have been removed", before.Name())
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var beforeJob models.Job
	db.First(&beforeJob, idQuery, beforeJobID)
	assert.Equal("Expired", beforeJob.Status)

	assert.FileExists(after.Name(), "%s not found; it should have been", after.Name())

	var afterJob models.Job
	db.First(&afterJob, idQuery, afterJobID)
	assert.Equal(archivedStatus, afterJob.Status)

	// I think this is an application directory and should always exist, but that doesn't seem to be the norm
	os.RemoveAll(os.Getenv(fhirArchiveDirKey))
}

func (s *CLITestSuite) TestRevokeToken() {
	originalAuthProvider := auth.GetProviderName()
	defer auth.SetProvider(originalAuthProvider)
	auth.SetProvider("alpha")
	// init

	assert := assert.New(s.T())

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	// Negative case - attempt to revoke a token passing in a blank token string
	args := []string{"bcda", "revoke-token", "--access-token", ""}
	err := s.testApp.Run(args)
	assert.Equal("Access token (--access-token) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Expect (for the moment) that alpha auth does not implement
	args = []string{"bcda", "revoke-token", "--access-token", "this-token-value-is-immaterial"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "RevokeAccessToken is not implemented for alpha auth")
	assert.Equal(0, buf.Len())
	buf.Reset()
}

func (s *CLITestSuite) TestStartAPI() {
	// Negative case
	queueDatabaseURLKey := "QUEUE_DATABASE_URL"
	originalQueueDBURL := os.Getenv(queueDatabaseURLKey)
	os.Setenv(queueDatabaseURLKey, "http://bad url.com/")
	args := []string{"bcda", "start-api"}
	err := s.testApp.Run(args)
	assert.NotNil(s.T(), err)
	os.Setenv(queueDatabaseURLKey, originalQueueDBURL)

	// We cannot test the positive case because we don't want to start the HTTP Server in unit test environment
}

func (s *CLITestSuite) TestCreateGroup() {
	ssasURLKey := "SSAS_URL"
	ssasUseTLSKey := "SSAS_USE_TLS"

	router := chi.NewRouter()
	router.Post("/group", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{ "ID": 100, "group_id": "test-create-group-id" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	origSSASURL := os.Getenv(ssasURLKey)
	os.Setenv(ssasURLKey, server.URL)
	defer os.Setenv(ssasURLKey, origSSASURL)

	origSSASUseTLS := os.Getenv(ssasUseTLSKey)
	os.Setenv(ssasUseTLSKey, "false")
	defer os.Setenv(ssasUseTLSKey, origSSASUseTLS)

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	id := "unit-test-group-1"
	name := "Unit Test Group 1"
	acoID := "A9995"
	args := []string{"bcda", "create-group", "--id", id, nameFlag, name, "--aco-id", acoID}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Equal("test-create-group-id", buf.String())
}

func (s *CLITestSuite) TestCreateGroup_InvalidACOID() {
	createGroupArg := "create-group"
	acoIDFlag := "--aco-id"

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	// Invalid format
	args := []string{"bcda", createGroupArg, "--id", "invalid-aco-id-group", nameFlag, "Invalid ACO ID Group", acoIDFlag, "1234"}
	err := s.testApp.Run(args)
	assert.EqualError(s.T(), err, "ACO ID (--aco-id) must be a CMS ID (A####) or UUID")
	assert.Empty(s.T(), buf.String())
	buf.Reset()

	// Valid format, but no matching ACO
	aUUID := "4e5519cb-428d-4934-a3f8-6d3efb1277b7"
	args = []string{"bcda", createGroupArg, "--id", "invalid-aco-id-group", nameFlag, "Invalid ACO ID Group", acoIDFlag, aUUID}
	err = s.testApp.Run(args)
	assert.EqualError(s.T(), err, "no ACO record found for "+aUUID)
	assert.Empty(s.T(), buf.String())
}

func (s *CLITestSuite) TestCreateACO() {
	createACOArg := "create-aco"
	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	// Successful ACO creation
	ACOName := "Unit Test ACO 1"
	args := []string{"bcda", createACOArg, nameFlag, ACOName}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID := strings.TrimSpace(buf.String())
	var testACO models.ACO
	db.First(&testACO, "Name=?", ACOName)
	assert.Equal(testACO.UUID.String(), acoUUID)
	buf.Reset()

	ACO2Name := "Unit Test ACO 2"
	aco2ID := "A9999"
	args = []string{"bcda", createACOArg, nameFlag, ACO2Name, cmsIDFlag, aco2ID}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID = strings.TrimSpace(buf.String())
	var testACO2 models.ACO
	db.First(&testACO2, "Name=?", ACO2Name)
	assert.Equal(testACO2.UUID.String(), acoUUID)
	assert.Equal(*testACO2.CMSID, aco2ID)
	buf.Reset()

	// Negative tests

	acoNameMissingErrorMessage := "ACO name (--name) must be provided"

	// No parameters
	args = []string{"bcda", createACOArg}
	err = s.testApp.Run(args)
	assert.Equal(acoNameMissingErrorMessage, err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// No ACO Name
	badACO := ""
	args = []string{"bcda", createACOArg, nameFlag, badACO}
	err = s.testApp.Run(args)
	assert.Equal(acoNameMissingErrorMessage, err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// ACO name without flag
	args = []string{"bcda", createACOArg, ACOName}
	err = s.testApp.Run(args)
	assert.Equal(acoNameMissingErrorMessage, err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Unexpected flag
	args = []string{"bcda", createACOArg, invlaidFlag, "efg"}
	err = s.testApp.Run(args)
	assert.Equal(invalidFlagErrorMessage, err.Error())
	assert.Contains(buf.String(), incorrectFlagErrorMessage)
	buf.Reset()

	// Invalid CMS ID
	args = []string{"bcda", createACOArg, nameFlag, ACOName, cmsIDFlag, "ABCDE"}
	err = s.testApp.Run(args)
	assert.Equal("ACO CMS ID (--cms-id) is invalid", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()
}

func (s *CLITestSuite) TestImportCCLFDirectory() {
	directoryFlag := "--directory"

	assert := assert.New(s.T())

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	var existngCCLFFiles []models.CCLFFile
	db.Where("aco_cms_id = ?", "A0001").Find(&existngCCLFFiles)
	for _, cclfFile := range existngCCLFFiles {
		err := cclfFile.Delete()
		assert.Nil(err)
	}

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	testUtils.SetPendingDeletionDir(s.Suite)

	args := []string{"bcda", "import-cclf-directory", directoryFlag, "../../shared_files/cclf/archives/valid/"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 6 files.")
	assert.Contains(buf.String(), "Failed to import 0 files.")
	assert.Contains(buf.String(), "Skipped 1 files.")

	buf.Reset()

	db.Where("aco_cms_id = ?", "A0001").Find(&existngCCLFFiles)
	for _, cclfFile := range existngCCLFFiles {
		err := cclfFile.Delete()
		assert.Nil(err)
	}

	testUtils.ResetFiles(s.Suite, "../../shared_files/cclf/archives/valid/")

	// dir has 4 files, but 2 will be ignored because of bad file names.
	args = []string{"bcda", "import-cclf-directory", directoryFlag, "../../shared_files/cclf/mixed/with_invalid_filenames/"}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 2 files.")
	assert.Contains(buf.String(), "Skipped 4 files.")
	buf.Reset()

	testUtils.ResetFiles(s.Suite, "../../shared_files/cclf/mixed/with_invalid_filenames/")
}

func (s *CLITestSuite) TestDeleteDirectoryContents() {
	deleteDirContentsArg := "delete-dir-contents"

	assert := assert.New(s.T())
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	dirToDelete := "../../shared_files/doomedDirectory"
	testUtils.MakeDirToDelete(s.Suite, dirToDelete)
	defer os.Remove(dirToDelete)

	args := []string{"bcda", deleteDirContentsArg, "--dirToDelete", dirToDelete}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), fmt.Sprintf("Successfully Deleted 4 files from %v", dirToDelete))
	buf.Reset()

	// File, not a directory
	args = []string{"bcda", deleteDirContentsArg, "--dirToDelete", "../../shared_files/cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "unable to delete Directory Contents because ../../shared_files/cclf/archives/valid/T.BCD.A0001.ZCY18.D181121.T1000000 does not reference a directory")
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

	os.Setenv("TESTDELETEDIRECTORY", "NOT/A/REAL/DIRECTORY")
	args = []string{"bcda", deleteDirContentsArg, "--envvar", "TESTDELETEDIRECTORY"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "flag provided but not defined: -envvar")
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

}

func (s *CLITestSuite) TestImportSuppressionDirectory() {

	assert := assert.New(s.T())

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path := "../../shared_files/synthetic1800MedicareFiles/test/"

	args := []string{"bcda", importSuppressionDirectoryArg, "--directory", path}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), completedMedicareImportMessage)
	assert.Contains(buf.String(), "Files imported: 2")
	assert.Contains(buf.String(), "Files failed: 0")
	assert.Contains(buf.String(), "Files skipped: 0")

	testUtils.ResetFiles(s.Suite, path)

	fs := []models.SuppressionFile{}
	db.Where("name in (?)", []string{"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", "T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241390"}).Find(&fs)
	assert.Len(fs, 2)
	for _, f := range fs {
		err := f.Delete()
		assert.Nil(err)
	}
}

func (s *CLITestSuite) TestImportSuppressionDirectory_Skipped() {
	assert := assert.New(s.T())

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path := "../../shared_files/suppressionfile_BadFileNames/"

	args := []string{"bcda", importSuppressionDirectoryArg, "--directory", path}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), completedMedicareImportMessage)
	assert.Contains(buf.String(), "Files imported: 0")
	assert.Contains(buf.String(), "Files failed: 0")
	assert.Contains(buf.String(), "Files skipped: 2")

	testUtils.ResetFiles(s.Suite, path)
}

func (s *CLITestSuite) TestImportSuppressionDirectory_Failed() {
	assert := assert.New(s.T())

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path := "../../shared_files/suppressionfile_BadHeader/"

	args := []string{"bcda", importSuppressionDirectoryArg, "--directory", path}
	err := s.testApp.Run(args)
	assert.EqualError(err, "one or more suppression files failed to import correctly")
	assert.Contains(buf.String(), completedMedicareImportMessage)
	assert.Contains(buf.String(), "Files imported: 0")
	assert.Contains(buf.String(), "Files failed: 1")
	assert.Contains(buf.String(), "Files skipped: 0")

	testUtils.ResetFiles(s.Suite, path)
}
