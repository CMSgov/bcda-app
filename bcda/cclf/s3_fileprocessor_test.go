package cclf

import (
	"context"
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
	cclfRefDate string
	basePath    string
	processor   CclfFileProcessor
}

func (s *S3ProcessorTestSuite) SetupSuite() {
	s.cclfRefDate = conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201") // Needed to allow our static CCLF files to continue to be processed

	s.basePath = "../../shared_files"
	s.processor = &S3FileProcessor{
		Handler: optout.S3FileHandler{
			Logger:   logrus.StandardLogger(),
			Endpoint: conf.GetEnv("BFD_S3_ENDPOINT"),
		},
	}
}

func (s *S3ProcessorTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.cclfRefDate)
}

func (s *S3ProcessorTestSuite) TestLoadCclfFiles() {
	cmsID := "A0001"
	tests := []struct {
		path            string
		numCCLFZipFiles int
		skipped         int
		failure         int
	}{
		{"cclf/archives/valid/", 1, 0, 0},
		{"cclf/mixed/with_invalid_filenames/", 1, 0, 0},
		{"cclf/mixed/0/valid_names/", 0, 0, 3},
		{"cclf/archives/8/valid/", 0, 0, 5},
		{"cclf/files/9/valid_names/", 0, 0, 0},
		{"cclf/mixed/with_folders/", 1, 0, 0},
	}

	for _, tt := range tests {
		s.T().Run(tt.path, func(t *testing.T) {
			bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, tt.path))
			defer cleanup()

			cclfMap, skipped, failure, err := s.processor.LoadCclfFiles(filepath.Join(bucketName, tt.path))
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

func (s *S3ProcessorTestSuite) TestLoadCclfFiles_SkipOtherEnvs() {
	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{{Name: "ENV", Value: "someenv"}})
	defer cleanupEnvVars()

	s3Bucket, cleanupS3 := testUtils.CreateZipsInS3(s.T(), testUtils.ZipInput{ZipName: "blah/not-someenv/T.BCD.A0001.ZCY18.D181120.T1000000", CclfNames: []string{"test", "zip"}})
	defer cleanupS3()

	cclfMap, skipped, failure, err := s.processor.LoadCclfFiles(s3Bucket)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Empty(s.T(), cclfMap)
}

func (s *S3ProcessorTestSuite) TestLoadCclfFiles_SingleFile() {
	cmsID := "A0001"
	tests := []struct {
		path            string
		filename        string
		numCCLFZipFiles int // Expected count for the cmsID, perfYear above
		skipped         int
		failure         int
	}{
		{"cclf/archives/valid/", "T.BCD.A0001.ZCY18.D181120.T1000000", 1, 0, 0},
	}

	for _, tt := range tests {
		s.T().Run(tt.path, func(t *testing.T) {
			bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, tt.path))
			defer cleanup()

			cclfMap, skipped, failure, err := s.processor.LoadCclfFiles(filepath.Join(bucketName, tt.path, tt.filename))
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

func (s *S3ProcessorTestSuite) TestLoadCclfFiles_InvalidPath() {
	cclfMap, skipped, failure, err := s.processor.LoadCclfFiles("foo")
	assert.ErrorContains(s.T(), err, "NoSuchBucket: The specified bucket does not exist")
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Empty(s.T(), cclfMap)
}

func TestS3ProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(S3ProcessorTestSuite))
}

func (s *S3ProcessorTestSuite) TestMultipleFileTypes() {
	// Hard code the reference date to ensure we do not reject any CCLF files because they are too old.
	cclfRefDate := conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "201201")
	defer conf.SetEnv(s.T(), "CCLF_REF_DATE", cclfRefDate)

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
			ZipName:   "T.BCD.A9990.ZCR19.D201113.T0000000",
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
	}
}

func (s *S3ProcessorTestSuite) TestCleanupCCLF() {
	assert := assert.New(s.T())
	cclfmap := make(map[string][]*cclfZipMetadata)
	acoID := "A0001"

	bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, constants.CCLF8CompPath))
	defer cleanup()

	// failed import: stay put
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
			filePath:      filepath.Join(bucketName, constants.CCLF8CompPath),
			imported:      false,
		},
	}

	deletedCount, err := s.processor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(0, deletedCount)
	assert.Nil(err)

	// Cleanup file after import
	cclfmap[acoID][0].imported = true

	deletedCount, err = s.processor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(1, deletedCount)
	assert.Nil(err)
}
