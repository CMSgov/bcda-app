package cclf

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"

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
		Logger:   logrus.StandardLogger(),
		Endpoint: conf.GetEnv("BFD_S3_ENDPOINT"),
	}
}

func (s *S3ProcessorTestSuite) TestProcessCCLFArchives() {
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

func (s *S3ProcessorTestSuite) TestProcessCCLFArchives_InvalidPath() {
	cclfMap, skipped, failure, err := processCCLFArchives("./foo")
	assert.EqualError(s.T(), err, "error in sorting cclf file: nil,: lstat ./foo: no such file or directory")
	assert.Equal(s.T(), 0, skipped)
	assert.Equal(s.T(), 0, failure)
	assert.Nil(s.T(), cclfMap)
}

func (s *S3ProcessorTestSuite) TestS3ProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(S3ProcessorTestSuite))
}

type ZipInput struct {
	zipName   string
	cclfNames []string
}

func (s *S3ProcessorTestSuite) TestMultipleFileTypes(t *testing.T) {

	// Create various CCLF files that have unique perfYear:fileType
	bucketName, cleanup := createZipsInS3(t,
		ZipInput{
			"T.BCD.A9990.ZCY20.D201113.T0000000",
			[]string{"T.BCD.A9990.ZC0Y20.D201113.T0000010", "T.BCD.A9990.ZC8Y20.D201113.T0000010"},
		},
		// different perf year
		ZipInput{
			"T.BCD.A9990.ZCY19.D201113.T0000000",
			[]string{"T.BCD.A9990.ZC0Y19.D201113.T0000010", "T.BCD.A9990.ZC8Y19.D201113.T0000010"},
		},
		// different file type
		ZipInput{
			"T.BCD.A9990.ZCR20.D201113.T0000000",
			[]string{"T.BCD.A9990.ZC0R20.D201113.T0000010", "T.BCD.A9990.ZC8R20.D201113.T0000010"},
		},
		// different perf year and file type
		ZipInput{
			"T.BCD.A9990.ZCR20.D201113.T0000000",
			[]string{"T.BCD.A9990.ZC0R19.D201113.T0000010", "T.BCD.A9990.ZC8R19.D201113.T0000010"},
		},
	)

	defer cleanup()

	m, skipped, f, err := s.processor.LoadCclfFiles(bucketName)
	assert.NoError(t, err)
	assert.Equal(t, 0, skipped)
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

func createZipsInS3(t *testing.T, zipInputs ...ZipInput) (string, func()) {
	tempBucket, err := uuid.NewUUID()
	assert.NoError(t, err)

	endpoint := conf.GetEnv("BFD_S3_ENDPOINT")

	config := aws.Config{
		Region:           aws.String("us-east-1"),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         &endpoint,
	}

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: config,
	})

	if err != nil {
		t.Fatalf("Failed to create new session for S3: %s", err.Error())
	}

	svc := s3.New(sess)

	_, err = svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(tempBucket.String()),
	})

	if err != nil {
		t.Fatalf("Failed to create bucket %s: %s", tempBucket.String(), err.Error())
	}

	for _, input := range zipInputs {
		var b bytes.Buffer
		f := bufio.NewWriter(&b)
		w := zip.NewWriter(f)

		for _, cclfName := range input.cclfNames {
			_, err := w.Create(cclfName)
			assert.NoError(t, err)
		}

		assert.NoError(t, w.Close())
		assert.NoError(t, f.Flush())

		uploader := s3manager.NewUploader(sess)

		_, s3Err := s3manager.Uploader.Upload(*uploader, &s3manager.UploadInput{
			Bucket: aws.String(tempBucket.String()),
			Key:    aws.String(input.zipName),
			Body:   bytes.NewReader(b.Bytes()),
		})

		assert.NoError(t, s3Err)
	}

	cleanup := func() {
		svc := s3.New(sess)
		iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
			Bucket: aws.String(tempBucket.String()),
		})

		// Traverse iterator deleting each object
		if err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter); err != nil {
			logrus.Printf("Unable to delete objects from bucket %s, %s\n", tempBucket, err)
		}
	}

	return tempBucket.String(), cleanup
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
