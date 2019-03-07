package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
)

const BADUUID = "QWERTY-ASDFG-ZXCVBN-POIUYT"
const TOKENHEADER string = "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9."

type MainTestSuite struct {
	testUtils.AuthTestSuite
	testApp       *cli.App
	expectedSizes map[string]int
}

func (s *MainTestSuite) SetupSuite() {
	s.expectedSizes = map[string]int{
		"dev":    50,
		"small":  10,
		"medium": 25,
		"large":  100,
	}
}

func (s *MainTestSuite) SetupTest() {
	s.testApp = setUpApp()
}

func (s *MainTestSuite) TearDownTest() {
	testUtils.PrintSeparator()
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func (s *MainTestSuite) TestGetEnvInt() {
	const DEFAULT_VALUE = 200
	os.Setenv("TEST_ENV_STRING", "blah")
	os.Setenv("TEST_ENV_INT", "232")

	assert.Equal(s.T(), 232, getEnvInt("TEST_ENV_INT", DEFAULT_VALUE))
	assert.Equal(s.T(), DEFAULT_VALUE, getEnvInt("TEST_ENV_STRING", DEFAULT_VALUE))
	assert.Equal(s.T(), DEFAULT_VALUE, getEnvInt("FAKE_ENV", DEFAULT_VALUE))
}

func (s *MainTestSuite) TestSetup() {
	assert.Equal(s.T(), 1, 1)
	app := setUpApp()
	assert.Equal(s.T(), app.Name, Name)
	assert.Equal(s.T(), app.Usage, Usage)
}

func (s *MainTestSuite) TestAutoMigrate() {
	// Plenty of other tests will rely on this happening
	// Other tests run these lines so as long as this doesn't error it sb fine
	args := []string{"bcda", "sql-migrate"}
	err := s.testApp.Run(args)
	assert.Nil(s.T(), err)
}

func (s *MainTestSuite) TestCreateACO() {

	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	s.SetupAuthBackend()

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	// Successful ACO creation
	ACOName := "UNIT TEST ACO"
	args := []string{"bcda", "create-aco", "--name", ACOName}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	acoUUID := strings.TrimSpace(buf.String())
	var testACO models.ACO
	db.First(&testACO, "Name=?", "UNIT TEST ACO")
	assert.Equal(testACO.UUID.String(), acoUUID)
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

	// we currently allow ACOs with duplicate names
}

func (s *MainTestSuite) TestCreateUser() {

	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	acoUUID := "DBBD1CE1-AE24-435C-807D-ED45953077D3"
	name, email := "Unit Test", "UnitTest@mail.com"

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())

	// Successful user creation
	args := []string{"bcda", "create-user", "--name", name, "--aco-id", acoUUID, "--email", email}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	userUUID := strings.TrimSpace(buf.String())
	var testUser models.User
	db.First(&testUser, "Email=?", email)
	assert.Equal(testUser.UUID.String(), userUUID)
	buf.Reset()

	// Bad/Negative tests

	// No parameters
	args = []string{"bcda", "create-user"}
	err = s.testApp.Run(args)
	assert.Equal("ACO ID (--aco-id) must be provided\nName (--name) must be provided\nEmail address (--email) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Name only
	args = []string{"bcda", "create-user", "--name", name}
	err = s.testApp.Run(args)
	assert.Equal("ACO ID (--aco-id) must be provided\nEmail address (--email) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// ACO ID only
	args = []string{"bcda", "create-user", "--aco-id", acoUUID}
	err = s.testApp.Run(args)
	assert.Equal("Name (--name) must be provided\nEmail address (--email) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Email only
	args = []string{"bcda", "create-user", "--email", email}
	err = s.testApp.Run(args)
	assert.Equal("ACO ID (--aco-id) must be provided\nName (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Unexpected flag
	args = []string{"bcda", "create-user", "--abcd", "efg"}
	err = s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")
	buf.Reset()

	// Blank UUID
	args = []string{"bcda", "create-user", "--name", name, "--aco-id", "", "--email", email}
	err = s.testApp.Run(args)
	assert.Equal("ACO ID (--aco-id) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Bad UUID
	args = []string{"bcda", "create-user", "--name", name, "--aco-id", BADUUID, "--email", email}
	err = s.testApp.Run(args)
	assert.Equal("ACO ID must be a UUID", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Blank Name
	args = []string{"bcda", "create-user", "--name", "", "--aco-id", acoUUID, "--email", email}
	err = s.testApp.Run(args)
	assert.Equal("Name (--name) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Blank E-mail address
	args = []string{"bcda", "create-user", "--name", name, "--aco-id", acoUUID, "--email", ""}
	err = s.testApp.Run(args)
	assert.Equal("Email address (--email) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Multiple blank input params
	args = []string{"bcda", "create-user", "--name", "", "--aco-id", "", "--email", ""}
	err = s.testApp.Run(args)
	assert.Equal("ACO ID (--aco-id) must be provided\nName (--name) must be provided\nEmail address (--email) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Duplicate User
	args = []string{"bcda", "create-user", "--name", name, "--aco-id", acoUUID, "--email", email}
	err = s.testApp.Run(args)
	assert.Contains(err.Error(), email, "%s should contain '%s' and 'already exists'", err, email)
	assert.Equal(0, buf.Len())
}

func (s *MainTestSuite) TestCreateToken() {
	// Set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())
	userUUID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	badUUID := "not_a_uuid"
	clientSecret := "not_a_secret"

	// Unexpected flag
	args := []string{"bcda", "create-token", "--abcd", "efg"}
	err := s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")
	buf.Reset()

	// No parameters
	args = []string{"bcda", "create-token"}
	err = s.testApp.Run(args)
	assert.Equal("ID (--id) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Blank ID
	args = []string{"bcda", "create-token", "--id", "", "--secret", clientSecret}
	err = s.testApp.Run(args)
	assert.Equal("ID (--id) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Alpha auth section
	originalAuthProvider := auth.GetProviderName()
	defer auth.SetProvider(originalAuthProvider)
	auth.SetProvider("alpha")
	// Test alpha auth bad ID
	args = []string{"bcda", "create-token", "--id", badUUID}
	err = s.testApp.Run(args)
	assert.Contains(err.Error(), "must be a UUID")
	buf.Reset()

	// Test alpha auth successful creation
	args = []string{"bcda", "create-token", "--id", userUUID}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.NotNil(buf)
	accessTokenString := strings.TrimSpace(buf.String())
	assert.Nil(err)
	assert.NotEmpty(accessTokenString)
	buf.Reset()
}

func checkTokenInfo(s *MainTestSuite, tokenInfo string) {
	assert := assert.New(s.T())
	assert.NotNil(tokenInfo)
	if len(tokenInfo) == 0 {
		assert.FailNow("tokenInfo has no content")
	}
	lines := strings.Split(tokenInfo, "\n")
	assert.Equal(3, len(lines))
	expDate, err := time.Parse(time.RFC850, lines[0])
	assert.Nil(err)
	assert.NotNil(expDate)
	assert.Regexp("[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}", lines[1], "no correctly formatted token id in second line %s", lines[1])
	assert.True(strings.HasPrefix(lines[2], TOKENHEADER), "incorrect token header %s", lines[2])
	assert.InDelta(500, len(tokenInfo), 100, "encoded token string length should be 500+-100; it is %d\n%s", len(tokenInfo), lines[2])
}

func (s *MainTestSuite) TestCreateAlphaTokenCLI() {

	// Due to the way the resulting token is returned to the user, not all scenarios can be executed via CLI

	// set up the test app writer (to redirect CLI responses from stdout to a byte buffer)
	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	assert := assert.New(s.T())
	var outputPattern *regexp.Regexp
	switch auth.GetProvider().(type) {
	case auth.AlphaAuthPlugin:
		outputPattern = regexp.MustCompile(`[a-zA-Z]+, \d+-[a-zA-Z]{3}-\d{2} \d{2}:\d{2}:\d{2} [A-Z]+\n[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}\n.+\..+\..+\n`)
	case auth.OktaAuthPlugin:
		outputPattern = regexp.MustCompile(`[a-zA-Z]+, \d+-[a-zA-Z]{3}-\d{2} \d{2}:\d{2}:\d{2} [A-Z]+\n[!-~]{16}\n\[!-~]{32}n`)
	}

	// execute positive scenarios via CLI
	args := []string{"bcda", "create-alpha-token", "--ttl", "720", "--size", "Dev"}
	err := s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())
	buf.Reset()

	// ttl is optional when using the CLI
	args = []string{"bcda", "create-alpha-token", "--size", "Dev"}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())
	buf.Reset()

	args = []string{"bcda", "create-alpha-token", "--size", "DEV"}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Regexp(outputPattern, buf.String())
	buf.Reset()

	// Execute CLI with invalid inputs
	args = []string{"bcda", "create-alpha-token"}
	err = s.testApp.Run(args)
	assert.Equal("invalid argument for --size.  Please use 'Dev', 'Small', 'Medium', 'Large', or 'Extra_Large'", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	args = []string{"bcda", "create-alpha-token", "--ttl", "ABCD", "--size", "Dev"}
	err = s.testApp.Run(args)
	assert.Equal("invalid argument 'ABCD' for --ttl; should be an integer > 0", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	args = []string{"bcda", "create-alpha-token", "--ttl", "720", "--size", "ABCD"}
	err = s.testApp.Run(args)
	assert.Equal("invalid argument for --size.  Please use 'Dev', 'Small', 'Medium', 'Large', or 'Extra_Large'", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	args = []string{"bcda", "create-alpha-token", "--abcd", "efg"}
	err = s.testApp.Run(args)
	assert.Equal("flag provided but not defined: -abcd", err.Error())
	assert.Contains(buf.String(), "Incorrect Usage: flag provided but not defined")

	// To execute all scenarios, invoke the rest of the tests directly (not by CLI)
	// (this is required in order to validate the strings returned)

	alphaTokenInfo, err := createAlphaToken(1, "Dev")
	assert.Nil(err)
	checkTokenInfo(s, alphaTokenInfo)

	anotherTokenInfo, err := createAlphaToken(720, "Dev")
	assert.Nil(err)
	checkTokenInfo(s, anotherTokenInfo)

	l1 := strings.Split(alphaTokenInfo, "\n")
	l2 := strings.Split(anotherTokenInfo, "\n")
	assert.NotEqual(l1[0], l2[0], "alpha expiration dates should be different (%s == %s)", l1[0], l2[0])
	assert.NotEqual(l1[1], l2[1], "alpha token uuids should be different (%s == %s)", l1[1], l2[1])
}

func (s *MainTestSuite) TestArchiveExpiring() {

	// init
	autoMigrate()
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
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Completed",
	}
	db.Save(&j)
	assert.NotNil(j.ID)

	os.Setenv("FHIR_PAYLOAD_DIR", "../bcdaworker/data/test")
	os.Setenv("FHIR_ARCHIVE_DIR", "../bcdaworker/data/test/archive")
	id := int(j.ID)
	assert.NotNil(id)

	path := fmt.Sprintf("%s/%d/", os.Getenv("FHIR_PAYLOAD_DIR"), id)
	newpath := os.Getenv("FHIR_ARCHIVE_DIR")

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
	expPath := fmt.Sprintf("%s/%d/fake.ndjson", os.Getenv("FHIR_ARCHIVE_DIR"), id)
	_, err = ioutil.ReadFile(expPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(expPath, "File not Found")

	var testjob models.Job
	db.First(&testjob, "id = ?", j.ID)

	// check the status of the job
	assert.Equal("Archived", testjob.Status)

	// clean up
	os.RemoveAll(os.Getenv("FHIR_ARCHIVE_DIR"))
}

func (s *MainTestSuite) TestArchiveExpiringWithThreshold() {

	// init
	autoMigrate()
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// save a job to our db
	j := models.Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Completed",
	}
	db.Save(&j)
	assert.NotNil(s.T(), j.ID)

	os.Setenv("FHIR_PAYLOAD_DIR", "../bcdaworker/data/test")
	os.Setenv("FHIR_ARCHIVE_DIR", "../bcdaworker/data/test/archive")
	id := int(j.ID)
	assert.NotNil(s.T(), id)

	path := fmt.Sprintf("%s/%d/", os.Getenv("FHIR_PAYLOAD_DIR"), id)

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
	dataPath := fmt.Sprintf("%s/%d/fake.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), id)
	_, err = ioutil.ReadFile(dataPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(s.T(), dataPath, "File not Found")

	var testjob models.Job
	db.First(&testjob, "id = ?", j.ID)

	// check the status of the job
	assert.Equal(s.T(), "Completed", testjob.Status)

	// clean up
	os.Remove(dataPath)
}

func setupArchivedJob(s *MainTestSuite, email string, modified time.Time) int {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	s.SetupAuthBackend()
	acoUUID, err := createACO("ACO " + email)
	assert.Nil(s.T(), err)

	userUUID, err := createUser(acoUUID, "Unit Test", email)
	assert.Nil(s.T(), err)

	// save a job to our db
	j := models.Job{
		ACOID:      uuid.Parse(acoUUID),
		UserID:     uuid.Parse(userUUID),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Archived",
	}
	db.Save(&j)
	db.Exec("UPDATE jobs SET updated_at=? WHERE id = ?", modified.Format("2006-01-02 15:04:05"), j.ID)
	db.First(&j, "id = ?", j.ID)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), j.ID)
	// compare times using formatted strings to avoid differences (like nano seconds) that we don't care about
	assert.Equal(s.T(), modified.Format("2006-01-02 15:04:05"), j.UpdatedAt.Format("2006-01-02 15:04:05"), "UpdatedAt should match %v", modified)

	return int(j.ID)
}

func setupJobArchiveFile(s *MainTestSuite, email string, modified time.Time, accessed time.Time) (int, *os.File) {
	// directory structure is FHIR_ARCHIVE_DIR/<JobId>/<datafile>.ndjson
	// for reference, see main.archiveExpiring() and its companion tests above
	jobId := setupArchivedJob(s, email, modified)
	path := fmt.Sprintf("%s/%d", os.Getenv("FHIR_ARCHIVE_DIR"), jobId)

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

func (s *MainTestSuite) TestCleanArchive() {

	// init
	autoMigrate()
	const Threshold = 30
	now := time.Now()

	assert := assert.New(s.T())

	// condition: FHIR_ARCHIVE_DIR doesn't exist
	os.Unsetenv("FHIR_ARCHIVE_DIR")
	args := []string{"bcda", "cleanup-archive", "--threshold", strconv.Itoa(Threshold)}
	err := s.testApp.Run(args)
	assert.Nil(err)
	os.Setenv("FHIR_ARCHIVE_DIR", "../bcdaworker/data/test/archive")

	// condition: no jobs exist
	args = []string{"bcda", "cleanup-archive", "--threshold", strconv.Itoa(Threshold)}
	err = s.testApp.Run(args)
	assert.Nil(err)

	// create a file that was last modified before the Threshold, but accessed after it
	modified := now.Add(-(time.Hour * (Threshold + 1)))
	accessed := now.Add(-(time.Hour * (Threshold - 1)))
	beforeJobId, before := setupJobArchiveFile(s, "before@test.com", modified, accessed)
	defer before.Close()

	// create a file that is clearly after the threshold (unless the threshold is 0)
	afterJobId, after := setupJobArchiveFile(s, "after@test.com", now, now)
	defer after.Close()

	// condition: bad threshold value
	args = []string{"bcda", "cleanup-archive", "--threshold", "abcde"}
	err = s.testApp.Run(args)
	assert.NotNil(err)

	// condition: before < Threshold < after <= now
	// a file created before the Threshold should be deleted; one created after should not
	// we use last modified as a proxy for created, because these files should not be changed after creation
	args = []string{"bcda", "cleanup-archive", "--threshold", strconv.Itoa(Threshold)}
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
	db.First(&beforeJob, "id = ?", beforeJobId)
	assert.Equal("Expired", beforeJob.Status)

	assert.FileExists(after.Name(), "%s not found; it should have been", after.Name())

	var afterJob models.Job
	db.First(&afterJob, "id = ?", afterJobId)
	assert.Equal("Archived", afterJob.Status)

	// I think this is an application directory and should always exist, but that doesn't seem to be the norm
	os.RemoveAll(os.Getenv("FHIR_ARCHIVE_DIR"))
}

func (s *MainTestSuite) TestRevokeToken() {
	originalAuthProvider := auth.GetProviderName()
	defer auth.SetProvider(originalAuthProvider)
	auth.SetProvider("alpha")
	// init
	s.SetupAuthBackend()

	assert := assert.New(s.T())

	// Create a token
	tokenInfo, err := createAlphaToken(720, "Small")
	assert.Nil(err)
	checkTokenInfo(s, tokenInfo)
	alphaTokenData := strings.Split(tokenInfo, "\n")
	alphaTokenString := alphaTokenData[2]

	// Create a token
	userUUID := "82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"
	tokenString, err := createAccessToken(userUUID, "")
	assert.Nil(err)

	buf := new(bytes.Buffer)
	s.testApp.Writer = buf

	// Negative case - attempt to revoke a token passing in a blank token string
	args := []string{"bcda", "revoke-token", "--access-token", ""}
	err = s.testApp.Run(args)
	assert.Equal("Access token (--access-token) must be provided", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Negative case - attempt to revoke a token passing in an invalid token string
	args = []string{"bcda", "revoke-token", "--access-token", "abcdefg"}
	err = s.testApp.Run(args)
	assert.Equal("token contains an invalid number of segments", err.Error())
	assert.Equal(0, buf.Len())
	buf.Reset()

	// Positive case - revoke a token passing in a valid token string (alpha)
	args = []string{"bcda", "revoke-token", "--access-token", alphaTokenString}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Equal("Access token has been deactivated\n", buf.String())
	buf.Reset()

	// Positive case - revoke a token passing in a valid token string
	args = []string{"bcda", "revoke-token", "--access-token", tokenString}
	err = s.testApp.Run(args)
	assert.Nil(err)
	assert.Equal("Access token has been deactivated\n", buf.String())
}

func (s *MainTestSuite) TestStartApi() {

	// Negative case
	originalQueueDbUrl := os.Getenv("QUEUE_DATABASE_URL")
	os.Setenv("QUEUE_DATABASE_URL", "http://bad url.com/")
	args := []string{"bcda", "start-api"}
	err := s.testApp.Run(args)
	assert.NotNil(s.T(), err)
	os.Setenv("QUEUE_DATABASE_URL", originalQueueDbUrl)

	// We cannot test the positive case because we don't want to start the HTTP Server in unit test environment
}

func (s *MainTestSuite) TestCreateAlphaToken() {
	const ttl = 42
	checkStructure(s, ttl, "dev")
}

func (s *MainTestSuite) TestCreateSmallAlphaToken() {
	const ttl = 24
	checkStructure(s, ttl, "Small")
}

func (s *MainTestSuite) TestCreateMediumAlphaToken() {
	const ttl = 24 * 365
	checkStructure(s, ttl, "MeDIum")
}

func (s *MainTestSuite) TestCreateLargeAlphaToken() {
	endOfUnixTime, _ := time.Parse(time.RFC1123, "Tue, 19 Jan 2038 03:14:07 GMT")
	ttl := int(time.Until(endOfUnixTime).Hours())
	checkStructure(s, ttl, "Large")
}

func checkTTL(s *MainTestSuite, claims jwt.MapClaims, ttl int) {
	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	assert.NotNil(s.T(), iat)
	assert.NotNil(s.T(), exp)

	delta, err := time.ParseDuration(fmt.Sprintf("%dh", ttl))
	if err != nil {
		assert.Fail(s.T(), "Can't parse ttl value of %s", ttl)
	}

	assert.True(s.T(), assert.WithinDuration(s.T(), iat, exp, delta, "expires date %s not within %s hours of issued at", exp.Format(time.RFC850), ttl))
}

func checkStructure(s *MainTestSuite, ttl int, acoSize string) {
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	tokenInfo, err := createAlphaToken(ttl, acoSize)
	require.Nil(s.T(), err, "Unexpected error %v", err)
	lines := strings.Split(tokenInfo, "\n")
	assert.Equal(s.T(), 3, len(lines))
	tokenString := lines[2]
	require.NotEmpty(s.T(), tokenString)
	//  need to handle credentials for different providers here; this will no longer always be a tokenString
	switch auth.GetProvider().(type) {
	case auth.AlphaAuthPlugin:
		claims := s.AuthBackend.GetJWTClaims(tokenString)
		require.NotNil(s.T(), claims)
		acoUUID := claims["aco"].(string)
		assert.NotNil(s.T(), acoUUID)
		var count int
		db.Table("beneficiaries").Where("aco_id = ?", acoUUID).Count(&count)
		assert.Equal(s.T(), s.expectedSizes[strings.ToLower(acoSize)], count)
		checkTTL(s, claims, ttl)
	case auth.OktaAuthPlugin:
		// nothing to see here; move along
	}
}
