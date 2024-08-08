package cclf

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/stretchr/testify/assert"
)

type LocalFileProcessorTestSuite struct {
	suite.Suite
	cclfRefDate        string
	pendingDeletionDir string

	processor CclfFileProcessor
	basePath  string
	cleanup   func()
}

func (s *LocalFileProcessorTestSuite) SetupTest() {
	s.basePath, s.cleanup = testUtils.CopyToTemporaryDirectory(s.T(), "../../shared_files/")
}

func (s *LocalFileProcessorTestSuite) SetupSuite() {
	s.cclfRefDate = conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201") // Needed to allow our static CCLF files to continue to be processed
	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		logrus.Fatal(err)
	}

	s.processor = &LocalFileProcessor{
		Handler: optout.LocalFileHandler{
			Logger:                 log.API,
			PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
			FileArchiveThresholdHr: uint(utils.GetEnvInt("FILE_ARCHIVE_THRESHOLD_HR", 72)),
		},
	}
	s.pendingDeletionDir = dir
	testUtils.SetPendingDeletionDir(s.Suite, dir)
}

func (s *LocalFileProcessorTestSuite) TearDownTest() {
	s.cleanup()
}

func (s *LocalFileProcessorTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.cclfRefDate)
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *LocalFileProcessorTestSuite) TestProcessCCLFArchives() {
	cmsID := "A0001"
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
		{filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/"), 1, 0, 0, 1, 1},
		{filepath.Join(s.basePath, "cclf/mixed/0/valid_names/"), 3, 0, 0, 3, 0},
		{filepath.Join(s.basePath, "cclf/archives/8/valid/"), 5, 0, 0, 0, 5},
		{filepath.Join(s.basePath, "cclf/files/9/valid_names/"), 0, 0, 0, 0, 0},
		{filepath.Join(s.basePath, "cclf/mixed/with_folders/"), 2, 0, 0, 1, 1},
	}

	for _, tt := range tests {
		s.T().Run(tt.path, func(t *testing.T) {
			cclfMap, skipped, failure, err := processCCLFArchives(tt.path)
			cclfZipFiles := cclfMap[cmsID]
			assert.NoError(t, err)
			assert.Equal(t, tt.skipped, skipped)
			assert.Equal(t, tt.failure, failure)
			assert.Equal(t, tt.numCCLFFiles, len(cclfZipFiles))
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

func (s *LocalFileProcessorTestSuite) TestProcessCCLFArchives_ExpireFiles() {
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

func (s *LocalFileProcessorTestSuite) TestProcessCCLFArchives_InvalidPath() {
	cclfMap, skipped, failure, err := processCCLFArchives("./foo")
	assert.EqualError(s.T(), err, "error in sorting cclf file: nil,: lstat ./foo: no such file or directory")
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Nil(s.T(), cclfMap)
}

func (s *LocalFileProcessorTestSuite) TestProcessCCLFArchives_Downloading() {
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

func (s *LocalFileProcessorTestSuite) TestProcessCCLFArchives_CorruptedFile() {
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

func TestLocalFileProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(LocalFileProcessorTestSuite))
}

func (s *LocalFileProcessorTestSuite) TestStillDownloading() {
	secondsAgo := time.Now().Add(time.Duration(-30) * time.Second)
	minutesAgo := time.Now().Add(time.Duration(-2) * time.Minute)

	assert.True(s.T(), stillDownloading(secondsAgo))
	assert.False(s.T(), stillDownloading(minutesAgo))
}

func (s *LocalFileProcessorTestSuite) TestMultipleFileTypes() {
	dir, err := os.MkdirTemp("", "*")
	assert.NoError(s.T(), err)
	// Hard code the reference date to ensure we do not reject any CCLF files because they are too old.
	cclfRefDate := conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "201201")
	defer conf.SetEnv(s.T(), "CCLF_REF_DATE", cclfRefDate)
	defer os.RemoveAll(dir)

	// Create various CCLF files that have unique perfYear:fileType
	createZip(s.T(), dir, "T.BCD.A9990.ZCY20.D201113.T0000000", "T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC8Y20.D201113.T0000010")
	// different perf year
	createZip(s.T(), dir, "T.BCD.A9990.ZCY19.D201113.T0000000", "T.BCD.A9990.ZC0Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000010")
	// different file type
	createZip(s.T(), dir, "T.BCD.A9990.ZCR20.D201113.T0000000", "T.BCD.A9990.ZC0R20.D201113.T0000010", "T.BCD.A9990.ZC8R20.D201113.T0000010")
	// different perf year and file type
	createZip(s.T(), dir, "T.BCD.A9990.ZCR19.D201113.T0000000", "T.BCD.A9990.ZC0R19.D201113.T0000010", "T.BCD.A9990.ZC8R19.D201113.T0000010")

	m, skipped, f, err := processCCLFArchives(dir)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, f)
	assert.Equal(s.T(), 1, len(m)) // Only one ACO present

	for _, fileMap := range m {
		// We should contain 4 unique entries, one for each unique perfYear:fileType tuple
		assert.Equal(s.T(), 4, len(fileMap))
		for _, files := range fileMap {
			assert.Equal(s.T(), 2, len(files)) // each tuple contains two files
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

func (s *LocalFileProcessorTestSuite) TestCleanupCCLF() {
	assert := assert.New(s.T())
	cclfmap := make(map[string]map[metadataKey][]*cclfFileMetadata)

	// failed import: file that's within the threshold - stay put
	acoID := "A0001"
	fileTime, _ := time.Parse(time.RFC3339, constants.TestFileTime)
	cclf0metadata := &cclfFileMetadata{
		name:         "T.BCD.ACO.ZC0Y18.D181120.T0001000",
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, "cclf/T.BCD.ACO.ZC0Y18.D181120.T0001000"),
		imported:     false,
		deliveryDate: time.Now(),
	}

	// failed import: file that's over the threshold - should move
	fileTime, _ = time.Parse(time.RFC3339, constants.TestFileTime)
	cclf8metadata := &cclfFileMetadata{
		name:         constants.CCLF8Name,
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		filePath:     filepath.Join(s.basePath, constants.CCLF8CompPath),
		imported:     false,
		deliveryDate: fileTime,
	}

	// successfully imported file - should move
	fileTime, _ = time.Parse(time.RFC3339, constants.TestFileTime)
	cclf9metadata := &cclfFileMetadata{
		name:      "T.BCD.A0001.ZC9Y18.D181120.T1000010",
		env:       "test",
		acoID:     acoID,
		cclfNum:   9,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181122.T1000000"),
		imported:  true,
	}

	cclfmap["A0001"] = map[metadataKey][]*cclfFileMetadata{
		{perfYear: 18, fileType: models.FileTypeDefault}: {cclf0metadata, cclf8metadata, cclf9metadata},
	}
	err := s.processor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Nil(err)

	//negative cases
	//File unable to be renamed
	fileTime, _ = time.Parse(time.RFC3339, constants.TestFileTime)
	cclf10metadata := &cclfFileMetadata{
		name:      "T.BCD.A0001.ZC9Y18.D181120.T1000010",
		env:       "test",
		acoID:     acoID,
		cclfNum:   10,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181122.Z1000000"),
		imported:  true,
	}
	//unsuccessful, not imported
	fileTime, _ = time.Parse(time.RFC3339, constants.TestFileTime)
	cclf11metadata := &cclfFileMetadata{
		name:      "T.BCD.A0001.ZC9Y18.D181120.T1000010",
		env:       "test",
		acoID:     acoID,
		cclfNum:   10,
		perfYear:  18,
		timestamp: fileTime,
		filePath:  filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181122.Z1000000"),
		imported:  false,
	}
	cclfmap["A0001"] = map[metadataKey][]*cclfFileMetadata{
		{perfYear: 18, fileType: models.FileTypeDefault}: {cclf10metadata, cclf11metadata},
	}
	err = s.processor.CleanUpCCLF(context.Background(), cclfmap)
	assert.EqualError(err, "2 files could not be cleaned up")

	files, err := os.ReadDir(conf.GetEnv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", conf.GetEnv("PENDING_DELETION_DIR"), err)
	}
	for _, file := range files {
		assert.NotEqual("T.BCD.ACO.ZC0Y18.D181120.T0001000", file.Name())
	}
}
