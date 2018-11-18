package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/auth"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

const BADUUID = "QWERTY-ASDFG-ZXCVBN-POIUYT"

type MainTestSuite struct {
	testUtils.AuthTestSuite
	testApp *cli.App
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
func (s *MainTestSuite) TestSetup() {
	assert.Equal(s.T(), 1, 1)
	app := setUpApp()
	assert.Equal(s.T(), app.Name, Name)
	assert.Equal(s.T(), app.Usage, Usage)
}

func (s *MainTestSuite) TestAutoMigrate() {
	// Plenty of other tests will rely on this happening
	// Other tests run these lines so as long as this doesn't error it sb fine
	autoMigrate()
}

func (s *MainTestSuite) TestCreateACO() {
	db := database.GetGORMDbConnection()
	//args := []string{CreateACO, "--name", "TEST_ACO"}
	//err := s.testApp.Run(args)
	//assert.Nil(s.T(), err)
	s.SetupAuthBackend()
	ACOName := "UNIT TEST ACO"
	acoUUID, err := createACO(ACOName)
	assert.NotNil(s.T(), acoUUID)
	assert.Nil(s.T(), err)
	var testACO auth.ACO
	db.First(&testACO, "Name=?", "UNIT TEST ACO")
	assert.Equal(s.T(), testACO.UUID.String(), acoUUID)

	// Might as well roll into user creation here bc otherwise I will just be rewriting this code
	name, email := "Unit Test", "UnitTest@mail.com"
	userUUID, err := createUser(acoUUID, name, email)
	assert.NotNil(s.T(), userUUID)
	assert.Nil(s.T(), err)
	var testUser auth.User
	db.First(&testUser, "Email=?", email)
	assert.Equal(s.T(), testUser.UUID.String(), userUUID)

	// We have a user and an ACO, time for a token
	accessTokenString, err := createAccessToken(acoUUID, userUUID)
	assert.NotNil(s.T(), accessTokenString)
	assert.Nil(s.T(), err)

	// Bad/Negative tests

	// No ACO Name
	badACOName := ""
	badACO, err := createACO(badACOName)
	assert.Equal(s.T(), badACO, "")
	assert.NotNil(s.T(), err)

	// Blank UUID
	badUserUUID, err := createUser("", name, email)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank UUID
	badUserUUID, err = createUser(BADUUID, name, email)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank Name
	badUserUUID, err = createUser(acoUUID, "", email)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank E-mail address
	badUserUUID, err = createUser(acoUUID, name, "")
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badUserUUID)

	// Blank ACO UUID
	badAccessTokenString, err := createAccessToken("", userUUID)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)

	// Bad ACO UUID
	badAccessTokenString, err = createAccessToken(BADUUID, userUUID)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)

	// Blank User UUID
	badAccessTokenString, err = createAccessToken(acoUUID, "")
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)

	// Bad User UUID
	badAccessTokenString, err = createAccessToken(acoUUID, BADUUID)
	assert.NotNil(s.T(), err)
	assert.Equal(s.T(), "", badAccessTokenString)
}

func (s *MainTestSuite) TestCreateUser() {

}

const TOKENHEADER string = "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9."

func checkTokenInfo(s *MainTestSuite, tokenInfo string, ttl string) {
	assert.NotNil(s.T(), tokenInfo)
	lines := strings.Split(tokenInfo, "\n")
	assert.Equal(s.T(), 3, len(lines))
	expDate, err := time.Parse(time.RFC850, lines[0])
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), expDate)
	assert.Regexp(s.T(), "[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}", lines[1], "no correctly formatted token id in second line %s", lines[1])
	assert.True(s.T(), strings.HasPrefix(lines[2], TOKENHEADER), "incorrect token header %s", lines[2])
	assert.InDelta(s.T(), 500, len(tokenInfo), 100, "encoded token string length should be 500+-100; it is %d\n%s", len(tokenInfo), lines[2])
}

func (s *MainTestSuite) TestCreateAlphaToken() {

	alphaTokenInfo, err := createAlphaToken("")
	assert.Nil(s.T(), err)
	checkTokenInfo(s, alphaTokenInfo, "0")

	anotherTokenInfo, err := createAlphaToken("720")
	assert.Nil(s.T(), err)
	checkTokenInfo(s, anotherTokenInfo, "720")

	l1 := strings.Split(alphaTokenInfo, "\n")
	l2 := strings.Split(anotherTokenInfo, "\n")
	assert.NotEqual(s.T(), l1[0], l2[0], "alpha expiration dates should be different (%s == %s)", l1[0], l2[0])
	assert.NotEqual(s.T(), l1[1], l2[1], "alpha token uuids should be different (%s == %s)", l1[1], l2[1])
}

func (s *MainTestSuite) TestArchiveExpiring() {
	autoMigrate()
	db := database.GetGORMDbConnection()

	// save a job to our db
	j := models.Job{
		AcoID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
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

	archiveExpiring(0)

	//check that the file has moved to the archive location
	expPath := fmt.Sprintf("%s/%d/fake.ndjson", os.Getenv("FHIR_ARCHIVE_DIR"), id)
	_, err = ioutil.ReadFile(expPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(s.T(), expPath, "File not Found")

	var testjob models.Job
	db.First(&testjob, "id = ?", j.ID)

	//check the status of the job
	assert.Equal(s.T(), "Archived", testjob.Status)

	// clean up
	os.RemoveAll(os.Getenv("FHIR_ARCHIVE_DIR"))
}

func (s *MainTestSuite) TestArchiveExpiringWithThreshold() {
	autoMigrate()
	db := database.GetGORMDbConnection()

	// save a job to our db
	j := models.Job{
		AcoID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
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

	archiveExpiring(1)

	//check that the file has not moved to the archive location
	dataPath := fmt.Sprintf("%s/%d/fake.ndjson", os.Getenv("FHIR_PAYLOAD_DIR"), id)
	_, err = ioutil.ReadFile(dataPath)
	if err != nil {
		s.T().Error(err)
	}
	assert.FileExists(s.T(), dataPath, "File not Found")

	var testjob models.Job
	db.First(&testjob, "id = ?", j.ID)

	//check the status of the job
	assert.Equal(s.T(), "Completed", testjob.Status)

	// clean up
	os.Remove(dataPath)
}
