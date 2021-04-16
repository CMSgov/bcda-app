package bcdacli

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/go-chi/chi"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

var origDate string

type CLITestSuite struct {
	suite.Suite
	testApp            *cli.App
	expectedSizes      map[string]int
	pendingDeletionDir string

	testACO models.ACO

	db *sql.DB
}

func (s *CLITestSuite) SetupSuite() {
	s.expectedSizes = map[string]int{
		"dev":    50,
		"small":  10,
		"medium": 25,
		"large":  100,
	}
	origDate = conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181125")

	dir, err := ioutil.TempDir("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)

	s.db = database.Connection

	cmsID := testUtils.RandomHexID()[0:4]
	s.testACO = models.ACO{Name: uuid.New(), UUID: uuid.NewRandom(), ClientID: uuid.New(), CMSID: &cmsID}
	postgrestest.CreateACO(s.T(), s.db, s.testACO)
}

func (s *CLITestSuite) SetupTest() {
	s.testApp = GetApp()
}

func (s *CLITestSuite) TearDownTest() {
	testUtils.PrintSeparator()
}

func (s *CLITestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", origDate)
	os.RemoveAll(s.pendingDeletionDir)
	postgrestest.DeleteACO(s.T(), s.db, s.testACO.UUID)
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TestGetEnvInt() {
	const DEFAULT_VALUE = 200
	conf.SetEnv(s.T(), "TEST_ENV_STRING", "blah")
	conf.SetEnv(s.T(), "TEST_ENV_INT", "232")

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
func (s *CLITestSuite) TestSavePublicKeyCLI() {
	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf
	assert := assert.New(s.T())

	cmsID := "A9901"
	u := uuid.NewRandom()
	aco := models.ACO{Name: "Public Key Test ACO", UUID: u, ClientID: u.String(), CMSID: &cmsID}
	postgrestest.CreateACO(s.T(), s.db, aco)
	defer postgrestest.DeleteACO(s.T(), s.db, aco.UUID)

	// Unexpected flag
	args := []string{"bcda", "save-public-key", "--abcd", "efg"}
	err := s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")
	buf.Reset()

	// Unspecified ACO
	args = []string{"bcda", "save-public-key", "--key-file", "../../shared_files/ATO_public.pem"}
	err = s.testApp.Run(args)
	assert.Equal("cms-id is required", err.Error())
	assert.Contains(buf.String(), "")

	// Unspecified File
	args = []string{"bcda", "save-public-key", "--cms-id", "A9901"}
	err = s.testApp.Run(args)
	assert.Equal("key-file is required", err.Error())
	assert.Contains(buf.String(), "")

	// Non-existent ACO
	args = []string{"bcda", "save-public-key", "--cms-id", "ABCDE", "--key-file", "../../shared_files/ATO_public.pem"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "no ACO record found for ABCDE")
	assert.Contains(buf.String(), "Unable to find ACO")

	// Missing file
	args = []string{"bcda", "save-public-key", "--cms-id", "A9901", "--key-file", "FILE_DOES_NOT_EXIST"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "open FILE_DOES_NOT_EXIST: no such file or directory")
	assert.Contains(buf.String(), "Unable to open file")

	// Invalid key
	args = []string{"bcda", "save-public-key", "--cms-id", "A9901", "--key-file", "../../shared_files/ATO_private.pem"}
	err = s.testApp.Run(args)
	assert.Contains(err.Error(), "invalid public key: unable to parse public key: asn1: structure error: tags don't match")
	assert.Contains(buf.String(), "Unable to generate public key for ACO")

	// Success
	args = []string{"bcda", "save-public-key", "--cms-id", "A9901", "--key-file", "../../shared_files/ATO_public.pem"}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Public key saved for ACO")
}

func (s *CLITestSuite) TestGenerateClientCredentials() {
	for idx, ips := range [][]string{nil, {testUtils.GetRandomIPV4Address(s.T()), testUtils.GetRandomIPV4Address(s.T())},
		{testUtils.GetRandomIPV4Address(s.T())}, nil} {
		s.T().Run(strconv.Itoa(idx), func(t *testing.T) {
			mockArgs := []interface{}{s.testACO.UUID.String(), "", s.testACO.GroupID}
			// ips argument is a variadic argument so we need to ensure that the list is expanded
			// when supplying the ips argument to the mock
			for _, ip := range ips {
				mockArgs = append(mockArgs, ip)
			}
			m := &auth.MockProvider{}
			m.On("RegisterSystem", mockArgs...).Return(
				auth.Credentials{ClientName: *s.testACO.CMSID, ClientID: s.testACO.UUID.String(), ClientSecret: uuid.New()},
				nil)
			auth.SetMockProvider(t, m)

			buf := new(bytes.Buffer)
			s.testApp.Writer = buf

			args := []string{"bcda", "generate-client-credentials", "--cms-id", *s.testACO.CMSID, "--ips", strings.Join(ips, ",")}
			err := s.testApp.Run(args)
			assert.Nil(t, err)
			assert.Regexp(t, regexp.MustCompile(".+\n.+\n.+"), buf.String())
			m.AssertExpectations(t)
		})
	}
}

func (s *CLITestSuite) TestGenerateClientCredentials_InvalidID() {
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf
	assert := assert.New(s.T())

	args := []string{"bcda", "generate-client-credentials", "--cms-id", "9994"}
	err := s.testApp.Run(args)
	assert.EqualError(err, "no ACO record found for 9994")
	assert.Empty(buf)
	buf.Reset()

	args = []string{"bcda", "generate-client-credentials", "--cms-id", "A6543"}
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

	mock := &auth.MockProvider{}
	mock.On("ResetSecret", s.testACO.ClientID).Return(
		auth.Credentials{ClientName: *s.testACO.CMSID, ClientID: s.testACO.ClientID,
			ClientSecret: uuid.New()},
		nil)
	auth.SetMockProvider(s.T(), mock)

	// execute positive scenarios via CLI
	args := []string{"bcda", "reset-client-credentials", "--cms-id", *s.testACO.CMSID}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())
	buf.Reset()

	// Execute CLI with invalid ACO CMS ID
	args = []string{"bcda", "reset-client-credentials", "--cms-id", "BLAH"}
	err = s.testApp.Run(args)
	assert.Equal("no ACO record found for BLAH", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Execute CLI with invalid inputs
	args = []string{"bcda", "reset-client-credentials", "--abcd", "efg"}
	err = s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")

	mock.AssertExpectations(s.T())
}

func (s *CLITestSuite) TestRevokeToken() {
	assert := assert.New(s.T())

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	accessToken := uuid.New()
	mock := &auth.MockProvider{}
	mock.On("RevokeAccessToken", accessToken).Return(nil)
	auth.SetMockProvider(s.T(), mock)
	assert.NoError(s.testApp.Run([]string{"bcda", "revoke-token", "--access-token", accessToken}))
	buf.Reset()

	// Negative case - attempt to revoke a token passing in a blank token string
	args := []string{"bcda", "revoke-token", "--access-token", ""}
	err := s.testApp.Run(args)
	assert.Equal("Access token (--access-token) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	mock.AssertExpectations(s.T())
}

func (s *CLITestSuite) TestArchiveExpiring() {
	assert := assert.New(s.T())

	// condition: no jobs exist
	args := []string{"bcda", "archive-job-files"}
	err := s.testApp.Run(args)
	assert.Nil(err)

	// timestamp to ensure that the job gets archived (older than the default 24h window)
	t := time.Now().Add(-48 * time.Hour)
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusCompleted,
		CreatedAt:  t,
		UpdatedAt:  t,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", "../bcdaworker/data/test")
	conf.SetEnv(s.T(), "FHIR_ARCHIVE_DIR", "../bcdaworker/data/test/archive")

	path := fmt.Sprintf("%s/%d/", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
	newpath := conf.GetEnv("FHIR_ARCHIVE_DIR")

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

	// condition: normal execution
	// execute the test case from CLI
	args = []string{"bcda", "archive-job-files"}
	err = s.testApp.Run(args)
	assert.Nil(err)

	// check that the file has moved to the archive location
	expPath := fmt.Sprintf("%s/%d/fake.ndjson", conf.GetEnv("FHIR_ARCHIVE_DIR"), j.ID)
	_, err = ioutil.ReadFile(expPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(expPath, "File not Found")

	testJob := postgrestest.GetJobByID(s.T(), s.db, j.ID)

	// check the status of the job
	assert.Equal(models.JobStatusArchived, testJob.Status)

	// clean up
	os.RemoveAll(conf.GetEnv("FHIR_ARCHIVE_DIR"))
}

func (s *CLITestSuite) TestArchiveExpiringWithoutPayloadDir() {
	assert := assert.New(s.T())

	// condition: no jobs exist
	args := []string{"bcda", "archive-job-files"}
	err := s.testApp.Run(args)
	assert.Nil(err)

	// timestamp to ensure that the job gets archived (older than the default 24h window)
	t := time.Now().Add(-48 * time.Hour)
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusCompleted,
		CreatedAt:  t,
		UpdatedAt:  t,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	// condition: normal execution
	// execute the test case from CLI
	args = []string{"bcda", "archive-job-files"}
	err = s.testApp.Run(args)
	assert.Nil(err)

	testJob := postgrestest.GetJobByID(s.T(), s.db, j.ID)

	// check the status of the job
	assert.Equal(models.JobStatusArchived, testJob.Status)

	// clean up
	os.RemoveAll(conf.GetEnv("FHIR_ARCHIVE_DIR"))
}

func (s *CLITestSuite) TestArchiveExpiringWithThreshold() {
	// save a job to our db
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     models.JobStatusCompleted,
	}
	postgrestest.CreateJobs(s.T(), s.db, &j)

	conf.SetEnv(s.T(), "FHIR_PAYLOAD_DIR", "../bcdaworker/data/test")
	conf.SetEnv(s.T(), "FHIR_ARCHIVE_DIR", "../bcdaworker/data/test/archive")

	path := fmt.Sprintf("%s/%d/", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
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

	// execute the test case from CLI
	args := []string{"bcda", "archive-job-files", "--threshold", "1"}
	err = s.testApp.Run(args)
	assert.Nil(s.T(), err)

	// check that the file has not moved to the archive location
	dataPath := fmt.Sprintf("%s/%d/fake.ndjson", conf.GetEnv("FHIR_PAYLOAD_DIR"), j.ID)
	_, err = ioutil.ReadFile(dataPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(s.T(), dataPath, "File not Found")

	testJob := postgrestest.GetJobByID(s.T(), s.db, j.ID)
	// check the status of the job
	assert.Equal(s.T(), models.JobStatusCompleted, testJob.Status)

	// clean up
	os.Remove(dataPath)
}

func (s *CLITestSuite) TestCleanArchive() {
	// init
	const Threshold = 30
	now := time.Now()

	assert := assert.New(s.T())

	// condition: FHIR_ARCHIVE_DIR doesn't exist
	conf.UnsetEnv(s.T(), "FHIR_ARCHIVE_DIR")
	args := []string{"bcda", "cleanup-archive", "--threshold", strconv.Itoa(Threshold)}
	err := s.testApp.Run(args)
	assert.Nil(err)
	conf.SetEnv(s.T(), "FHIR_ARCHIVE_DIR", "../bcdaworker/data/test/archive")

	// condition: no jobs exist
	args = []string{"bcda", "cleanup-archive", "--threshold", strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)

	// create a file that was last modified before the Threshold, but accessed after it
	modified := now.Add(-(time.Hour * (Threshold + 1)))
	beforeJobID, before := s.setupJobFile(modified, models.JobStatusArchived, conf.GetEnv("FHIR_ARCHIVE_DIR"))
	defer before.Close()

	// create a file that is clearly after the threshold (unless the threshold is 0)
	afterJobID, after := s.setupJobFile(now, models.JobStatusArchived, conf.GetEnv("FHIR_ARCHIVE_DIR"))
	defer after.Close()

	// condition: bad threshold value
	args = []string{"bcda", "cleanup-archive", "--threshold", "abcde"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid value \"abcde\" for flag -threshold: parse error")

	// condition: before < Threshold < after <= now
	// a file created before the Threshold should be deleted; one created after should not
	// we use last modified as a proxy for created, because these files should not be changed after creation
	args = []string{"bcda", "cleanup-archive", "--threshold", strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)

	assertFileNotExists(s.T(), before.Name())

	beforeJob := postgrestest.GetJobByID(s.T(), s.db, beforeJobID)
	assert.Equal(models.JobStatusExpired, beforeJob.Status)

	assert.FileExists(after.Name(), "%s not found; it should have been", after.Name())

	afterJob := postgrestest.GetJobByID(s.T(), s.db, afterJobID)
	assert.Equal(models.JobStatusArchived, afterJob.Status)

	// I think this is an application directory and should always exist, but that doesn't seem to be the norm
	os.RemoveAll(conf.GetEnv("FHIR_ARCHIVE_DIR"))
}

func (s *CLITestSuite) TestCleanupFailed() {
	const threshold = 30
	modified := time.Now().Add(-(time.Hour * (threshold + 1)))
	beforePayloadJobID, beforePayload := s.setupJobFile(modified, models.JobStatusFailed, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	beforeStagingJobID, beforeStaging := s.setupJobFile(modified, models.JobStatusFailed, conf.GetEnv("FHIR_STAGING_DIR"))
	// Job is old enough, but does not match the status
	completedJobID, completed := s.setupJobFile(modified, models.JobStatusCompleted, conf.GetEnv("FHIR_PAYLOAD_DIR"))

	afterPayloadJobID, afterPayload := s.setupJobFile(time.Now(), models.JobStatusFailed, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	afterStagingJobID, afterStaging := s.setupJobFile(time.Now(), models.JobStatusFailed, conf.GetEnv("FHIR_STAGING_DIR"))

	// Check that we can clean up jobs that do not have data
	noDataID, noData := s.setupJobFile(modified, models.JobStatusFailed, conf.GetEnv("FHIR_STAGING_DIR"))
	dir, _ := path.Split(noData.Name())
	os.RemoveAll(dir)
	assertFileNotExists(s.T(), noData.Name())

	shouldExist := []*os.File{afterPayload, afterStaging, completed}
	shouldNotExist := []*os.File{beforePayload, beforeStaging, noData}

	defer func() {
		for _, f := range append(shouldExist, shouldNotExist...) {
			os.Remove(f.Name())
			f.Close()
		}
	}()

	err := s.testApp.Run([]string{"bcda", "cleanup-failed", "--threshold", strconv.Itoa(threshold)})
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), models.JobStatusFailedExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforePayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusFailedExpired,
		postgrestest.GetJobByID(s.T(), s.db, noDataID).Status)
	assert.Equal(s.T(), models.JobStatusFailedExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforeStagingJobID).Status)

	assert.Equal(s.T(), models.JobStatusFailed,
		postgrestest.GetJobByID(s.T(), s.db, afterPayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusFailed,
		postgrestest.GetJobByID(s.T(), s.db, afterStagingJobID).Status)
	assert.Equal(s.T(), models.JobStatusCompleted,
		postgrestest.GetJobByID(s.T(), s.db, completedJobID).Status)

	for _, f := range shouldExist {
		assert.FileExists(s.T(), f.Name())
	}
	for _, f := range shouldNotExist {
		assertFileNotExists(s.T(), f.Name())
	}
}

func (s *CLITestSuite) TestCleanupCancelled() {
	const threshold = 30
	modified := time.Now().Add(-(time.Hour * (threshold + 1)))
	beforePayloadJobID, beforePayload := s.setupJobFile(modified, models.JobStatusCancelled, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	beforeStagingJobID, beforeStaging := s.setupJobFile(modified, models.JobStatusCancelled, conf.GetEnv("FHIR_STAGING_DIR"))
	// Job is old enough, but does not match the status
	completedJobID, completed := s.setupJobFile(modified, models.JobStatusCompleted, conf.GetEnv("FHIR_PAYLOAD_DIR"))

	afterPayloadJobID, afterPayload := s.setupJobFile(time.Now(), models.JobStatusCancelled, conf.GetEnv("FHIR_PAYLOAD_DIR"))
	afterStagingJobID, afterStaging := s.setupJobFile(time.Now(), models.JobStatusCancelled, conf.GetEnv("FHIR_STAGING_DIR"))

	// Check that we can clean up jobs that do not have data
	noDataID, noData := s.setupJobFile(modified, models.JobStatusCancelled, conf.GetEnv("FHIR_STAGING_DIR"))
	dir, _ := path.Split(noData.Name())
	os.RemoveAll(dir)
	assertFileNotExists(s.T(), noData.Name())

	shouldExist := []*os.File{afterPayload, afterStaging, completed}
	shouldNotExist := []*os.File{beforePayload, beforeStaging, noData}

	defer func() {
		for _, f := range append(shouldExist, shouldNotExist...) {
			os.Remove(f.Name())
			f.Close()
		}
	}()

	err := s.testApp.Run([]string{"bcda", "cleanup-cancelled", "--threshold", strconv.Itoa(threshold)})
	assert.NoError(s.T(), err)

	assert.Equal(s.T(), models.JobStatusCancelledExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforePayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusCancelledExpired,
		postgrestest.GetJobByID(s.T(), s.db, noDataID).Status)
	assert.Equal(s.T(), models.JobStatusCancelledExpired,
		postgrestest.GetJobByID(s.T(), s.db, beforeStagingJobID).Status)

	assert.Equal(s.T(), models.JobStatusCancelled,
		postgrestest.GetJobByID(s.T(), s.db, afterPayloadJobID).Status)
	assert.Equal(s.T(), models.JobStatusCancelled,
		postgrestest.GetJobByID(s.T(), s.db, afterStagingJobID).Status)
	assert.Equal(s.T(), models.JobStatusCompleted,
		postgrestest.GetJobByID(s.T(), s.db, completedJobID).Status)

	for _, f := range shouldExist {
		assert.FileExists(s.T(), f.Name())
	}
	for _, f := range shouldNotExist {
		assertFileNotExists(s.T(), f.Name())
	}
}

func (s *CLITestSuite) TestStartAPI() {
	httpsPort, httpPort := strconv.Itoa(getRandomPort(s.T())), strconv.Itoa(getRandomPort(s.T()))
	args := []string{"bcda", "start-api", "--https-port", httpsPort, "--http-port", httpPort}
	go func() {
		if err := s.testApp.Run(args); err != nil {
			s.FailNow(err.Error())
		}
		s.Fail("start-api command should not return")
	}()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-timeout:
			s.FailNow("Failed to get health response in 10 seconds")
		default:
			// Still use http because our testing environment has HTTP_ONLY=true
			resp, err := http.Get(fmt.Sprintf("http://localhost:%s/_health", httpsPort))
			// Allow transient failures
			if err != nil {
				log.Warnf("Error occurred when making request. Retrying. %s", err.Error())
				continue
			}
			s.Equal(http.StatusOK, resp.StatusCode)
			return
		}
	}
}

func (s *CLITestSuite) TestCreateGroup() {
	router := chi.NewRouter()
	router.Post("/group", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{ "ID": 100, "group_id": "test-create-group-id" }`))
		if err != nil {
			log.Fatal(err)
		}
	})
	server := httptest.NewServer(router)

	origSSASURL := conf.GetEnv("SSAS_URL")
	conf.SetEnv(s.T(), "SSAS_URL", server.URL)
	defer conf.SetEnv(s.T(), "SSAS_URL", origSSASURL)

	origSSASUseTLS := conf.GetEnv("SSAS_USE_TLS")
	conf.SetEnv(s.T(), "SSAS_USE_TLS", "false")
	defer conf.SetEnv(s.T(), "SSAS_USE_TLS", origSSASUseTLS)

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	id := "unit-test-group-1"
	name := "Unit Test Group 1"
	acoID := "A9995"
	args := []string{"bcda", "create-group", "--id", id, "--name", name, "--aco-id", acoID}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Equal("test-create-group-id", buf.String())
}

func (s *CLITestSuite) TestCreateGroup_InvalidACOID() {
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	// Invalid format
	args := []string{"bcda", "create-group", "--id", "invalid-aco-id-group", "--name", "Invalid ACO ID Group", "--aco-id", "1234"}
	err := s.testApp.Run(args)
	assert.EqualError(s.T(), err, "ACO ID (--aco-id) must be a supported CMS ID or UUID")
	assert.Empty(s.T(), buf.String())
	buf.Reset()

	// Valid format, but no matching ACO
	aUUID := "4e5519cb-428d-4934-a3f8-6d3efb1277b7"
	args = []string{"bcda", "create-group", "--id", "invalid-aco-id-group", "--name", "Invalid ACO ID Group", "--aco-id", aUUID}
	err = s.testApp.Run(args)
	assert.EqualError(s.T(), err, "no ACO record found for "+aUUID)
	assert.Empty(s.T(), buf.String())
}

func (s *CLITestSuite) TestCreateACO() {
	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	// Successful ACO creation
	ACOName := "Unit Test ACO 1"
	args := []string{"bcda", "create-aco", "--name", ACOName}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID := uuid.Parse(strings.TrimSpace(buf.String()))

	testACO := postgrestest.GetACOByUUID(s.T(), s.db, acoUUID)
	assert.Equal(ACOName, testACO.Name)
	buf.Reset()

	ACO2Name := "Unit Test ACO 2"
	aco2ID := "A9999"
	args = []string{"bcda", "create-aco", "--name", ACO2Name, "--cms-id", aco2ID}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID = uuid.Parse(strings.TrimSpace(buf.String()))

	testACO2 := postgrestest.GetACOByUUID(s.T(), s.db, acoUUID)
	assert.Equal(ACO2Name, testACO2.Name)
	assert.Equal(aco2ID, *testACO2.CMSID)
	buf.Reset()

	// Negative tests

	// No parameters
	args = []string{"bcda", "create-aco"}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// No ACO Name
	badACO := ""
	args = []string{"bcda", "create-aco", "--name", badACO}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// ACO name without flag
	args = []string{"bcda", "create-aco", ACOName}
	err = s.testApp.Run(args)
	assert.Equal("ACO name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Unexpected flag
	args = []string{"bcda", "create-aco", "--abcd", "efg"}
	err = s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")
	buf.Reset()

	// Invalid CMS ID
	args = []string{"bcda", "create-aco", "--name", ACOName, "--cms-id", "ABCDE"}
	err = s.testApp.Run(args)
	assert.Equal("ACO CMS ID (--cms-id) is invalid", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()
}

func (s *CLITestSuite) TestImportCCLFDirectory() {
	targetACO := "A0002"
	assert := assert.New(s.T())

	postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, targetACO)
	defer postgrestest.DeleteCCLFFilesByCMSID(s.T(), s.db, targetACO)

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path, cleanup := testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/cclf/archives/valid2/")
	defer cleanup()

	args := []string{"bcda", "import-cclf-directory", "--directory", path}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF import.")
	assert.Contains(buf.String(), "Successfully imported 2 files.")
	assert.Contains(buf.String(), "Failed to import 0 files.")
	assert.Contains(buf.String(), "Skipped 1 files.")

	buf.Reset()
}

func (s *CLITestSuite) TestDeleteDirectoryContents() {
	assert := assert.New(s.T())
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	dirToDelete, err := ioutil.TempDir("", "*")
	assert.NoError(err)
	testUtils.MakeDirToDelete(s.Suite, dirToDelete)
	defer os.RemoveAll(dirToDelete)

	args := []string{"bcda", "delete-dir-contents", "--dirToDelete", dirToDelete}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), fmt.Sprintf("Successfully Deleted 4 files from %v", dirToDelete))
	buf.Reset()

	// File, not a directory
	file, err := ioutil.TempFile("", "*")
	assert.NoError(err)
	defer os.Remove(file.Name())
	args = []string{"bcda", "delete-dir-contents", "--dirToDelete", file.Name()}
	err = s.testApp.Run(args)
	assert.EqualError(err, fmt.Sprintf("unable to delete Directory Contents because %s does not reference a directory", file.Name()))
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

	conf.SetEnv(s.T(), "TESTDELETEDIRECTORY", "NOT/A/REAL/DIRECTORY")
	args = []string{"bcda", "delete-dir-contents", "--envvar", "TESTDELETEDIRECTORY"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "flag provided but not defined: -envvar")
	assert.NotContains(buf.String(), "Successfully Deleted")
	buf.Reset()

}

func (s *CLITestSuite) TestImportSuppressionDirectory() {
	assert := assert.New(s.T())

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path, cleanup := testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/synthetic1800MedicareFiles/test2/")
	defer cleanup()

	args := []string{"bcda", "import-suppression-directory", "--directory", path}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed 1-800-MEDICARE suppression data import.")
	assert.Contains(buf.String(), "Files imported: 2")
	assert.Contains(buf.String(), "Files failed: 0")
	assert.Contains(buf.String(), "Files skipped: 0")

	fs := postgrestest.GetSuppressionFileByName(s.T(), s.db,
		"T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000010",
		"T#EFT.ON.ACO.NGD1800.DPRF.D190816.T0241391")

	assert.Len(fs, 2)
	for _, f := range fs {
		postgrestest.DeleteSuppressionFileByID(s.T(), s.db, f.ID)
	}
}

func (s *CLITestSuite) TestImportSuppressionDirectory_Skipped() {
	assert := assert.New(s.T())

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path, cleanup := testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/suppressionfile_BadFileNames/")
	defer cleanup()

	args := []string{"bcda", "import-suppression-directory", "--directory", path}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed 1-800-MEDICARE suppression data import.")
	assert.Contains(buf.String(), "Files imported: 0")
	assert.Contains(buf.String(), "Files failed: 0")
	assert.Contains(buf.String(), "Files skipped: 2")
}

func (s *CLITestSuite) TestImportSuppressionDirectory_Failed() {
	assert := assert.New(s.T())

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path, cleanup := testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/suppressionfile_BadHeader/")
	defer cleanup()

	args := []string{"bcda", "import-suppression-directory", "--directory", path}
	err := s.testApp.Run(args)
	assert.EqualError(err, "one or more suppression files failed to import correctly")
	assert.Contains(buf.String(), "Completed 1-800-MEDICARE suppression data import.")
	assert.Contains(buf.String(), "Files imported: 0")
	assert.Contains(buf.String(), "Files failed: 1")
	assert.Contains(buf.String(), "Files skipped: 0")
}

func (s *CLITestSuite) TestBlacklistACO() {
	blacklistedCMSID := testUtils.RandomHexID()[0:4]
	notBlacklistedCMSID := testUtils.RandomHexID()[0:4]
	notFoundCMSID := testUtils.RandomHexID()[0:4]

	blacklistedACO := models.ACO{UUID: uuid.NewUUID(), CMSID: &blacklistedCMSID,
		TerminationDetails: &models.Termination{
			TerminationDate: time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			CutoffDate:      time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			BlacklistType:   models.Involuntary,
		}}
	notBlacklistedACO := models.ACO{UUID: uuid.NewUUID(), CMSID: &notBlacklistedCMSID,
		TerminationDetails: nil}

	defer func() {
		postgrestest.DeleteACO(s.T(), s.db, blacklistedACO.UUID)
		postgrestest.DeleteACO(s.T(), s.db, notBlacklistedACO.UUID)
	}()

	postgrestest.CreateACO(s.T(), s.db, blacklistedACO)
	postgrestest.CreateACO(s.T(), s.db, notBlacklistedACO)

	s.NoError(s.testApp.Run([]string{"bcda", "unblacklist-aco", "--cms-id", blacklistedCMSID}))
	s.NoError(s.testApp.Run([]string{"bcda", "blacklist-aco", "--cms-id", notBlacklistedCMSID}))

	s.Error(s.testApp.Run([]string{"bcda", "unblacklist-aco", "--cms-id", notFoundCMSID}))
	s.Error(s.testApp.Run([]string{"bcda", "blacklist-aco", "--cms-id", notFoundCMSID}))

	newlyUnblacklistedACO := postgrestest.GetACOByUUID(s.T(), s.db, blacklistedACO.UUID)
	s.False(newlyUnblacklistedACO.Blacklisted())

	newlyBlacklistedACO := postgrestest.GetACOByUUID(s.T(), s.db, notBlacklistedACO.UUID)
	s.True(newlyBlacklistedACO.Blacklisted())
}

func getRandomPort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err.Error())
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Fatal(err.Error())
		}
	}()

	return listener.Addr().(*net.TCPAddr).Port
}

func (s *CLITestSuite) TestCloneCCLFZips() {
	assert := assert.New(s.T())

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	// set up a test directory for cclf file generating and cloning
	path, err := ioutil.TempDir(".", "clone_cclf")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(path)

	// test cclf zip file names and a dummy file name that should not be cloned
	zipFiles := []string{
		"T.BCD.A0002.ZCY18.D181120.T9999990",
		"T.BCD.A0002.ZCY18.D181120.T9999991",
		"T.BCD.A0002.ZCY18.D181120.T9999992",
		"P.BCD.E0002.ZCY20.D200914.T0850090",
		"P.BCD.V002.ZCY20.D201002.T0811490",
		"not_a_cclf_file",
	}
	// cclf file names that are contained within the cclf zip files
	cclfFiles := []string{
		"T.BCD.A0001.ZC48Y18.D181120.T1000001",
		"T.BCD.A0001.ZCAY18.D181120.T1000001",
		"T.BCD.A0001.ZCBY18.D181120.T1000001",
		"T.BCD.A0001.ZC48Y18.D181120.T1000002",
		"T.BCD.A0001.ZC48Y18.D181120.T1000003",
		"P.V001.ACO.ZC8Y20.D201002.T0806400",
		"P.CEC.ZC8Y20.D201108.T0958300",
	}

	// create the test files under the temporary directory
	for _, zf := range zipFiles {
		err := createTestZipFile(filepath.Join(path, zf), cclfFiles...)
		if err != nil {
			log.Fatal(err)
		}
	}

	beforecount := getFileCount(s.T(), path)

	args := []string{"bcda", "generate-cclf-runout-files", "--directory", path}
	err = s.testApp.Run(args)
	fmt.Print(buf.String())
	assert.Nil(err)
	assert.Contains(buf.String(), "Completed CCLF runout file generation.")
	assert.Contains(buf.String(), "Generated 5 zip files.")
	buf.Reset()

	// runout zip file names that will be generated
	zipRFiles := []string{
		"T.BCD.A0002.ZCR18.D181120.T9999990",
		"T.BCD.A0002.ZCR18.D181120.T9999991",
		"T.BCD.A0002.ZCR18.D181120.T9999992",
		"P.BCD.E0002.ZCR20.D200914.T0850090",
		"P.BCD.V002.ZCR20.D201002.T0811490",
	}

	// assert the zip file count matches after cloning
	assert.Equal(beforecount+len(zipRFiles), getFileCount(s.T(), path))

	// runout cclf file names that will be generated for each zip file
	cclfRFiles := []string{
		"T.BCD.A0001.ZC48R18.D181120.T1000001",
		"T.BCD.A0001.ZCAR18.D181120.T1000001",
		"T.BCD.A0001.ZCBR18.D181120.T1000001",
		"T.BCD.A0001.ZC48R18.D181120.T1000002",
		"T.BCD.A0001.ZC48R18.D181120.T1000003",
		"P.V001.ACO.ZC8R20.D201002.T0806400",
		"P.CEC.ZC8R20.D201108.T0958300",
	}

	// assert that each zip was cloned with the proper name and each zip file
	// contains the correct cclf file clones
	for _, zrf := range zipRFiles {
		assert.FileExists(filepath.Join(path, zrf))

		zr, err := zip.OpenReader(filepath.Join(path, zrf))
		assert.NoError(err)
		defer zr.Close()

		for i, f := range zr.File {
			assert.Equal(cclfRFiles[i], f.Name)
		}
	}
}

func (s *CLITestSuite) TestGenerateAlrData() {
	initialCount := postgrestest.GetALRCount(s.T(), s.db, "A9994")
	args := []string{"bcda", "generate-synthetic-alr-data", "--cms-id", "A9994",
		"--alr-template-file", "../alr/gen/testdata/PY21ALRTemplatePrelimProspTable1.csv"}
	err := s.testApp.Run(args)
	assert.NoError(s.T(), err)
	assert.Greater(s.T(), postgrestest.GetALRCount(s.T(), s.db, "A9994"), initialCount)

	// No CCLF file
	err = s.testApp.Run([]string{"bcda", "generate-synthetic-alr-data", "--cms-id", "UNKNOWN_ACO",
		"--alr-template-file", "../alr/gen/testdata/PY21ALRTemplatePrelimProspTable1.csv"})
	assert.EqualError(s.T(), err, "no CCLF8 file found for CMS ID UNKNOWN_ACO")
}

// func (s *CLITestSuite) TestFoo() {
// 	SomeuntestedFunction()
// }
func (s *CLITestSuite) TestBar() {
	SometestedFunction()
}

func (s *CLITestSuite) setupJobFile(modified time.Time, status models.JobStatus, rootPath string) (uint, *os.File) {
	j := models.Job{
		ACOID:      s.testACO.UUID,
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     status,
		UpdatedAt:  modified,
	}

	postgrestest.CreateJobs(s.T(), s.db, &j)

	path := fmt.Sprintf("%s/%d", rootPath, j.ID)

	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		s.T().Error(err)
	}
	jobFile, err := os.Create(fmt.Sprintf("%s/%s", path, "fake.ndjson"))
	if err != nil {
		s.T().Error(err)
	}
	defer jobFile.Close()

	return j.ID, jobFile
}

func createTestZipFile(zFile string, cclfFiles ...string) error {
	zf, err := os.Create(zFile)
	if err != nil {
		return err
	}
	defer zf.Close()

	w := zip.NewWriter(zf)

	for _, f := range cclfFiles {
		f, err := w.Create(f)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte("foo bar"))
		if err != nil {
			return err
		}
	}

	return w.Close()
}

func getFileCount(t *testing.T, path string) int {
	f, err := ioutil.ReadDir(path)
	assert.NoError(t, err)
	return len(f)
}

func assertFileNotExists(t *testing.T, path string) {
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "file %s should not be found", path)
}
