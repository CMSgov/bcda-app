package cclf

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
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
		fmt.Println(err.Error())
		log.API.Error(err.Error())
		return "", err
	}

	return parts[1], nil
}

func getCCLFFileMetadata(cmsID, fileName string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	const (
		prefix = `(P|T)\.`
		suffix = `\.ZC(0|8)(Y|R)(\d{2})\.(D\d{6}\.T\d{6})\d`
		aco    = `(?:\.ACO)`
		bcd    = `(?:BCD\.)`

		// CCLF filename convention for SSP with BCD identifier: P.BCD.A****.ZC[0|8][Y|R]**.Dyymmdd.Thhmmsst
		ssp = `A\d{4}`
		// CCLF filename convention for NGACO: P.V***.ACO.ZC[0|8][Y|R].Dyymmdd.Thhmmsst
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

		pattern = prefix + `(` + bcd + ssp + `|` + ngaco + aco + `|` + cec +
			`|` + ckcc + aco + `|` + kcf + aco + `|` + dc + aco +
			`|` + test + aco + `|` + sandbox + aco + `)` + suffix
	)

	filenameRegexp := regexp.MustCompile(pattern)
	parts := filenameRegexp.FindStringSubmatch(fileName)

	if len(parts) != 7 {
		err := fmt.Errorf("invalid filename ('%s') for CCLF file, parts: %v", fileName, parts)
		fmt.Println(err.Error())
		log.API.Error(err)
		return metadata, err
	}

	cclfNum, err := strconv.Atoi(parts[3])
	if err != nil {
		err = errors.Wrapf(err, "failed to parse CCLF number from file: %s", fileName)
		fmt.Println(err.Error())
		log.API.Error(err)
		return metadata, err
	}

	perfYear, err := strconv.Atoi(parts[5])
	if err != nil {
		err = errors.Wrapf(err, "failed to parse performance year from file: %s", fileName)
		fmt.Println(err.Error())
		log.API.Error(err)
		return metadata, err
	}

	filenameDate := parts[6]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		err = errors.Wrapf(err, "failed to parse date '%s' from file: %s", filenameDate, fileName)
		fmt.Println(err.Error())
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
		fmt.Println(err.Error())
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