package cclf

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
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

	"github.com/ccoveille/go-safecast"
	"github.com/stretchr/testify/assert"
)

type LocalFileProcessorTestSuite struct {
	suite.Suite
	cclfRefDate        string
	pendingDeletionDir string

	cclfProcessor CclfFileProcessor
	csvProcessor  CSVFileProcessor
	basePath      string
	cleanup       func()
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

	var hours, e = safecast.ToUint(utils.GetEnvInt("FILE_ARCHIVE_THRESHOLD_HR", 72))
	if e != nil {
		fmt.Println("Error converting FILE_ARCHIVE_THRESHOLD_HR to uint", e)
	}

	s.cclfProcessor = &LocalFileProcessor{
		Handler: optout.LocalFileHandler{
			Logger:                 log.API,
			PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
			FileArchiveThresholdHr: hours,
		},
	}
	s.pendingDeletionDir = dir
	s.csvProcessor = &LocalFileProcessor{
		Handler: optout.LocalFileHandler{
			Logger:                 log.API,
			PendingDeletionDir:     conf.GetEnv("PENDING_DELETION_DIR"),
			FileArchiveThresholdHr: hours,
		},
	}
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
		path            string
		numCCLFZipFiles int // Expected count for the cmsID, perfYear above
		skipped         int
		failure         int
	}{
		{filepath.Join(s.basePath, "cclf/archives/valid/"), 1, 0, 0},
		{filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/"), 1, 0, 0},
		{filepath.Join(s.basePath, "cclf/mixed/0/valid_names/"), 0, 0, 3}, // properly named archives with only CCLF0 files
		{filepath.Join(s.basePath, "cclf/archives/8/valid/"), 0, 0, 5},    // properly named archives with only CCLF8 files
		{filepath.Join(s.basePath, "cclf/files/9/valid_names/"), 0, 0, 0},
		{filepath.Join(s.basePath, "cclf/mixed/with_folders/"), 1, 0, 0},
	}

	for _, tt := range tests {
		s.T().Run(tt.path, func(t *testing.T) {
			cclfMap, skipped, failure, err := processCCLFArchives(tt.path)
			cclfZipFiles := cclfMap[cmsID]
			assert.NoError(t, err)
			assert.Equal(t, tt.skipped, skipped)
			assert.Equal(t, tt.failure, failure)
			assert.Equal(t, tt.numCCLFZipFiles, len(cclfZipFiles))
			for _, cclfZipFile := range cclfZipFiles {
				assert.Equal(t, 18, cclfZipFile.cclf0Metadata.perfYear)
				assert.Equal(t, 18, cclfZipFile.cclf8Metadata.perfYear)
				assert.Equal(t, models.FileTypeDefault, cclfZipFile.cclf0Metadata.fileType)
				assert.Equal(t, models.FileTypeDefault, cclfZipFile.cclf8Metadata.fileType)
			}
		})
	}
}

func (s *LocalFileProcessorTestSuite) TestProcessCCLFArchives_ExpireFiles() {
	cmsID := "A0001"

	assert := assert.New(s.T())
	folderPath := filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/")
	filePath := filepath.Join(folderPath, "T.BCDE.ACO.ZC0Y18.D181120.T0001000")

	// Update file timestamp to now
	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err.Error())
	}

	cclfMap, skipped, failure, err := processCCLFArchives(folderPath)
	assert.Nil(err)

	cclfList := cclfMap[cmsID]
	assert.Equal(1, len(cclfList))
	assert.Equal(0, skipped)
	assert.Equal(0, failure)

	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	// Update file timestamp to 73 hours ago
	timeChange := origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)
	if err != nil {
		s.FailNow(constants.TestChangeTimeErr, err)
	}

	cclfMap, skipped, failure, err = processCCLFArchives(folderPath)
	assert.Nil(err)

	cclfList = cclfMap[cmsID]
	assert.Equal(1, len(cclfList))
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

