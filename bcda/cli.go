package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/models"
)

func createACO(name, cmsID string) (string, error) {
	if name == "" {
		return "", errors.New("ACO name (--name) must be provided")
	}

	var cmsIDPt *string
	if cmsID != "" {
		acoIDFmt := regexp.MustCompile(`^A\d{4}$`)
		if !acoIDFmt.MatchString(cmsID) {
			return "", errors.New("ACO CMS ID (--cms-id) is invalid")
		}
		cmsIDPt = &cmsID
	}

	acoUUID, err := models.CreateACO(name, cmsIDPt)
	if err != nil {
		return "", err
	}

	return acoUUID.String(), nil
}

type cclfFileMetadata struct {
	env       string
	acoID     string
	cclfNum   int
	timestamp time.Time
	filePath  string
}

type cclfFileValidator struct {
	totalRecordCount int
	maxRecordLength  int
}

func importCCLF0(fileMetadata cclfFileMetadata) (map[string]cclfFileValidator, error) {
	var validator map[string]cclfFileValidator
	if fileMetadata.filePath == "" {
		return nil, errors.New("file path (--file) must be provided")
	}

	if fileMetadata.cclfNum != 0 {
		return nil, errors.New("invalid CCLF0 filename")
	}

	file, err := os.Open(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		return nil, err
	}

	fmt.Printf("File contains %s data for ACO %s at %s.\n", fileMetadata.env, fileMetadata.acoID, fileMetadata.timestamp)

	const (
		fileNumStart, fileNumEnd           = 0, 13
		totalRecordStart, totalRecordEnd   = 35, 55
		recordLengthStart, recordLengthEnd = 56, 69
	)

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			filetype := string(bytes.TrimSpace(b[fileNumStart:fileNumEnd]))
			if filetype == "CCLF8" || filetype == "CCLF9" {
				if validator == nil {
					validator = make(map[string]cclfFileValidator)
				}
				count, err := strconv.Atoi(string(bytes.TrimSpace(b[totalRecordStart:totalRecordEnd])))
				if err != nil {
					return nil, err
				}
				length, err := strconv.Atoi(string(bytes.TrimSpace(b[recordLengthStart:recordLengthEnd])))
				if err != nil {
					return nil, err
				}
				validator[filetype] = cclfFileValidator{totalRecordCount: count, maxRecordLength: length}
			}
		}
	}
	return validator, nil
}

func importCCLF8(fileMetadata cclfFileMetadata, cclfFileValidator map[string]cclfFileValidator) error {
	if fileMetadata.filePath == "" {
		return errors.New("file path (--file) must be provided")
	}

	if fileMetadata.cclfNum != 8 {
		return errors.New("invalid CCLF8 filename")
	}

	file, err := os.Open(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		return err
	}

	fmt.Printf("File contains %s data for ACO %s at %s.\n", fileMetadata.env, fileMetadata.acoID, fileMetadata.timestamp)

	const (
		mbiStart, mbiEnd   = 0, 11
		hicnStart, hicnEnd = 11, 22
	)

	sc := bufio.NewScanner(file)
	count := 0
	validator := cclfFileValidator["CCLF8"]

	for sc.Scan() {
		b := sc.Bytes()
		bytelength := len(bytes.TrimSpace(b))
		if bytelength > 0 && bytelength <= validator.maxRecordLength {
			count++
			// currently only errors if there are more records than we expect.
			if count > validator.totalRecordCount {
				return errors.New("maximum record count reached: did not match cclf0 validation")
			}
			fmt.Printf("\nMBI: %s\n", b[mbiStart:mbiEnd])
			fmt.Printf("HICN: %s\n", b[hicnStart:hicnEnd])
		} else {
			return errors.New("incorrect record length: did not match cclf0 validation")
		}
	}
	return nil
}

