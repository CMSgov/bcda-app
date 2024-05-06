package cclf

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type FileProcessorTestSuite struct {
	suite.Suite
	cclfRefDate        string
	pendingDeletionDir string

	basePath string
	cleanup  func()
}

func (s *FileProcessorTestSuite) SetupTest() {
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")
}

func (s *FileProcessorTestSuite) SetupSuite() {
	s.cclfRefDate = conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201") // Needed to allow our static CCLF files to continue to be processed
	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		log.Fatal(err)
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)
}

func (s *FileProcessorTestSuite) TearDownTest() {
	s.cleanup()
}

func (s *FileProcessorTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.cclfRefDate)
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives() {
	cmsID, key := "A0001", metadataKey{perfYear: 18, fileType: models.FileTypeDefault}
	tests := []struct {
		path         string
		numCCLFFiles int // Expected count for the cmsID, perfYear above
		skipped      int
		failure      int
		numCCLF0     int // Expected count for the cmsID, perfYear above
		numCCLF8     int // Expected count for the cmsID, perfYear above
	}{
		{filepath.Join(s.basePath, "cclf/archives/valid/"), 2, 0, 0, 1, 1},
		{filepath.Join(s.basePath, "cclf/archives/bcd/"), 2, 0, 0, 1, 1},
		{filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/"), 2, 0, 0, 1, 1},
		{filepath.Join(s.basePath, "cclf/mixed/0/valid_names/"), 3, 0, 0, 3, 0},
		{filepath.Join(s.basePath, "cclf/archives/8/valid/"), 5, 0, 0, 0, 5},
		{filepath.Join(s.basePath, "cclf/files/9/valid_names/"), 0, 0, 0, 0, 0},
		{filepath.Join(s.basePath, "cclf/mixed/with_folders/"), 2, 0, 0, 1, 1},
	}

	for _, tt := range tests {
		s.T().Run(tt.path, func(t *testing.T) {
			cclfMap, skipped, failure, err := processCCLFArchives(tt.path)
			cclfFiles := cclfMap[cmsID][key]
			assert.NoError(t, err)
			assert.Equal(t, tt.skipped, skipped)
			assert.Equal(t, tt.failure, failure)
			assert.Equal(t, tt.numCCLFFiles, len(cclfFiles))
			var numCCLF0, numCCLF8 int
			for _, cclfFile := range cclfFiles {
				if cclfFile.cclfNum == 0 {
					numCCLF0++
				} else if cclfFile.cclfNum == 8 {
					numCCLF8++
				} else {
					assert.Fail(t, "Unexpected CCLF num received %d", cclfFile.cclfNum)
				}
			}
			assert.Equal(t, tt.numCCLF0, numCCLF0)
			assert.Equal(t, tt.numCCLF8, numCCLF8)
		})
	}
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives_ExpireFiles() {
	assert := assert.New(s.T())
	key := metadataKey{perfYear: 18, fileType: models.FileTypeDefault}
	folderPath := filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/")
	filePath := filepath.Join(folderPath, "T.BCDE.ACO.ZC0Y18.D181120.T0001000")

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err.Error())
	}

	cclfMap, skipped, failure, err := processCCLFArchives(folderPath)
	assert.Nil(err)
	cclfList := cclfMap["A0001"][key]
	assert.Equal(2, len(cclfList))
	assert.Equal(0, skipped)
	assert.Equal(0, failure)
	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	timeChange := origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)
	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err)
	}

	cclfMap, skipped, failure, err = processCCLFArchives(folderPath)
	assert.Nil(err)
	cclfList = cclfMap["A0001"][key]
	assert.Equal(2, len(cclfList))
	assert.Equal(0, skipped)
	assert.Equal(0, failure)

	// assert that this file is not still here.
	_, err = os.Open(filePath)
	assert.EqualError(err, fmt.Sprintf("open %s: no such file or directory", filePath))
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives_InvalidPath() {
	cclfMap, skipped, failure, err := processCCLFArchives("./foo")
	assert.EqualError(s.T(), err, "error in sorting cclf file: nil,: lstat ./foo: no such file or directory")
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Nil(s.T(), cclfMap)
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives_Downloading() {
	assert := assert.New(s.T())
	folderPath := filepath.Join(s.basePath, "cclf/archives/corrupted/")
	filePath := filepath.Join(folderPath, "T.BCD.A0001.ZCY18.D181120.T1000000")
	secondsAgo := time.Now().Add(time.Duration(-30) * time.Second)

	err := os.Chtimes(filePath, secondsAgo, secondsAgo)

	if err == nil {
		cclfMap, skipped, failure, err := processCCLFArchives(filePath)
		assert.Nil(err)
		assert.Equal(0, skipped)
		assert.Equal(0, failure)
		assert.Empty(cclfMap)
	}
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives_CorruptedFile() {
	assert := assert.New(s.T())
	folderPath := filepath.Join(s.basePath, "cclf/archives/corrupted/")
	filePath := filepath.Join(folderPath, "T.BCD.A0001.ZCY18.D181120.T1000000")
	yesterday := time.Now().AddDate(0, 0, -1)
	err := os.Chtimes(filePath, yesterday, yesterday)

	if err == nil {
		cclfMap, skipped, failure, err := processCCLFArchives(filePath)
		assert.Nil(err)
		assert.Equal(0, skipped)
		assert.Equal(1, failure)
		assert.NotNil(cclfMap)
	}
}

func TestFileProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(FileProcessorTestSuite))
}