func (s *LocalFileProcessorTestSuite) TestProcessCCLFArchives_DuplicateCCLFs() {
	dir, err := os.MkdirTemp("", "*")
	assert.NoError(s.T(), err)

	// Hard code the reference date to ensure we do not reject any CCLF files because they are too old.
	cclfRefDate := conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "201201")
	defer conf.SetEnv(s.T(), "CCLF_REF_DATE", cclfRefDate)
	defer os.RemoveAll(dir)

	// Multiple CCLF0s
	createZip(s.T(), dir, "T.BCD.A9990.ZCY20.D201113.T0000000", "T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC0Y20.D201113.T0000011", "T.BCD.A9990.ZC8Y20.D201113.T0000010")

	// Multiple CCLF8s
	createZip(s.T(), dir, "T.BCD.A9990.ZCY20.D201114.T0000000", "T.BCD.A9990.ZC0Y20.D201114.T0000010", "T.BCD.A9990.ZC8Y20.D201114.T0000010", "T.BCD.A9990.ZC8Y20.D201114.T0000011")

	m, skipped, f, err := processCCLFArchives(dir)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 2, f)
	assert.Empty(s.T(), m)
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

	for _, zipFiles := range m {
		// We should contain 4 unique entries, one for each unique perfYear:fileType tuple
		assert.Equal(s.T(), 4, len(zipFiles))
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
	cclfmap := make(map[string][]*cclfZipMetadata)
	acoID := "A0001"

	// failed import: file delivered recently -- stay put
	fileTime, _ := time.Parse(time.RFC3339, constants.TestFileTime)
	cclf0metadata := &cclfFileMetadata{
		name:         "T.BCD.ACO.ZC0Y18.D181120.T0001000",
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		deliveryDate: time.Now(),
	}

	fileTime, _ = time.Parse(time.RFC3339, constants.TestFileTime)
	cclf8metadata := &cclfFileMetadata{
		name:         constants.CCLF8Name,
		env:          "test",
		acoID:        acoID,
		cclfNum:      8,
		perfYear:     18,
		timestamp:    fileTime,
		deliveryDate: time.Now(),
	}

	cclfmap[acoID] = []*cclfZipMetadata{
		{
			cclf0Metadata: *cclf0metadata,
			cclf8Metadata: *cclf8metadata,
			filePath:      filepath.Join(s.basePath, constants.CCLF8CompPath),
			imported:      false,
		},
	}

	deletedCount, err := s.cclfProcessor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(0, deletedCount)
	assert.Nil(err)

	// failed import: file delivery expired - should move
	cclfmap[acoID][0].cclf0Metadata.deliveryDate = fileTime
	cclfmap[acoID][0].cclf8Metadata.deliveryDate = fileTime

	deletedCount, err = s.cclfProcessor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(1, deletedCount)
	assert.Nil(err)

	//negative cases

	// File unable to be renamed after import (file does not exist)
	cclfmap[acoID][0].filePath = filepath.Join(s.basePath, "cclf/archives/valid/T.BCD.A0001.ZCY18.D181122.Z1000000")
	cclfmap[acoID][0].imported = true

	deletedCount, err = s.cclfProcessor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(0, deletedCount)
	assert.EqualError(err, "1 files could not be cleaned up")

	// File unable to be renamed after failed import (file does not exist)
	cclfmap[acoID][0].imported = false

	deletedCount, err = s.cclfProcessor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(0, deletedCount)
	assert.EqualError(err, "1 files could not be cleaned up")

	files, err := os.ReadDir(conf.GetEnv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", conf.GetEnv("PENDING_DELETION_DIR"), err)
	}
	for _, file := range files {
		assert.NotEqual("T.BCD.ACO.ZC0Y18.D181120.T0001000", file.Name())
	}
}

func (s *LocalFileProcessorTestSuite) TestLoadCSV() {
	tests := []struct {
		name string
		file string
		err  error
	}{
		{"Valid file", "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000", nil},
		{"Opt-out", "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", errors.New("error")},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(tt *testing.T) {
			file := filepath.Join(s.basePath, test.file)
			e, _, err := s.csvProcessor.LoadCSV(file)
			if test.err != nil {
				assert.Nil(s.T(), e)
				assert.NotNil(s.T(), err)
			} else {
				assert.NotEmpty(s.T(), e)
				assert.Nil(s.T(), err)
			}

		})
	}
}

func (s *LocalFileProcessorTestSuite) TestCleanUpCSV() {
	expiredTime, _ := time.Parse(time.RFC3339, constants.TestFileTime)
	file := csvFile{
		metadata: csvFileMetadata{
			name:      "P.PCPB.M2411.D181120.T1000000",
			env:       "test",
			acoID:     "FOOACO",
			cclfNum:   8,
			perfYear:  24,
			timestamp: time.Now(),
			fileID:    0,
			fileType:  1,
		},
		data: bytes.NewReader([]byte("MBIS\nMBI000001\nMBI000002\nMBI000003")),
	}

	tests := []struct {
		name         string
		filepath     string
		deliverytime time.Time
		imported     bool
		delFiles     int
		baseFiles    int
	}{
		{"Not imported and expired", filepath.Join(s.basePath, "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"), expiredTime, false, 1, 0},
		{"Not imported and not expired", filepath.Join(s.basePath, "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"), time.Now(), false, 1, 0},
		{"Successfully imported", filepath.Join(s.basePath, "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"), time.Now(), true, 1, 0},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(tt *testing.T) {
			file.metadata.deliveryDate = test.deliverytime
			file.imported = test.imported
			file.filepath = test.filepath
			err := s.csvProcessor.CleanUpCSV(file)
			assert.Nil(s.T(), err)
			delDir, err := os.ReadDir(conf.GetEnv("PENDING_DELETION_DIR"))
			if err != nil {
				s.FailNow("failed to read directory: %s", conf.GetEnv("PENDING_DELETION_DIR"), err)
			}
			assert.Len(s.T(), delDir, test.delFiles)
			if test.delFiles == 1 {
				assert.Equal(s.T(), file.metadata.name, delDir[0].Name())
			}
			baseDir, err := os.ReadDir(filepath.Join(s.basePath, "cclf/archives/csv"))
			if err != nil {
				s.FailNow("failed to read directory: %s", conf.GetEnv("PENDING_DELETION_DIR"), err)
			}
			assert.Len(s.T(), baseDir, test.baseFiles)
			if test.baseFiles == 1 {
				assert.Equal(s.T(), file.metadata.name, baseDir[0].Name())
			}
		})
	}

}
