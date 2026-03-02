package cclf

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/pkg/errors"
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
		{"validIotaPath", "path/T.BCD.IOTA123.ZCY18.D181120.T2000000", false, "IOTA123"},
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

func TestGetCSVMetadata(t *testing.T) {
	start := time.Now()
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second(), 0,
		time.UTC)

	dateFormat := "D060102.T1504050"

	validTime := startUTC.Add(-24 * time.Hour)
	fileDateTime := validTime.Format(dateFormat)

	tests := []struct {
		name     string
		fileName string
		errMsg   string
		metadata csvFileMetadata
	}{
		{"valid MDTCoC csv filename", "P.PCPB.M2411." + fileDateTime, "", csvFileMetadata{
			env:       "production",
			name:      "P.PCPB.M2411." + fileDateTime,
			cclfNum:   8,
			acoID:     "CT000000",
			timestamp: validTime,
			perfYear:  24,
			fileType:  models.FileTypeDefault,
		},
		},
		{"valid CDAC csv filename", "P.BCD.DA0000.MBIY25." + fileDateTime, "", csvFileMetadata{
			env:       "production",
			name:      "P.BCD.DA0000.MBIY25." + fileDateTime,
			cclfNum:   8,
			acoID:     "DA0000",
			timestamp: validTime,
			perfYear:  25,
			fileType:  models.FileTypeDefault,
		},
		},
		{"valid GUIDE csv filename", "P.GUIDE.GUIDE-00001.Y25." + fileDateTime, "", csvFileMetadata{
			env:       "production",
			name:      "P.GUIDE.GUIDE-00001.Y25." + fileDateTime,
			cclfNum:   8,
			acoID:     "GUIDE-00001",
			timestamp: validTime,
			perfYear:  25,
			fileType:  models.FileTypeDefault,
		},
		},
		{"invalid csv filename", "P.PPB.M2411." + fileDateTime, "Invalid filename", csvFileMetadata{}},
		{"invalid csv filename - extra digit", "P.PCPB.M24112." + fileDateTime, "Invalid filename", csvFileMetadata{}},
		{"invalid csv filename - env", "A.PCPB.M24112." + fileDateTime, "Invalid filename", csvFileMetadata{}},
		{"invalid csv filename - dupe match", "P.PCPBPCPB.M2411." + fileDateTime, "Invalid filename", csvFileMetadata{}},
		{"invalid csv filename - dupe match", "P.P.GUIDE.GUIDE-.Y25." + fileDateTime, "Invalid filename", csvFileMetadata{}},
		{"invalid csv filename - dupe match", "T.GUIDE.Y25." + fileDateTime, "Invalid filename", csvFileMetadata{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := GetCSVMetadata(tt.fileName)
			if tt.errMsg == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tt.errMsg)
			}
			assert.Equal(t, tt.metadata, metadata)
		})
	}
}

