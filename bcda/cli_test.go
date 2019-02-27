package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

type CLITestSuite struct {
	suite.Suite
	testApp *cli.App
}

func (s *CLITestSuite) SetupTest() {
	s.testApp = setUpApp()
}

func TestCLITestSuite(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
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

	expTime, _ = time.Parse(time.RFC3339, "2019-01-08T23:55:00Z")
	metadata, err = getCCLFFileMetadata("/path/P.A0001.ACO.ZC9Y18.D190108.T2355000")
	assert.Equal("production", metadata.env)
	assert.Equal("A0001", metadata.acoID)
	assert.Equal(9, metadata.cclfNum)
	assert.Equal(expTime, metadata.timestamp)
}
