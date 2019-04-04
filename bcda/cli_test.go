package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

type CLITestSuite struct {
	testUtils.AuthTestSuite
	testApp *cli.App
}

func (s *CLITestSuite) SetupTest() {
	s.testApp = setUpApp()
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

func (s *CLITestSuite) TestCreateACO() {

	// init
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	s.SetupAuthBackend()

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
	acoUUID := strings.TrimSpace(buf.String())
	var testACO models.ACO
	db.First(&testACO, "Name=?", ACOName)
	assert.Equal(testACO.UUID.String(), acoUUID)
	buf.Reset()

	ACO2Name := "Unit Test ACO 2"
	aco2ID := "A9999"
	args = []string{"bcda", "create-aco", "--name", ACO2Name, "--cms-id", aco2ID}
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

func (s *CLITestSuite) TestImportCCLF8() {
	assert := assert.New(s.T())

	args := []string{"bcda", "import-cclf8"}
	err := s.testApp.Run(args)
	assert.EqualError(err, "file path (--file) must be provided")

	args = []string{"bcda", "import-cclf8", "--file", "../shared_files/cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid CCLF8 filename")

	args = []string{"bcda", "import-cclf8", "--file", "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D18NOV20.T1000010"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid filename")

	args = []string{"bcda", "import-cclf8", "--file", "../shared_files/cclf/T.ABCDE.ACO.ZC8Y18.D181120.T1000010"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid filename")

	args = []string{"bcda", "import-cclf8", "--file", "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"}
	err = s.testApp.Run(args)
	assert.Nil(err)
}

func (s *CLITestSuite) TestImportCCLF9() {
	assert := assert.New(s.T())

	args := []string{"bcda", "import-cclf9"}
	err := s.testApp.Run(args)
	assert.EqualError(err, "file path (--file) must be provided")

	args = []string{"bcda", "import-cclf9", "--file", "../shared_files/cclf/T.A0001.ACO.ZC8Y18.D181120.T1000009"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid CCLF9 filename")

	args = []string{"bcda", "import-cclf9", "--file", "../shared_files/cclf/T.A0001.ACO.ZC9Y18.D18NOV20.T1000009"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid filename")

	args = []string{"bcda", "import-cclf9", "--file", "../shared_files/cclf/T.ABCDE.ACO.ZC9Y18.D181120.T1000010"}
	err = s.testApp.Run(args)
	assert.EqualError(err, "invalid filename")

	args = []string{"bcda", "import-cclf9", "--file", "../shared_files/cclf/T.A0001.ACO.ZC9Y18.D181120.T1000010"}
	err = s.testApp.Run(args)
	assert.Nil(err)
}

func (s *CLITestSuite) TestGetCCLFFileMetadata() {
	assert := assert.New(s.T())

	_, err := getCCLFFileMetadata("/path/to/file")
	assert.EqualError(err, "invalid filename")

	metadata, err := getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T9909420")
	assert.EqualError(err, "failed to parse date 'D190117.T990942' from filename")

	expTime, _ := time.Parse(time.RFC3339, "2019-01-17T21:09:42Z")
	metadata, err = getCCLFFileMetadata("/path/T.A0000.ACO.ZC8Y18.D190117.T2109420")
	assert.Equal("test", metadata.env)
	assert.Equal("A0000", metadata.acoID)
	assert.Equal(8, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
	assert.Nil(err)

	expTime, _ = time.Parse(time.RFC3339, "2019-01-08T23:55:00Z")
	metadata, err = getCCLFFileMetadata("/path/P.A0001.ACO.ZC9Y18.D190108.T2355000")
	assert.Equal("production", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(9, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
	assert.Nil(err)
}