func TestValidateCSVFileName(t *testing.T) {
	start := time.Now()
	startUTC := time.Date(start.Year(), start.Month(), start.Day(), start.Hour(), start.Minute(), start.Second(), 0,
		time.UTC)

	dateFormat := "D060102.T1504050"

	validTime := startUTC.Add(-24 * time.Hour)
	fileDateTime := validTime.Format(dateFormat)

	futureTime := startUTC.Add(24 * time.Hour)

	tests := []struct {
		name     string
		fileName string
		err      error
		metadata csvFileMetadata
	}{
		{"valid MDTCoC csv filename", "P.PCPB.M2411." + fileDateTime, nil, csvFileMetadata{
			env:       "production",
			timestamp: validTime,
			perfYear:  24,
			fileType:  models.FileTypeDefault,
			acoID:     "CT000000",
		},
		},
		{"valid MDTCoC csv test filename", "T.PCPB.M2411." + fileDateTime, nil, csvFileMetadata{
			env:       "test",
			timestamp: validTime,
			perfYear:  24,
			fileType:  models.FileTypeDefault,
			acoID:     "CT000000",
		},
		},
		{"invalid MDTCoC csv - file date too old", "P.PCPB.M2411.D201101.T0000001", errors.New("out of range"), csvFileMetadata{}},
		{"invalid MDTCoC csv - file date in the future", "P.PCPB.M2411." + futureTime.Format(dateFormat), errors.New("out of range"), csvFileMetadata{}},

		{"valid CDAC csv filename", "P.BCD.DA0000.MBIY25." + fileDateTime, nil, csvFileMetadata{
			env:       "production",
			timestamp: validTime,
			perfYear:  25,
			fileType:  models.FileTypeDefault,
			acoID:     "DA0000",
		},
		},
		{"valid CDAC csv test filename", "T.BCD.DA0000.MBIY25." + fileDateTime, nil, csvFileMetadata{
			env:       "test",
			timestamp: validTime,
			perfYear:  25,
			fileType:  models.FileTypeDefault,
			acoID:     "DA0000",
		},
		},
		{"invalid CDAC csv - file date too old", "P.BCD.DA0000.MBIY11.D201101.T0000001", errors.New("out of range"), csvFileMetadata{}},
		{"invalid CDAC csv - file date in the future", "P.BCD.DA0000.MBIY11." + futureTime.Format(dateFormat), errors.New("out of range"), csvFileMetadata{}},
		{"valid GUIDE csv filename", "P.GUIDE.GUIDE-00001.Y25." + fileDateTime, nil, csvFileMetadata{
			env:       "production",
			acoID:     "GUIDE-00001",
			timestamp: validTime,
			perfYear:  25,
			fileType:  models.FileTypeDefault,
		},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			acos, err := getACOConfigs()
			var actualmetadata csvFileMetadata
			for _, v := range acos {
				filenameRegexp := regexp.MustCompile(v.AttributionFile.NamePattern)
				parts := filenameRegexp.FindStringSubmatch(test.fileName)
				if len(parts) >= 2 {
					if v.AttributionFile.ModelIdentifier == parts[2] {
						actualmetadata, err = validateCSVMetadata(v.AttributionFile, parts)
					}
				}

			}
			if test.err != nil {
				assert.Contains(t, err.Error(), test.err.Error())
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, test.metadata, actualmetadata)
		})
	}

}

