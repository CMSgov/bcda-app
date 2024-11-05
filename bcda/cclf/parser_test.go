package cclf

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/stretchr/testify/assert"
)

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
		sspID, cecID, ngacoID, ckccID, kcfID, dcID, testID, sbxID = "A9999", "E9999", "V999", "C9999", "K9999", "D9999", "TEST999", "SBXBD001"
		sspProd, sspTest                                          = "P.BCD." + sspID, "T.BCD." + sspID
		cecProd, cecTest                                          = "P.CEC", "T.CEC"
		ngacoProd, ngacoTest                                      = "P." + ngacoID + ".ACO", "T." + ngacoID + ".ACO"
		ckccProd, ckccTest                                        = "P." + ckccID + ".ACO", "T." + ckccID + ".ACO"
		kcfProd, kcfTest                                          = "P." + kcfID + ".ACO", "T." + kcfID + ".ACO"
		dcProd, dcTest                                            = "P." + dcID + ".ACO", "T." + dcID + ".ACO"
		testProd, testTest                                        = "P." + testID + ".ACO", "T." + testID + ".ACO"
		sbxProd, sbxTest                                          = "P." + sbxID + ".ACO", "T." + sbxID + ".ACO"
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
	ckccProdFile, ckccTestFile := gen(ckccProd, validTime), gen(ckccTest, validTime)
	kcfProdFile, kcfTestFile := gen(kcfProd, validTime), gen(kcfTest, validTime)
	dcProdFile, dcTestFile := gen(dcProd, validTime), gen(dcTest, validTime)
	testProdFile, testTestFile := gen(testProd, validTime), gen(testTest, validTime)
	sbxProdFile, sbxTestFile := gen(sbxProd, validTime), gen(sbxTest, validTime)

	tests := []struct {
		name     string
		cmsID    string
		fileName string
		errMsg   string
		metadata cclfFileMetadata
	}{
		{"Non CCLF0 or CCLF8 file", sspID, "P.A0001.ACO.ZC9Y18.D190108.T2355000", "invalid filename", cclfFileMetadata{}},
		{"Unsupported CCLF file type", "Z9999", "P.Z0001.ACO.ZC8Y18.D190108.T2355000", "invalid filename", cclfFileMetadata{}},
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
		{"Production CKCC file", ckccID, ckccProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      ckccProdFile,
				cclfNum:   8,
				acoID:     ckccID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test CKCC file", ckccID, ckccTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      ckccTestFile,
				cclfNum:   8,
				acoID:     ckccID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Production KCF file", kcfID, kcfProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      kcfProdFile,
				cclfNum:   8,
				acoID:     kcfID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test KCF file", kcfID, kcfTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      kcfTestFile,
				cclfNum:   8,
				acoID:     kcfID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Production DC file", dcID, dcProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      dcProdFile,
				cclfNum:   8,
				acoID:     dcID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test DC file", dcID, dcTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      dcTestFile,
				cclfNum:   8,
				acoID:     dcID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Production TEST file", testID, testProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      testProdFile,
				cclfNum:   8,
				acoID:     testID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test TEST file", testID, testTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      testTestFile,
				cclfNum:   8,
				acoID:     testID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Production SBX file", sbxID, sbxProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      sbxProdFile,
				cclfNum:   8,
				acoID:     sbxID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{"Test SBX file", sbxID, sbxTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      sbxTestFile,
				cclfNum:   8,
				acoID:     sbxID,
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

func TestCheckIfAttributionCSVFile(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		testIsCSV bool
	}{
		{name: "Is an Attribution CSV File path", path: "P.PCPB.M2014.D00302.T2420001", testIsCSV: true},
		{name: "Is not an Attribution CSV File path", path: "T.PCPB.M2014.D00302.T2420001", testIsCSV: false},
		{name: "Is an Attribution CCLF File path", path: "P.PCPB.M2014.D00302.T2420001", testIsCSV: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(sub *testing.T) {
			isCSV := CheckIfAttributionCSVFile(test.path)
			if test.testIsCSV {
				assert.True(sub, isCSV)
			} else {
				assert.False(sub, isCSV)
			}
		})
	}
}
