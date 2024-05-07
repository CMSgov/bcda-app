package cclf

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/stretchr/testify/assert"
)

type S3ProcessorTestSuite struct {
	suite.Suite
	basePath  string
	processor CclfFileProcessor
}

func (s *S3ProcessorTestSuite) SetupSuite() {
	s.basePath = "../../shared_files"
	s.processor = &S3FileProcessor{
		Handler: optout.S3FileHandler{
			Logger:   logrus.StandardLogger(),
			Endpoint: conf.GetEnv("BFD_S3_ENDPOINT"),
		},
	}
}

func (s *S3ProcessorTestSuite) TestLoadCclfFiles() {
	cmsID, key := "A0001", metadataKey{perfYear: 18, fileType: models.FileTypeDefault}
	tests := []struct {
		path         string
		numCCLFFiles int // Expected count for the cmsID, perfYear above
		skipped      int
		failure      int
		numCCLF0     int // Expected count for the cmsID, perfYear above
		numCCLF8     int // Expected count for the cmsID, perfYear above
	}{
		{"cclf/archives/valid/", 2, 0, 0, 1, 1},
		{"cclf/archives/bcd/", 2, 0, 0, 1, 1},
		{"cclf/mixed/with_invalid_filenames/", 2, 0, 0, 1, 1},
		{"cclf/mixed/0/valid_names/", 3, 0, 0, 3, 0},
		{"cclf/archives/8/valid/", 5, 0, 0, 0, 5},
		{"cclf/files/9/valid_names/", 0, 0, 0, 0, 0},
		{"cclf/mixed/with_folders/", 2, 0, 0, 1, 1},
	}

	for _, tt := range tests {
		s.T().Run(tt.path, func(t *testing.T) {
			bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, tt.path))
			defer cleanup()

			cclfMap, skipped, failure, err := s.processor.LoadCclfFiles(filepath.Join(bucketName, tt.path))
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

func (s *S3ProcessorTestSuite) TestLoadCclfFiles_InvalidPath() {
	cclfMap, skipped, failure, err := s.processor.LoadCclfFiles("foo")
	assert.EqualError(s.T(), err, "error in sorting cclf file: nil,: lstat ./foo: no such file or directory")
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Nil(s.T(), cclfMap)
}

func TestS3ProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(S3ProcessorTestSuite))
}

func (s *S3ProcessorTestSuite) TestMultipleFileTypes() {
	// Create various CCLF files that have unique perfYear:fileType
	bucketName, cleanup := testUtils.CreateZipsInS3(s.T(),
		testUtils.ZipInput{
			ZipName:   "T.BCD.A9990.ZCY20.D201113.T0000000",
			CclfNames: []string{"T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC8Y20.D201113.T0000010"},
		},
		// different perf year
		testUtils.ZipInput{
			ZipName:   "T.BCD.A9990.ZCY19.D201113.T0000000",
			CclfNames: []string{"T.BCD.A9990.ZC0Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000010"},
		},
		// different file type
		testUtils.ZipInput{
			ZipName:   "T.BCD.A9990.ZCR20.D201113.T0000000",
			CclfNames: []string{"T.BCD.A9990.ZC0R20.D201113.T0000010", "T.BCD.A9990.ZC8R20.D201113.T0000010"},
		},
		// different perf year and file type
		testUtils.ZipInput{
			ZipName:   "T.BCD.A9990.ZCR20.D201113.T0000000",
			CclfNames: []string{"T.BCD.A9990.ZC0R19.D201113.T0000010", "T.BCD.A9990.ZC8R19.D201113.T0000010"},
		},
	)

	defer cleanup()

	m, skipped, f, err := s.processor.LoadCclfFiles(bucketName)
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

func (s *CCLFTestSuite) TestCleanupCCLF() {
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
	err := s.importer.FileProcessor.CleanUpCCLF(context.Background(), cclfmap)
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
	err = s.importer.FileProcessor.CleanUpCCLF(context.Background(), cclfmap)
	assert.EqualError(err, "2 files could not be cleaned up")

	files, err := os.ReadDir(conf.GetEnv("PENDING_DELETION_DIR"))
	if err != nil {
		s.FailNow("failed to read directory: %s", conf.GetEnv("PENDING_DELETION_DIR"), err)
	}
	for _, file := range files {
		assert.NotEqual("T.BCD.ACO.ZC0Y18.D181120.T0001000", file.Name())
	}
}