func TestGetCCLFMetadata(t *testing.T) {
	const (
		sspID, iotaID, cecID, ngacoID, ckccID, kcfID, dcID, testID, sbxID = "A9999", "IOTA965", "E9999", "V999", "C9999", "K9999", "D9999", "TEST999", "SBXBD001"
		sspProd, sspTest                                                  = "P.BCD." + sspID, "T.BCD." + sspID
		iotaProd, iotaTest                                                = "P." + iotaID + ".PRT", "T." + iotaID + ".PRT"
		cecProd, cecTest                                                  = "P.CEC", "T.CEC"
		ngacoProd, ngacoTest                                              = "P." + ngacoID + ".ACO", "T." + ngacoID + ".ACO"
		ckccProd, ckccTest                                                = "P." + ckccID + ".ACO", "T." + ckccID + ".ACO"
		kcfProd, kcfTest                                                  = "P." + kcfID + ".ACO", "T." + kcfID + ".ACO"
		dcProd, dcTest                                                    = "P." + dcID + ".ACO", "T." + dcID + ".ACO"
		testProd, testTest                                                = "P." + testID + ".ACO", "T." + testID + ".ACO"
		sbxProd, sbxTest                                                  = "P." + sbxID + ".ACO", "T." + sbxID + ".ACO"
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
	iotaProdFile, iotaTestFile, iotaRunoutFile := gen(iotaProd, validTime), gen(iotaTest, validTime), strings.Replace(gen(iotaProd, validTime), "ZC8Y", "ZC8R", 1)
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
		{"Unsupported CSV file type", sspID, "P.PCPB.M2014.D00302.T2420001", "invalid filename", cclfFileMetadata{}},
		{"Invalid date (no 13th month)", sspID, "T.BCD.A0001.ZC0Y18.D181320.T0001000", "failed to parse date", cclfFileMetadata{}},
		{"CCLF file too old", sspID, gen(sspProd, startUTC.Add(-365*24*time.Hour)), "out of range", cclfFileMetadata{}},
		{"CCLF file too new", sspID, gen(sspProd, startUTC.Add(365*24*time.Hour)), "out of range", cclfFileMetadata{}},
		{
			"Production SSP file", sspID, sspProdFile, "",
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
		{
			"Test SSP file", sspID, sspTestFile, "",
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
		{
			"Production IOTA file", iotaID, iotaProdFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      iotaProdFile,
				cclfNum:   8,
				acoID:     iotaID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{
			"Test IOTA file", iotaID, iotaTestFile, "",
			cclfFileMetadata{
				env:       "test",
				name:      iotaTestFile,
				cclfNum:   8,
				acoID:     iotaID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeDefault,
			},
		},
		{
			"Runout IOTA file", iotaID, iotaRunoutFile, "",
			cclfFileMetadata{
				env:       "production",
				name:      iotaRunoutFile,
				cclfNum:   8,
				acoID:     iotaID,
				timestamp: validTime,
				perfYear:  perfYear,
				fileType:  models.FileTypeRunout,
			},
		},
		{
			"Production CEC file", cecID, cecProdFile, "",
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
		{
			"Test CEC file", cecID, cecTestFile, "",
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
		{
			"Production NGACO file", ngacoID, ngacoProdFile, "",
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
		{
			"Test NGACO file", ngacoID, ngacoTestFile, "",
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
		{
			"Production CKCC file", ckccID, ckccProdFile, "",
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
		{
			"Test CKCC file", ckccID, ckccTestFile, "",
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
		{
			"Production KCF file", kcfID, kcfProdFile, "",
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
		{
			"Test KCF file", kcfID, kcfTestFile, "",
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
		{
			"Production DC file", dcID, dcProdFile, "",
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
		{
			"Test DC file", dcID, dcTestFile, "",
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
		{
			"Production TEST file", testID, testProdFile, "",
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
		{
			"Test TEST file", testID, testTestFile, "",
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
		{
			"Production sandbox file", sbxID, sbxProdFile, "",
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
		{
			"Test sandbox file", sbxID, sbxTestFile, "",
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
		{name: "Is an Attribution MDTCoC CSV File path", path: "P.PCPB.M2014.D003026.T0000001", testIsCSV: true},
		{name: "Is not an Attribution MDTCoC CSV File path (incorrect first)", path: "M.PCPB.M2014.D00302.T2420001", testIsCSV: false},
		{name: "Is not an Attribution MDTCoC CSV File path (incorrect second)", path: "P.BFD.N2014.D00302.T2420001", testIsCSV: false},
		{name: "Is not an Attribution MDTCoC CSV File path (incorrect third)", path: "P.PCPB.M2014.D00302.T2420001", testIsCSV: false},
		{name: "Is not an Attribution MDTCoC CSV File path (incorrect fourth)", path: "P.PCPB.M2014.D00302.T2420001", testIsCSV: false},
		{name: "Is not an Attribution MDTCoC CSV File path (incorrect fifth)", path: "P.PCPB.M2014.D00302.T24200011", testIsCSV: false},
		{name: "Is not an Attribution MDTCoC CSV File path (CCLF file)", path: "T.BCD.A0001.ZCY18.D181121.T1000000", testIsCSV: false},
		{name: "Is not an Attribution MDTCoC CSV File path (opt-out file)", path: "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", testIsCSV: false},

		{name: "Is an Attribution CDAC CSV File path", path: "P.BCD.DA0000.MBIY25.D003026.T0000001", testIsCSV: true},
		{name: "Is not an Attribution CDAC CSV File path (incorrect first)", path: "x.BCD.DA0000.MBIY25.D00302.T2420001", testIsCSV: false},
		{name: "Is not an Attribution CDAC CSV File path (incorrect second)", path: "P.BxD.DA0000.MBIY25.T2420001", testIsCSV: false},
		{name: "Is not an Attribution CDAC CSV File path (incorrect third)", path: "P.BCD.Dx00x0.MBIY25.D00302.T2420001", testIsCSV: false},
		{name: "Is not an Attribution CDAC CSV File path (incorrect fourth)", path: "P.BCD.DA0000.MBxY2x.D00302.T2420001", testIsCSV: false},
		{name: "Is not an Attribution CDAC CSV File path (incorrect fifth)", path: "P.BCD.DA0000.MBIY25.D0x30x.T24200011", testIsCSV: false},
		{name: "Is not an Attribution CDAC CSV File path (CCLF file)", path: "T.BCD.A0001.ZCY18.D181121.T1000000", testIsCSV: false},
		{name: "Is not an Attribution CDAC CSV File path (opt-out file)", path: "T#EFT.ON.ACO.NGD1800.DPRF.D181120.T1000009", testIsCSV: false},
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