func TestStillDownloading(t *testing.T) {
	secondsAgo := time.Now().Add(time.Duration(-30) * time.Second)
	minutesAgo := time.Now().Add(time.Duration(-2) * time.Minute)

	assert.True(t, stillDownloading(secondsAgo))
	assert.False(t, stillDownloading(minutesAgo))
}

func TestMultipleFileTypes(t *testing.T) {
	dir, err := os.MkdirTemp("", "*")
	assert.NoError(t, err)
	// Hard code the reference date to ensure we do not reject any CCLF files because they are too old.
	cclfRefDate := conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(t, "CCLF_REF_DATE", "201201")
	defer conf.SetEnv(t, "CCLF_REF_DATE", cclfRefDate)
	defer os.RemoveAll(dir)

	// Create various CCLF files that have unique perfYear:fileType
	createZip(t, dir, "T.BCD.A9990.ZCY20.D201113.T0000000", "T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC8Y20.D201113.T0000010")
	// different perf year
	createZip(t, dir, "T.BCD.A9990.ZCY19.D201113.T0000000", "T.BCD.A9990.ZC0Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000010")
	// different file type
	createZip(t, dir, "T.BCD.A9990.ZCR20.D201113.T0000000", "T.BCD.A9990.ZC0R20.D201113.T0000010", "T.BCD.A9990.ZC8R20.D201113.T0000010")
	// different perf year and file type
	createZip(t, dir, "T.BCD.A9990.ZCR19.D201113.T0000000", "T.BCD.A9990.ZC0R19.D201113.T0000010", "T.BCD.A9990.ZC8R19.D201113.T0000010")

	m, s, f, err := processCCLFArchives(dir)
	assert.NoError(t, err)
	assert.Equal(t, 0, s)
	assert.Equal(t, 0, f)
	assert.Equal(t, 1, len(m)) // Only one ACO present

	for _, fileMap := range m {
		// We should contain 4 unique entries, one for each unique perfYear:fileType tuple
		assert.Equal(t, 4, len(fileMap))
		for _, files := range fileMap {
			assert.Equal(t, 2, len(files)) // each tuple contains two files
		}
	}
}

func createZip(t *testing.T, dir, zipName string, cclfNames ...string) {
	name := filepath.Join(dir, zipName)
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	assert.NoError(t, err)
	defer f.Close()

	w := zip.NewWriter(f)
	for _, cclfName := range cclfNames {
		_, err := w.Create(cclfName)
		assert.NoError(t, err)
	}

	assert.NoError(t, w.Close())
}
