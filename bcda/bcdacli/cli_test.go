package bcdacli

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/go-chi/chi/v5"
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

	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(&s.Suite, dir)

	s.db = database.Connect()
	db = s.db
	repository = postgres.NewRepository(s.db)

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

func (s *CLITestSuite) SetProvider(p auth.Provider) {
	provider = p
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

func (s *CLITestSuite) TestIgnoreSignals() {
	// 1. Start the signal handler and retrieve the signal channel.
	// 2. Retrieve the process which allows access to Signal handling.
	// 3. Sending both SIGINT and SIGTERM signals to verify the signals are being handled (ignored).
	// 4. Assert that the signal channel is empty, meaning both signals have been handled.
	sigs := ignoreSignals()
	defer signal.Stop(sigs)

	process, err := os.FindProcess(os.Getpid())
	assert.NoError(s.T(), err)

	err = process.Signal(syscall.SIGINT)
	assert.NoError(s.T(), err)
	err = process.Signal(syscall.SIGTERM)
	assert.NoError(s.T(), err)

	time.Sleep(100 * time.Millisecond) // Assure both signal requests have a chance to be handled

	assert.Equal(s.T(), 0, len(sigs), "ignoreSignals has not pulled the signal from sigs channel")
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
			m.On("FindAndCreateACOCredentials", *s.testACO.CMSID, ips).Return("mock\ncreds\ntest", nil)

			buf := new(bytes.Buffer)
			s.testApp.Writer = buf

			msg, err := generateClientCredentials(m, *s.testACO.CMSID, ips)
			assert.Nil(t, err)
			assert.Regexp(t, regexp.MustCompile(".+\n.+\n.+"), msg)
			assert.Equal(t, "mock\ncreds\ntest", msg)
			m.AssertExpectations(t)
		})
	}
}

func (s *CLITestSuite) TestGenerateClientCredentials_InvalidID() {
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf
	assert := assert.New(s.T())

	args := []string{"bcda", constants.GenClientCred, constants.CMSIDArg, "9994"}
	err := s.testApp.Run(args)
	assert.ErrorContains(err, "no ACO record found for 9994")
	assert.Empty(buf)
	buf.Reset()

	args = []string{"bcda", constants.GenClientCred, constants.CMSIDArg, "A6543"}
	err = s.testApp.Run(args)
	assert.ErrorContains(err, "no ACO record found for A6543")
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

	// execute positive scenario
	msg, err := resetClientCredentials(repository, mock, *s.testACO.CMSID)
	assert.Nil(err)
	assert.Regexp(outputPattern, msg)

	// Execute with invalid ACO CMS ID
	msg, err = resetClientCredentials(repository, mock, "BLAH")
	assert.Equal("no ACO record found for BLAH", err.Error())
	assert.Equal(0, len(msg))

	mock.AssertExpectations(s.T())
}

func (s *CLITestSuite) TestRevokeToken() {
	assert := assert.New(s.T())

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	accessToken := uuid.New()
	mock := &auth.MockProvider{}
	mock.On("RevokeAccessToken", accessToken).Return(nil)

	err := revokeAccessToken(mock, accessToken)
	assert.Nil(err)

	// Negative case - attempt to revoke a token passing in a blank token string
	err = revokeAccessToken(mock, "")
	assert.Equal("Access token (--access-token) must be provided", err.Error())
	mock.AssertExpectations(s.T())
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
	args := []string{"bcda", constants.CreateGroupArg, "--id", id, constants.NameArg, name, constants.ACOIDArg, acoID}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Equal("test-create-group-id", buf.String())
}

