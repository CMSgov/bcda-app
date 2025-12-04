package cclf

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
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
	cclfRefDate   string
	basePath      string
	cclfProcessor CclfFileProcessor
	csvProcessor  CSVFileProcessor
}

func (s *S3ProcessorTestSuite) SetupSuite() {
	s.cclfRefDate = conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "181201") // Needed to allow our static CCLF files to continue to be processed
	client := testUtils.TestS3Client(s.T(), testUtils.TestAWSConfig(s.T()))

	s.basePath = "../../shared_files"
	s.cclfProcessor = &S3FileProcessor{
		Handler: optout.S3FileHandler{
			Client:   client,
			Logger:   logrus.StandardLogger(),
			Endpoint: conf.GetEnv("BFD_S3_ENDPOINT"),
		},
	}
	s.csvProcessor = &S3FileProcessor{
		Handler: optout.S3FileHandler{
			Client:   client,
			Logger:   logrus.StandardLogger(),
			Endpoint: conf.GetEnv("BFD_S3_ENDPOINT"),
		},
	}
}

func (s *S3ProcessorTestSuite) TearDownSuite() {
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.cclfRefDate)
}

func (s *S3ProcessorTestSuite) TestLoadCclfFiles() {
	ctx := context.Background()
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

			cclfMap, skipped, failure, err := s.cclfProcessor.LoadCclfFiles(ctx, filepath.Join(bucketName, tt.path))
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
	ctx := context.Background()
	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{{Name: "ENV", Value: "dev"}})
	s.T().Cleanup(func() { cleanupEnvVars() })
	s3Bucket, cleanupS3 := testUtils.CreateZipsInS3(s.T(), testUtils.ZipInput{ZipName: "blah/not-dev/T.BCD.A0001.ZCY18.D181120.T1000000", CclfNames: []string{"T.BCD.A0001.ZC0Y18.D181120.T1000000", "T.BCD.A0001.ZC8Y18.D181120.T1000000"}})
	s.T().Cleanup(func() { cleanupS3() })

	cclfMap, skipped, failure, err := s.cclfProcessor.LoadCclfFiles(ctx, s3Bucket)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Empty(s.T(), cclfMap)
}

func (s *S3ProcessorTestSuite) TestLoadCclfFiles_DuplicateCCLFs() {
	ctx := context.Background()
	bucketName, cleanupS3 := testUtils.CreateZipsInS3(s.T(),
		// Multiple CCLF0s
		testUtils.ZipInput{
			ZipName:   "T.BCD.A9990.ZCY20.D201113.T0000000",
			CclfNames: []string{"T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC0Y20.D201113.T0000011", "T.BCD.A9990.ZC8Y20.D201113.T0000010"},
		},
		// Multiple CCLF8s
		testUtils.ZipInput{
			ZipName:   "T.BCD.A9990.ZCY19.D201113.T0000000",
			CclfNames: []string{"T.BCD.A9990.ZC0Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000011"},
		},
	)
	defer cleanupS3()

	cclfMap, skipped, failure, err := s.cclfProcessor.LoadCclfFiles(ctx, bucketName)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 2, failure)
	assert.Empty(s.T(), cclfMap)
}

func (s *S3ProcessorTestSuite) TestLoadCclfFiles_SingleFile() {
	ctx := context.Background()
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

			cclfMap, skipped, failure, err := s.cclfProcessor.LoadCclfFiles(ctx, filepath.Join(bucketName, tt.path, tt.filename))
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
	ctx := context.Background()
	cclfMap, skipped, failure, err := s.cclfProcessor.LoadCclfFiles(ctx, "foo")
	assert.ErrorContains(s.T(), err, "NoSuchBucket")
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Empty(s.T(), cclfMap)
}

func TestS3ProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(S3ProcessorTestSuite))
}

func (s *S3ProcessorTestSuite) TestMultipleFileTypes() {
	ctx := context.Background()
	// Hard code the reference date to ensure we do not reject any CCLF files because they are too old.
	cclfRefDate := conf.GetEnv("CCLF_REF_DATE")
	conf.SetEnv(s.T(), "CCLF_REF_DATE", "201201")
	s.T().Cleanup(func() { conf.SetEnv(s.T(), "CCLF_REF_DATE", cclfRefDate) })

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
	s.T().Cleanup(func() { cleanup() })

	m, skipped, f, err := s.cclfProcessor.LoadCclfFiles(ctx, bucketName)
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

	deletedCount, err := s.cclfProcessor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(0, deletedCount)
	assert.Nil(err)

	// Cleanup file after import
	cclfmap[acoID][0].imported = true

	deletedCount, err = s.cclfProcessor.CleanUpCCLF(context.Background(), cclfmap)
	assert.Equal(1, deletedCount)
	assert.Nil(err)
}

func (s *S3ProcessorTestSuite) TestCleanupCSV() {
	assert := assert.New(s.T())
	ctx := context.Background()
	path := "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"
	bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, path))

	tests := []struct {
		name     string
		filepath string
		imported bool
		err      error
	}{
		{"Clean up sucessful import", filepath.Join(bucketName, path), true, nil},
		{"Clean up failed import", filepath.Join(bucketName, path), false, nil},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(tt *testing.T) {
			defer cleanup()
			csv := csvFile{
				metadata: csvFileMetadata{},
				imported: test.imported,
				filepath: test.filepath,
			}
			err := s.csvProcessor.CleanUpCSV(ctx, csv)
			assert.Nil(err)

		})
	}

}

func (s *S3ProcessorTestSuite) TestLoadCSV() {
	assert := assert.New(s.T())
	ctx := context.Background()
	path := "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"

	bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, path))
	defer cleanup()

	tests := []struct {
		name     string
		filepath string
		err      error
	}{
		{"Load CSV sucessful", filepath.Join(bucketName, path), nil},
		{"Load CSV failed", "foo/bar", errors.New("S3 error")},
	}

	for _, test := range tests {
		s.T().Run(test.name, func(tt *testing.T) {
			defer cleanup()
			r, _, err := s.csvProcessor.LoadCSV(ctx, test.filepath)
			if test.err == nil {
				assert.Nil(err)
				assert.NotNil(r)
			} else {
				s.T().Log("FOO BAR")
				assert.NotNil(err)
				assert.Nil(r)
			}
		})
	}
}

func (s *S3ProcessorTestSuite) TestLoadCSV_InvalidPath() {
}

func (s *S3ProcessorTestSuite) TestLoadCSV_SkipOtherEnvs() {
	ctx := context.Background()
	cleanupEnvVars := testUtils.SetEnvVars(s.T(), []testUtils.EnvVar{{Name: "ENV", Value: "dev"}})
	defer cleanupEnvVars()

	path := "cclf/archives/csv/P.PCPB.M2411.D181120.T1000000"

	bucketName, cleanup := testUtils.CopyToS3(s.T(), filepath.Join(s.basePath, path))
	defer cleanup()
	_, _, err := s.csvProcessor.LoadCSV(ctx, filepath.Join(bucketName, path))
	assert.NotNil(s.T(), err)
	assert.Contains(s.T(), err.Error(), "Skipping import")

}
