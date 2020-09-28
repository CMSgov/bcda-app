package cclf

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
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
	s.cclfRefDate = os.Getenv("CCLF_REF_DATE")
	os.Setenv("CCLF_REF_DATE", "181201") // Needed to allow our static CCLF files to continue to be processed
	dir, err := ioutil.TempDir("", "*")
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
	os.Setenv("CCLF_REF_DATE", s.cclfRefDate)
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives() {
	cmsID, perfYear := "A0001", 18
	tests := []struct {
		path         string
		numCCLFFiles int // Expected count for the cmsID, perfYear above
		skipped      int
		numCCLF0     int // Expected count for the cmsID, perfYear above
		numCCLF8     int // Expected count for the cmsID, perfYear above
	}{
		{filepath.Join(s.basePath, "cclf/archives/valid/"), 2, 1, 1, 1},
		{filepath.Join(s.basePath, "cclf/archives/bcd/"), 2, 1, 1, 1},
		{filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/"), 2, 5, 1, 1},
		{filepath.Join(s.basePath, "cclf/mixed/0/valid_names/"), 3, 3, 3, 0},
		{filepath.Join(s.basePath, "cclf/archives/8/valid/"), 5, 0, 0, 5},
		{filepath.Join(s.basePath, "cclf/files/9/valid_names/"), 0, 4, 0, 0},
		{filepath.Join(s.basePath, "cclf/mixed/with_folders/"), 2, 13, 1, 1},
	}

	for _, tt := range tests {
		s.T().Run(tt.path, func(t *testing.T) {
			cclfMap, skipped, err := processCCLFArchives(tt.path)
			cclfFiles := cclfMap[cmsID][perfYear]
			assert.NoError(t, err)
			assert.Equal(t, tt.skipped, skipped)
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
	folderPath := filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/")
	filePath := filepath.Join(folderPath, "T.BCDE.ACO.ZC0Y18.D181120.T0001000")

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err.Error())
	}

	cclfMap, skipped, err := processCCLFArchives(folderPath)
	assert.Nil(err)
	cclfList := cclfMap["A0001"][18]
	assert.Equal(2, len(cclfList))
	assert.Equal(5, skipped)
	// assert that this file is still here.
	_, err = os.Open(filePath)
	assert.Nil(err)

	timeChange := origTime.Add(-(time.Hour * 73)).Truncate(time.Second)
	err = os.Chtimes(filePath, timeChange, timeChange)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err)
	}

	cclfMap, skipped, err = processCCLFArchives(folderPath)
	assert.Nil(err)
	cclfList = cclfMap["A0001"][18]
	assert.Equal(2, len(cclfList))
	assert.Equal(5, skipped)

	// assert that this file is not still here.
	_, err = os.Open(filePath)
	assert.EqualError(err, fmt.Sprintf("open %s: no such file or directory", filePath))
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives_InvalidPath() {
	cclfMap, skipped, err := processCCLFArchives("./foo")
	assert.EqualError(s.T(), err, "error in sorting cclf file: nil,: lstat ./foo: no such file or directory")
	assert.Equal(s.T(), 0, skipped)
	assert.Nil(s.T(), cclfMap)
}

func TestFileProcessorTestSuite(t *testing.T) {
	suite.Run(t, new(FileProcessorTestSuite))
}
func TestGetCMSID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		hasError bool
		cmsID    string
	}{
		{"validSSPPath", "path/T.BCD.A0001.ZCY18.D181120.T1000000", false, "A0001"},
		{"validNGACOPath", "path/T.BCD.V299.ZCY19.D191005.T0209260", false, "V299"},
		{"validCECPath", "path/T.BCD.E9999.ZCY19.D191005.T0209260", false, "E9999"},
		{"missingBCD", "path/T.A0001.ACO.ZC8Y18.D18NOV20.T1000009", true, ""},
		{"empty", "", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(sub *testing.T) {
			cmsID, err := getCMSID(tt.path)
			if tt.hasError {
				assert.Contains(sub, err.Error(), tt.path)
			}
			assert.Equal(sub, tt.cmsID, cmsID)
		})
	}
}

