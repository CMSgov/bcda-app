package cclf

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/conf"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

type metadataKey struct {
	perfYear int
	fileType models.CCLFFileType
}

// processCCLFArchives walks through all of the CCLF files captured in the root path and generates
// a mapping between CMS_ID + perf year and associated CCLF Metadata
func processCCLFArchives(rootPath string) (map[string]map[metadataKey][]*cclfFileMetadata, int, error) {
	p := &processor{0, make(map[string]map[metadataKey][]*cclfFileMetadata)}
	if err := filepath.Walk(rootPath, p.walk); err != nil {
		return nil, 0, err
	}
	return p.cclfMap, p.skipped, nil
}

type processor struct {
	skipped int
	cclfMap map[string]map[metadataKey][]*cclfFileMetadata
}

func (p *processor) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		// In case the caller supplied an err, we know that info is nil
		// See: https://golang.org/pkg/path/filepath/#WalkFunc
		var fileName = "nil"
		err = errors.Wrapf(err, "error in sorting cclf file: %v,", fileName)
		fmt.Println(err.Error())
		log.Error(err)
		return err
	}

	if info.IsDir() {
		msg := fmt.Sprintf("Unable to sort %s: directory, not a CCLF archive.", path)
		fmt.Println(msg)
		log.Warn(msg)
		return nil
	}

	zipReader, err := zip.OpenReader(filepath.Clean(path))
	if err != nil {
		p.skipped = p.skipped + 1
		msg := fmt.Sprintf("Skipping %s: file is not a CCLF archive.", path)
		fmt.Println(msg)
		log.Warn(msg)
		return nil
	}
	if err = zipReader.Close(); err != nil {
		log.Warnf("Failed to close zip file %s", err.Error())
	}

	// validate the top level zipped folder
	cmsID, err := getCMSID(info.Name())
	if err != nil {
		return p.handleArchiveError(path, info, err)
	}

	supported := models.IsSupportedACO(cmsID)
	if !supported {
		return p.handleArchiveError(path, info, fmt.Errorf("cmsID %s not supported", cmsID))
	}

	for _, f := range zipReader.File {
		metadata, err := getCCLFFileMetadata(cmsID, f.Name)
		metadata.filePath = path
		metadata.deliveryDate = info.ModTime()

		if err != nil {
			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			msg := fmt.Sprintf("Unknown file found: %s.", f.Name)
			fmt.Println(msg)
			log.Error(msg)
			continue
		}

		key := metadataKey{perfYear: metadata.perfYear, fileType: metadata.fileType}
		sub := p.cclfMap[metadata.acoID]
		if sub == nil {
			sub = make(map[metadataKey][]*cclfFileMetadata)
			p.cclfMap[metadata.acoID] = sub
		}
		sub[key] = append(sub[key], &metadata)
	}

	return nil
}

func (p *processor) handleArchiveError(path string, info os.FileInfo, cause error) error {
	p.skipped = p.skipped + 1
	msg := fmt.Sprintf("Skipping CCLF archive (%s): %s.", info.Name(), cause)
	fmt.Println(msg)
	log.Warn(msg)
	err := checkDeliveryDate(path, info.ModTime())
	if err != nil {
		err = fmt.Errorf("error moving unknown file %s to pending deletion dir", path)
		fmt.Println(err.Error())
		log.Error(err)
	}

	return err
}

func getCMSID(name string) (string, error) {
	// CCLF foldername convention with BCD identifier: P.BCD.<ACO_ID>.ZC[Y|R]**.Dyymmdd.Thhmmsst
	exp := regexp.MustCompile(`(?:T|P)\.BCD\.(.*)\.ZC[Y|R]\d{2}\.D\d{6}\.T\d{7}`)
	parts := exp.FindStringSubmatch(name)
	if len(parts) != 2 {
		err := fmt.Errorf("invalid name ('%s') for CCLF archive, parts: %v", name, parts)
		fmt.Println(err.Error())
		log.Error(err.Error())
		return "", err
	}

	return parts[1], nil
}

func getCCLFFileMetadata(cmsID, fileName string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	// CCLF filename convention for SSP with BCD identifier: P.BCD.A****.ZC[0|8][Y|R]**.Dyymmdd.Thhmmsst
	// CCLF filename convention for NGACO:  P.V***.ACO.ZC[0|8][Y|R].Dyymmdd.Thhmmsst
	// CCLF file name convetion for CEC: P.CEC.ZC[0|8][Y|R].Dyymmdd.Thhmmsst
	filenameRegexp := regexp.MustCompile(`(T|P)(?:\.BCD)?\.(.*?)(?:\.ACO)?\.ZC(0|8)(Y|R)(\d{2})\.(D\d{6}\.T\d{6})\d`)
	parts := filenameRegexp.FindStringSubmatch(fileName)

	if len(parts) != 7 {
		err := fmt.Errorf("invalid filename ('%s') for CCLF file, parts: %v", fileName, parts)
		log.Error(err)
		return metadata, err
	}

	cclfNum, err := strconv.Atoi(parts[3])
	if err != nil {
		err = errors.Wrapf(err, "failed to parse CCLF number from file: %s", fileName)
		fmt.Println(err.Error())
		log.Error(err)
		return metadata, err
	}

	perfYear, err := strconv.Atoi(parts[5])
	if err != nil {
		err = errors.Wrapf(err, "failed to parse performance year from file: %s", fileName)
		fmt.Println(err.Error())
		log.Error(err)
		return metadata, err
	}

	filenameDate := parts[6]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		err = errors.Wrapf(err, "failed to parse date '%s' from file: %s", filenameDate, fileName)
		fmt.Println(err.Error())
		log.Error(err)
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
		log.Error(err)
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

func checkDeliveryDate(folderPath string, deliveryDate time.Time) error {
	deleteThreshold := time.Hour * time.Duration(utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72))
	if deliveryDate.Add(deleteThreshold).Before(time.Now()) {
		folderName := filepath.Base(folderPath)
		newpath := fmt.Sprintf("%s/%s", conf.GetEnv("PENDING_DELETION_DIR"), folderName)
		err := os.Rename(folderPath, newpath)
		if err != nil {
			return err
		}
	}
	return nil
}
