package cclf

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

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
	conf.SetEnv(s.T(), "CCLF_REF_DATE", s.cclfRefDate)
	os.RemoveAll(s.pendingDeletionDir)
}

func (s *FileProcessorTestSuite) TestProcessCCLFArchives() {
	cmsID, key := "A0001", metadataKey{perfYear: 18, fileType: models.FileTypeDefault}
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
			cclfFiles := cclfMap[cmsID][key]
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
	key := metadataKey{perfYear: 18, fileType: models.FileTypeDefault}
	folderPath := filepath.Join(s.basePath, "cclf/mixed/with_invalid_filenames/")
	filePath := filepath.Join(folderPath, "T.BCDE.ACO.ZC0Y18.D181120.T0001000")

	origTime := time.Now().Truncate(time.Second)
	err := os.Chtimes(filePath, origTime, origTime)
	if err != nil {
		s.FailNow("Failed to change modified time for file", err.Error())
	}

	cclfMap, skipped, err := processCCLFArchives(folderPath)
	assert.Nil(err)
	cclfList := cclfMap["A0001"][key]
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
	cclfList = cclfMap["A0001"][key]
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
		{"validSSPRunoutPath", "path/T.BCD.A0002.ZCR18.D181120.T1000000", false, "A0002"},
		{"validNGACOPath", "path/T.BCD.V299.ZCY19.D191005.T0209260", false, "V299"},
		{"validCECPath", "path/T.BCD.E9999.ZCY19.D191005.T0209260", false, "E9999"},
		{"missingBCD", "path/T.A0001.ACO.ZCY18.D18NOV20.T1000009", true, ""},
		{"not ZCY or ZCR", "path/T.BCD.A0001.ZC18.D181120.T1000000", true, ""},
		{"missing ZCY and ZCR", "path/T.BCD.A0001.ZCA18.D181120.T1000000", true, ""},
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

	const (
		dateFormat     = "D060102.T1504050"
		perfYearFormat = "06"
	)
	gen := func(prefix string, t time.Time) string {
		return fmt.Sprintf("%s.ZC8Y%s.%s", prefix, t.Format(perfYearFormat), t.Format(dateFormat))
	}

	// Timestamp that'll satisfy the time window requirement
	validTime := startUTC.Add(-24 * time.Hour)
	perfYear, err := strconv.Atoi(validTime.Format(perfYearFormat))
	assert.NoError(t, err)
	sspProdFile, sspTestFile, sspRunoutFile := gen(sspProd, validTime), gen(sspTest, validTime),
		strings.Replace(gen(sspProd, validTime), "ZC8Y", "ZC8R", 1)
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
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test SSP file", sspID, sspTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      sspTestFile,
				cclfNum:   8,
				acoID:     sspID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{
			"Runout SSP file", sspID, sspRunoutFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      sspRunoutFile,
				cclfNum:   8,
				acoID:     sspID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeRunout,
			},
		},
		{"Production CEC file", cecID, cecProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      cecProdFile,
				cclfNum:   8,
				acoID:     cecID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test CEC file", cecID, cecTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      cecTestFile,
				cclfNum:   8,
				acoID:     cecID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Production NGACO file", ngacoID, ngacoProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      ngacoProdFile,
				cclfNum:   8,
				acoID:     ngacoID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test NGACO file", ngacoID, ngacoTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      ngacoTestFile,
				cclfNum:   8,
				acoID:     ngacoID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
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

func TestMultipleFileTypes(t *testing.T) {
	dir, err := ioutil.TempDir("", "*")
	assert.NoError(t, err)
	// Hard code the reference date to ensure to ensure we do not reject any CCLF files because they are too old.
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

	m, s, err := processCCLFArchives(dir)
	assert.NoError(t, err)
	assert.Equal(t, 0, s)
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