func TestGetCCLFMetadata(t *testing.T) {
	const (
		sspID, cecID, ngacoID = "A9999", "E9999", "V999"
		sspProd, sspTest      = "P.BCD." + sspID, "T.BCD." + sspID
		cecProd, cecTest      = "P.CEC", "T.CEC"
		ngacoProd, ngacoTest  = "P." + ngacoID + ".ACO", "T." + ngacoID + ".ACO"
	)

	start := time.Now()
	// Need to use UTC zone information to make the time comparison easier
	// CCLF file format does not contain any tz information, so we assume UTC time
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second(), 0,
		time.UTC)

	gen := func(prefix string, t time.Time) string {
		const (
			format         = "D060102.T1504050"
			perfYearFormat = "06"
		)
		return fmt.Sprintf("%s.ZC8Y%s.%s", prefix, t.Format(perfYearFormat), t.Format(format))
	}

	// Timestamp that'll satisfy the time window requirement
	validTime := startUTC.Add(-24 * time.Hour)
	sspProdFile, sspTestFile := gen(sspProd, validTime), gen(sspTest, validTime)
	cecProdFile, cecTestFile := gen(cecProd, validTime), gen(cecTest, validTime)
	ngacoProdFile, ngacoTestFile := gen(ngacoProd, validTime), gen(ngacoTest, validTime)

	tests := []struct {
		name     string
		cmsID    string
		fileName string
		errMsg   string
		metadata cclfFileMetadata
	}{
		{"Non CCLF0 or CCLF8 file", sspID, "P.A0001.ACO.ZC9Y18.D190108.T2355000", "invalid filename", cclfFileMetadata{}},
		{"Invalid date (no 13th month)", sspID, "T.BCD.A0001.ZC0Y18.D181320.T0001000", "failed to parse date", cclfFileMetadata{}},
		{"CCLF file too old", sspID, gen(sspProd, startUTC.Add(-365*24*time.Hour)), "out of range", cclfFileMetadata{}},
		{"CCLF file too new", sspID, gen(sspProd, startUTC.Add(365*24*time.Hour)), "out of range", cclfFileMetadata{}},
		{"Production SSP file", sspID, sspProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      sspProdFile,
				cclfNum:   8,
				acoID:     sspID,
				timestamp: validTime,
				perfYear:  20,
			},
		},
		{"Test SSP file", sspID, sspTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      sspTestFile,
				cclfNum:   8,
				acoID:     sspID,
				timestamp: validTime,
				perfYear:  20,
			},
		},
		{"Production CEC file", cecID, cecProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      cecProdFile,
				cclfNum:   8,
				acoID:     cecID,
				timestamp: validTime,
				perfYear:  20,
			},
		},
		{"Test CEC file", cecID, cecTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      cecTestFile,
				cclfNum:   8,
				acoID:     cecID,
				timestamp: validTime,
				perfYear:  20,
			},
		},
		{"Production NGACO file", ngacoID, ngacoProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      ngacoProdFile,
				cclfNum:   8,
				acoID:     ngacoID,
				timestamp: validTime,
				perfYear:  20,
			},
		},
		{"Test NGACO file", ngacoID, ngacoTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      ngacoTestFile,
				cclfNum:   8,
				acoID:     ngacoID,
				timestamp: validTime,
				perfYear:  20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(sub *testing.T) {
			metadata, err := getCCLFFileMetadata(tt.cmsID, tt.fileName)
			if tt.errMsg == "" {
				assert.NoError(sub, err)
			} else {
				assert.Contains(sub, err.Error(), tt.errMsg)
			}
			assert.Equal(sub, tt.metadata, metadata)
		})
	}
}