func (s *CLITestSuite) TestCreateGroup_InvalidACOID() {
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	// Invalid format
	args := []string{"bcda", constants.CreateGroupArg, "--id", "invalid-aco-id-group", constants.NameArg, "Invalid ACO ID Group", constants.ACOIDArg, "1234"}
	err := s.testApp.Run(args)
	assert.EqualError(s.T(), err, "ACO ID (--aco-id) must be a supported CMS ID or UUID")
	assert.Empty(s.T(), buf.String())
	buf.Reset()

	// Valid format, but no matching ACO
	aUUID := "4e5519cb-428d-4934-a3f8-6d3efb1277b7"
	args = []string{"bcda", constants.CreateGroupArg, "--id", "invalid-aco-id-group", constants.NameArg, "Invalid ACO ID Group", constants.ACOIDArg, aUUID}
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
	args := []string{"bcda", constants.CreateACOID, constants.NameArg, ACOName}

	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID := uuid.Parse(strings.TrimSpace(buf.String()))

	testACO := postgrestest.GetACOByUUID(s.T(), s.db, acoUUID)
	assert.Equal(ACOName, testACO.Name)
	buf.Reset()
	defer postgrestest.DeleteACO(s.T(), s.db, acoUUID)

	ACO2Name := "Unit Test ACO 2"
	aco2ID := "A9999"
	args = []string{"bcda", constants.CreateACOID, constants.NameArg, ACO2Name, constants.CMSIDArg, aco2ID}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID = uuid.Parse(strings.TrimSpace(buf.String()))

	testACO2 := postgrestest.GetACOByUUID(s.T(), s.db, acoUUID)
	assert.Equal(ACO2Name, testACO2.Name)
	assert.Equal(aco2ID, *testACO2.CMSID)
	buf.Reset()
	defer postgrestest.DeleteACO(s.T(), s.db, acoUUID)

	// Negative tests

	// No parameters
	args = []string{"bcda", constants.CreateACOID}
	err = s.testApp.Run(args)
	assert.Equal(constants.TestACOName, err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// No ACO Name
	badACO := ""
	args = []string{"bcda", constants.CreateACOID, constants.NameArg, badACO}
	err = s.testApp.Run(args)
	assert.Equal(constants.TestACOName, err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// ACO name without flag
	args = []string{"bcda", constants.CreateACOID, ACOName}
	err = s.testApp.Run(args)
	assert.Equal(constants.TestACOName, err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Unexpected flag
	args = []string{"bcda", constants.CreateACOID, "--abcd", "efg"}
	err = s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")
	buf.Reset()

	// Invalid CMS ID
	args = []string{"bcda", constants.CreateACOID, constants.NameArg, ACOName, constants.CMSIDArg, "ABCDE"}
	err = s.testApp.Run(args)
	assert.Equal("ACO CMS ID (--cms-id) is invalid", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()
}

func (s *CLITestSuite) TestImportSuppressionDirectoryFromLocal() {
	assert := assert.New(s.T())

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	path, cleanup := testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/synthetic1800MedicareFiles/test2/")
	defer cleanup()

	args := []string{"bcda", constants.ImportSupDir, constants.DirectoryArg, path}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Contains(buf.String(), constants.CompleteMedSupDataImp)
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

func (s *CLITestSuite) TestDenylistACO() {
	denylistedCMSID := testUtils.RandomHexID()[0:4]
	notDenylistedCMSID := testUtils.RandomHexID()[0:4]
	notFoundCMSID := testUtils.RandomHexID()[0:4]

	denylistedACO := models.ACO{UUID: uuid.NewUUID(), CMSID: &denylistedCMSID,
		TerminationDetails: &models.Termination{
			TerminationDate: time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			CutoffDate:      time.Date(2020, time.December, 31, 23, 59, 59, 0, time.Local),
			DenylistType:    models.Involuntary,
		}}
	notDenylistedACO := models.ACO{UUID: uuid.NewUUID(), CMSID: &notDenylistedCMSID,
		TerminationDetails: nil}

	defer func() {
		postgrestest.DeleteACO(s.T(), s.db, denylistedACO.UUID)
		postgrestest.DeleteACO(s.T(), s.db, notDenylistedACO.UUID)
	}()

	postgrestest.CreateACO(s.T(), s.db, denylistedACO)
	postgrestest.CreateACO(s.T(), s.db, notDenylistedACO)

	s.NoError(s.testApp.Run([]string{"bcda", "undenylist-aco", constants.CMSIDArg, denylistedCMSID}))
	s.NoError(s.testApp.Run([]string{"bcda", "denylist-aco", constants.CMSIDArg, notDenylistedCMSID}))

	s.Error(s.testApp.Run([]string{"bcda", "undenylist-aco", constants.CMSIDArg, notFoundCMSID}))
	s.Error(s.testApp.Run([]string{"bcda", "denylist-aco", constants.CMSIDArg, notFoundCMSID}))

	newlyUndenylistedACO := postgrestest.GetACOByUUID(s.T(), s.db, denylistedACO.UUID)
	s.False(newlyUndenylistedACO.Denylisted())

	newlyDenylistedACO := postgrestest.GetACOByUUID(s.T(), s.db, notDenylistedACO.UUID)
	s.True(newlyDenylistedACO.Denylisted())
}

func getRandomPort(t *testing.T) int {
	listener, err := net.Listen("tcp", "localhost:0")
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
	path, err := os.MkdirTemp(".", "clone_cclf")
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

	args := []string{"bcda", "generate-cclf-runout-files", constants.DirectoryArg, path}
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
	f, err := os.ReadDir(path)
	assert.NoError(t, err)
	return len(f)
}