func importCCLF9(fileMetadata cclfFileMetadata, cclfFileValidator map[string]cclfFileValidator) error {
	if fileMetadata.filePath == "" {
		return errors.New("file path (--file) must be provided")
	}
	if fileMetadata.cclfNum != 9 {
		return errors.New("invalid CCLF9 filename")
	}

	file, err := os.Open(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		return err
	}

	fmt.Printf("File contains %s data for ACO %s at %s.\n", fileMetadata.env, fileMetadata.acoID, fileMetadata.timestamp)

	const (
		currIDStart, currIDEnd               = 1, 12
		prevIDStart, prevIDEnd               = 12, 23
		prevIDEffDateStart, prevIDEffDateEnd = 23, 33
		prevIDObsDateStart, prevIDObsDateEnd = 33, 43
	)

	sc := bufio.NewScanner(file)
	count := 0
	validator := cclfFileValidator["CCLF9"]

	for sc.Scan() {
		b := sc.Bytes()
		bytelength := len(bytes.TrimSpace(b))
		if bytelength > 0 && bytelength <= validator.maxRecordLength {
			count++
			// currently only errors if there are more records than we expect.
			if count > validator.totalRecordCount {
				return errors.New("maximum record count reached: did not match cclf0 validation")
			}
			fmt.Printf("\nXREF: %s\n", b[0:1])
			fmt.Printf("Current identifier: %s\n", b[currIDStart:currIDEnd])
			fmt.Printf("Previous identifier: %s\n", b[prevIDStart:prevIDEnd])
			fmt.Printf("Previous identifier effective date: %s\n", b[prevIDEffDateStart:prevIDEffDateEnd])
			fmt.Printf("Previous identifier obsolete date: %s\n", b[prevIDObsDateStart:prevIDObsDateEnd])
		} else {
			return errors.New("incorrect record length: did not match cclf0 validation")
		}
	}
	return nil
}

func getCCLFFileMetadata(filePath string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	// CCLF8/9 filename convention for SSP: P.A****.ACO.ZC*Y**.Dyymmdd.Thhmmsst
	// Prefix: T = test, P = prod; A**** = ACO ID; ZC* = CCLF file number; Y** = performance year
	filenameRegexp := regexp.MustCompile(`(T|P)\.(A\d{4})\.ACO\.ZC(0|8|9)Y\d{2}\.(D\d{6}\.T\d{6})\d`)
	filenameMatches := filenameRegexp.FindStringSubmatch(filePath)
	if len(filenameMatches) < 5 {
		return metadata, errors.New("invalid filename")
	}

	filenameDate := filenameMatches[4]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		return metadata, fmt.Errorf("failed to parse date '%s' from filename", filenameDate)
	}

	cclfNum, err := strconv.Atoi(filenameMatches[3])
	if err != nil {
		return metadata, err
	}

	if filenameMatches[1] == "T" {
		metadata.env = "test"
	} else if filenameMatches[1] == "P" {
		metadata.env = "production"
	}

	metadata.cclfNum = cclfNum
	metadata.acoID = filenameMatches[2]
	metadata.timestamp = t

	return metadata, nil
}

func importCCLFDirectory(filePath string) (success, failure, skipped int, err error) {
	var cclf0, cclf8, cclf9 []cclfFileMetadata

	err = filepath.Walk(filePath, sortCCLFFiles(&cclf0, &cclf8, &cclf9, &skipped))
	if err != nil {
		return 0, 0, 0, err
	}

	for _, file := range cclf0 {
		aco := file.acoID
		cclfValidator, err := importCCLF0(file)
		if err != nil {
			log.Error(err)
			return 0, 0, 0, err
		}

		for _, file = range cclf8 {
			if file.acoID == aco {
				err = importCCLF8(file, cclfValidator)
				if err != nil {
					log.Error(err)
					failure++
					continue
				}
				log.Info(fmt.Sprintf("Successfully imported CCLF8 file: %v", file))
				success++
			}
		}
		for _, file = range cclf9 {
			if file.acoID == aco {
				err = importCCLF9(file, cclfValidator)
				if err != nil {
					log.Error(err)
					failure++
					continue
				}
				log.Info(fmt.Sprintf("Successfully imported CCLF9 file: %v", file))
				success++
			}
		}
	}

	if failure > 0 {
		err = errors.New("one or more files failed to import correctly")
	} else {
		err = nil
	}
	return success, failure, skipped, err
}

func sortCCLFFiles(cclf0, cclf8, cclf9 *[]cclfFileMetadata, skipped *int) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Directories are not CCLF files
		if info.IsDir() {
			return nil
		}
		metadata, err := getCCLFFileMetadata(info.Name())
		metadata.filePath = path
		if err != nil {
			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			log.Error(err)
			*skipped = *skipped + 1
			return nil
		}
		if metadata.cclfNum == 0 {
			*cclf0 = append(*cclf0, metadata)
		} else if metadata.cclfNum == 8 {
			*cclf8 = append(*cclf8, metadata)
		} else if metadata.cclfNum == 9 {
			*cclf9 = append(*cclf9, metadata)
		}
		return nil
	}
}
