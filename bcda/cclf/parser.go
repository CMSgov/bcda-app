package cclf

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/service"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/log"
	"github.com/pkg/errors"
)

func getCMSID(name string) (string, error) {
	// CCLF foldername convention with BCD identifier: P.BCD.<ACO_ID>.ZC[Y|R]**.Dyymmdd.Thhmmsst
	exp := regexp.MustCompile(`(?:T|P)\.BCD\.(.*)\.ZC[Y|R]\d{2}\.D\d{6}\.T\d{7}`)
	parts := exp.FindStringSubmatch(name)
	if len(parts) != 2 {
		err := fmt.Errorf("invalid name ('%s') for CCLF archive, parts: %v", name, parts)
		log.API.Error(err.Error())
		return "", err
	}

	return parts[1], nil
}

func CheckIfAttributionCSVFile(filePath string) bool {
	MDTCOCPattern := `(P|T)\.(PCPB)\.(M)([0-9][0-9])(\d{2})\.(D\d{6}\.T\d{6})\d`
	CDACPattern := `(P|T)\.(BCD)\.(DA)(\d{4})\.(MBIY)(\d{2})\.(D\d{6}\.T\d{6})\d`
	MDTCOCRegexp := regexp.MustCompile(MDTCOCPattern)
	CDACRegexp := regexp.MustCompile(CDACPattern)
	return (MDTCOCRegexp.MatchString(filePath) || CDACRegexp.MatchString(filePath))
}

type CSVParser struct {
	FilePath string
}

func getACOConfigs() ([]service.ACOConfig, error) {
	configs, err := service.LoadConfig()
	if err != nil {
		return []service.ACOConfig{}, err
	}
	return configs.ACOConfigs, err

}

// GetCSVMetadata builds a metadata struct based on the filename parts.
// The filename regex is part of aco configuration.
func GetCSVMetadata(path string) (csvFileMetadata, error) {
	var metadata csvFileMetadata
	var err error

	acos, err := getACOConfigs()
	if err != nil {
		return csvFileMetadata{}, errors.New("Failed to load ACO configs")
	}
	if acos == nil {
		return csvFileMetadata{}, errors.New("No ACO configs found.")
	}

	for _, v := range acos {
		// skip files that aren't csvs
		if v.AttributionFile.FileType == "csv" {
			filenameRegexp := regexp.MustCompile(v.AttributionFile.NamePattern)
			matches := filenameRegexp.FindStringSubmatch(path)
			// safely check slice that contains the model identifier
			if len(matches) >= 2 {
				// verify the regex submatch is the model identifier
				if v.AttributionFile.ModelIdentifier == matches[2] {
					metadata, err = validateCSVMetadata(v.AttributionFile, matches)
					if err != nil {
						return csvFileMetadata{}, err
					}
					break
				}
			}
		}

	}

	if metadata == (csvFileMetadata{}) {
		return metadata, errors.New("Invalid filename for csv attribution file")
	}

	metadata.name = path
	metadata.cclfNum = 8
	return metadata, nil
}

// Validate the csv attribution filename contains the required values.
// Ingestion of the file fails if the validation fails.
// This also sets various parts of metadata.
func validateCSVMetadata(attributionFile service.AttributionFile, subMatches []string) (csvFileMetadata, error) {
	var metadata csvFileMetadata
	var err error

	metadata.perfYear, err = strconv.Atoi(subMatches[attributionFile.PerformanceYear])
	if err != nil {
		err = errors.Wrapf(err, "failed to parse performance year from file")
		log.API.Error(err)
		return csvFileMetadata{}, err
	}

	filenameDate := subMatches[attributionFile.FileDate]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		err = errors.Wrapf(err, "failed to parse date '%s' from file", filenameDate)
		return csvFileMetadata{}, err
	}

	maxFileDays := utils.GetEnvInt("CCLF_MAX_AGE", 45)
	refDateString := conf.GetEnv("CCLF_REF_DATE")
	refDate, err := time.Parse("060102", refDateString)
	if err != nil {
		refDate = time.Now()
	}

	// File must not be older than 45 days
	filesNotBefore := refDate.Add(-1 * time.Duration(int64(maxFileDays*24)*int64(time.Hour)))
	filesNotAfter := refDate
	if t.Before(filesNotBefore) || t.After(filesNotAfter) {
		err = errors.New(fmt.Sprintf("date '%s' out of range; comparison date %s", filenameDate, refDate.Format("060102")))
		return csvFileMetadata{}, err
	}

	metadata.timestamp = t
	switch subMatches[1] {
	case "T":
		metadata.env = "test"
	case "P":
		metadata.env = "production"
	}

	// if attributionFile.model_identifier == PCPB
	switch attributionFile.ModelIdentifier {
	case "PCPB":
		metadata.acoID = "CT000000"
	case "GUIDE", "BCD":
		metadata.acoID = subMatches[3]
	default:
		return csvFileMetadata{}, errors.New("failed to get aco ID for attribution file")
	}

	return metadata, nil
}

// getCCLFFileMetadata takes an attribution file name and converts it to a cclfFileMetadata entry.
// The cclfFileMetadat entry will be insert into the database as a record in the cclf_files table.
func getCCLFFileMetadata(cmsID, fileName string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	const (
		prefix = `(P|T)\.`
		suffix = `\.ZC(0|8)(Y|R)(\d{2})\.(D\d{6}\.T\d{6})\d`
		aco    = `(?:\.ACO)`
		bcd    = `(?:BCD\.)`

		// CCLF file name convention for SSP with BCD identifier: P.BCD.A****.ZC[0|8][Y|R]**.Dyymmdd.Thhmmsst
		ssp = `A\d{4}`
		// CCLF file name convention for IOTA with BCD identifier: P.BCD.IOTA***.ZC(Y|R).Dyymmdd.Thhmmsst
		iotaPattern = `IOTA\d{3}`
		// CCLF file name convention for NGACO: P.V***.ACO.ZC[0|8][Y|R].Dyymmdd.Thhmmsst
		ngaco = `V\d{3}`
		// CCLF file name convention for CEC: P.CEC.ZC[0|8][Y|R].Dyymmdd.Thhmmsst
		cec = `CEC`
		// CCLF file name convention for CKCC: P.C****.ACO.ZC(Y|R)**.Dyymmdd.Thhmmsst
		ckcc = `C\d{4}`
		// CCLF file name convention for KCF: P.K****.ACO.ZC[0|8](Y|R)**.Dyymmdd.Thhmmsst
		kcf = `K\d{4}`
		// CCLF file name convention for DC: P.D****.ACO.ZC(Y|R)**.Dyymmdd.Thhmmsst
		dc = `D\d{4}`
		// CCLF file name convention for TEST: P.TEST***.ACO.ZC(Y|R)**.Dyymmdd.Thhmmsst
		test = `TEST\d{3}`
		// CCLF file name convention for SBX: P.SBX*****.ACO.ZC(Y|R)**.Dyymmdd.Thhmmsst
		sandbox = `SBX[A-Z]{2}\d{3}`

		pattern = prefix + `(` +
			bcd + ssp + `|` +
			bcd + iotaPattern + `|` +
			ngaco + aco + `|` +
			cec + `|` +
			ckcc + aco + `|` +
			kcf + aco + `|` +
			dc + aco + `|` +
			test + aco + `|` +
			sandbox + aco +
			`)` + suffix
	)

	filenameRegexp := regexp.MustCompile(pattern)
	parts := filenameRegexp.FindStringSubmatch(fileName)

	if len(parts) != 7 {
		err := fmt.Errorf("invalid filename ('%s') for CCLF file, parts: %v", fileName, parts)
		log.API.Warning(err)
		return metadata, err
	}

	cclfNum, err := strconv.Atoi(parts[3])
	if err != nil {
		err = errors.Wrapf(err, "failed to parse CCLF number from file: %s", fileName)
		log.API.Error(err)
		return metadata, err
	}

	perfYear, err := strconv.Atoi(parts[5])
	if err != nil {
		err = errors.Wrapf(err, "failed to parse performance year from file: %s", fileName)
		log.API.Error(err)
		return metadata, err
	}

	filenameDate := parts[6]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		err = errors.Wrapf(err, "failed to parse date '%s' from file: %s", filenameDate, fileName)
		log.API.Error(err)
		return metadata, err
	}

	maxFileDays := utils.GetEnvInt("CCLF_MAX_AGE", 45)
	refDateString := conf.GetEnv("CCLF_REF_DATE")
	refDate, err := time.Parse("060102", refDateString)
	if err != nil {
		refDate = time.Now()
	}

	// Files must not be too old
	filesNotBefore := refDate.Add(-1 * time.Duration(int64(maxFileDays*24)*int64(time.Hour)))
	filesNotAfter := refDate
	if t.Before(filesNotBefore) || t.After(filesNotAfter) {
		err = errors.New(fmt.Sprintf("date '%s' from file %s out of range; comparison date %s", filenameDate, fileName, refDate.Format("060102")))
		log.API.Error(err)
		return metadata, err
	}

	switch parts[1] {
	case "T":
		metadata.env = "test"
	case "P":
		metadata.env = "production"
	}

	switch parts[4] {
	case "Y":
		metadata.fileType = models.FileTypeDefault
	case "R":
		metadata.fileType = models.FileTypeRunout
	}

	metadata.name = parts[0]
	metadata.cclfNum = cclfNum
	metadata.acoID = cmsID
	metadata.timestamp = t
	metadata.perfYear = perfYear

	return metadata, nil
}
