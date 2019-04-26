package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/database"
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
	name         string
	env          string
	acoID        string
	cclfNum      int
	perfYear     int
	timestamp    time.Time
	filePath     string
	imported     bool
	deliveryDate time.Time
}

type cclfFileValidator struct {
	totalRecordCount int
	maxRecordLength  int
}

const deleteThresholdHr = 8

func importCCLF0(fileMetadata *cclfFileMetadata) (map[string]cclfFileValidator, error) {
	if fileMetadata == nil {
		err := errors.New("file CCLF0 not found")
		log.Error(err)
		return nil, err
	}

	fmt.Printf("Importing CCLF0 file %s...\n", fileMetadata.name)
	log.Infof("Importing CCLF0 file %s...", fileMetadata.name)

	file, err := os.Open(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		log.Error(err)
		return nil, err
	}

	const (
		fileNumStart, fileNumEnd           = 0, 13
		totalRecordStart, totalRecordEnd   = 35, 55
		recordLengthStart, recordLengthEnd = 56, 69
	)

	var validator map[string]cclfFileValidator
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
					log.Error(err)
					return nil, err
				}
				length, err := strconv.Atoi(string(bytes.TrimSpace(b[recordLengthStart:recordLengthEnd])))
				if err != nil {
					log.Error(err)
					return nil, err
				}
				validator[filetype] = cclfFileValidator{totalRecordCount: count, maxRecordLength: length}
			}
		}
	}

	if _, ok := validator["CCLF8"]; !ok {
		err := fmt.Errorf("failed to parse CCLF8 from CCLF0 file: %v", fileMetadata)
		log.Error(err)
		return nil, err
	}
	if _, ok := validator["CCLF9"]; !ok {
		err := fmt.Errorf("failed to parse CCLF9 from CCLF0 file: %v", fileMetadata)
		log.Error(err)
		return nil, err
	}
	fmt.Printf("Imported CCLF0 file %s.\n", fileMetadata.name)
	log.Infof("Imported CCLF0 file %s.", fileMetadata.name)

	return validator, nil
}

func importCCLF8(fileMetadata *cclfFileMetadata) error {
	if fileMetadata == nil {
		err := errors.New("file CCLF8 not found")
		log.Error(err)
		return err
	}

	fmt.Printf("Importing CCLF8 file %s...\n", fileMetadata.name)
	log.Infof("Importing CCLF8 file %s...", fileMetadata.name)

	if fileMetadata.cclfNum != 8 {
		err := fmt.Errorf("expected CCLF file number 8, but was %d", fileMetadata.cclfNum)
		log.Error(err)
		return err
	}

	file, err := os.Open(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		log.Error(err)
		return err
	}

	cclf8File := models.CCLFFile{
		CCLFNum:         8,
		Name:            fileMetadata.name,
		ACOCMSID:        fileMetadata.acoID,
		Timestamp:       fileMetadata.timestamp,
		PerformanceYear: fileMetadata.perfYear,
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err = db.Create(&cclf8File).Error
	if err != nil {
		return errors.Wrap(err, "could not create CCLF8 file record")
	}

	const (
		mbiStart, mbiEnd   = 0, 11
		hicnStart, hicnEnd = 11, 22
	)

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			err = db.Create(&models.CCLFBeneficiary{
				FileID: cclf8File.ID,
				MBI:    string(bytes.TrimSpace(b[mbiStart:mbiEnd])),
				HICN:   string(bytes.TrimSpace(b[hicnStart:hicnEnd])),
				//TODO: BeneficiaryID
			}).Error
			if err != nil {
				return errors.Wrap(err, "could not create CCLF8 beneficiary record")
			}
		}
	}

	fmt.Printf("Imported CCLF8 file %s.\n", fileMetadata.name)
	log.Infof("Imported CCLF8 file %s.", fileMetadata.name)

	return nil
}

func importCCLF9(fileMetadata *cclfFileMetadata) error {
	if fileMetadata == nil {
		err := errors.New("file CCLF9 not found")
		log.Error(err)
		return err
	}

	fmt.Printf("Importing CCLF9 file %s...\n", fileMetadata.name)
	log.Infof("Importing CCLF9 file %s...", fileMetadata.name)

	if fileMetadata.cclfNum != 9 {
		err := fmt.Errorf("expected CCLF file number 9, but was %d", fileMetadata.cclfNum)
		log.Error(err)
		return err
	}

	file, err := os.Open(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		log.Error(err)
		return err
	}

	cclf9File := models.CCLFFile{
		CCLFNum:         9,
		Name:            fileMetadata.name,
		ACOCMSID:        fileMetadata.acoID,
		Timestamp:       fileMetadata.timestamp,
		PerformanceYear: fileMetadata.perfYear,
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err = db.Create(&cclf9File).Error
	if err != nil {
		return errors.Wrap(err, "could not create CCLF9 file record")
	}

	const (
		currIDStart, currIDEnd               = 1, 12
		prevIDStart, prevIDEnd               = 12, 23
		prevIDEffDateStart, prevIDEffDateEnd = 23, 33
		prevIDObsDateStart, prevIDObsDateEnd = 33, 43
	)

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			cclf9 := models.CCLFBeneficiaryXref{
				FileID:        cclf9File.ID,
				XrefIndicator: string(b[0:1]),
				CurrentNum:    string(b[currIDStart:currIDEnd]),
				PrevNum:       string(b[prevIDStart:prevIDEnd]),
				PrevsEfctDt:   string(b[prevIDEffDateStart:prevIDEffDateEnd]),
				PrevsObsltDt:  string(b[prevIDObsDateStart:prevIDObsDateEnd]),
			}
			err = db.Create(&cclf9).Error
			if err != nil {
				return errors.Wrap(err, "could not create CCLF9 cross reference record")
			}
		}
	}
	fmt.Printf("Imported CCLF9 file %s.\n", fileMetadata.name)
	log.Infof("Imported CCLF9 file %s.", fileMetadata.name)

	return nil
}

func getCCLFFileMetadata(filePath string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	// CCLF filename convention for SSP: P.A****.ACO.ZC*Y**.Dyymmdd.Thhmmsst
	// Prefix: T = test, P = prod; A**** = ACO ID; ZC* = CCLF file number; Y** = performance year
	filenameRegexp := regexp.MustCompile(`(T|P)\.(A\d{4})\.ACO\.ZC(0|8|9)Y(\d{2})\.(D\d{6}\.T\d{6})\d`)
	filenameMatches := filenameRegexp.FindStringSubmatch(filePath)
	if len(filenameMatches) < 5 {
		return metadata, fmt.Errorf("invalid filename for file: %s", filePath)
	}

	cclfNum, err := strconv.Atoi(filenameMatches[3])
	if err != nil {
		return metadata, err
	}

	perfYear, err := strconv.Atoi(filenameMatches[4])
	if err != nil {
		return metadata, err
	}

	filenameDate := filenameMatches[5]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		return metadata, fmt.Errorf("failed to parse date '%s' from file: %s", filenameDate, filePath)
	}

	if filenameMatches[1] == "T" {
		metadata.env = "test"
	} else if filenameMatches[1] == "P" {
		metadata.env = "production"
	}

	metadata.name = filenameMatches[0]
	metadata.cclfNum = cclfNum
	metadata.acoID = filenameMatches[2]
	metadata.timestamp = t
	metadata.perfYear = perfYear

	return metadata, nil
}

func importCCLFDirectory(filePath string) (success, failure, skipped int, err error) {
	var cclfmap = make(map[string][]*cclfFileMetadata)

	err = filepath.Walk(filePath, sortCCLFFiles(&cclfmap, &skipped))
	if err != nil {
		return 0, 0, 0, err
	}

	if len(cclfmap) == 0 {
		return 0, 0, 0, errors.New("failed to find any CCLF files in directory")
	}

	for _, cclflist := range cclfmap {
		var cclf0, cclf8, cclf9 *cclfFileMetadata
		for _, cclf := range cclflist {
			if cclf.cclfNum == 0 {
				cclf0 = cclf
			} else if cclf.cclfNum == 8 {
				cclf8 = cclf
			} else if cclf.cclfNum == 9 {
				cclf9 = cclf
			}
		}
		cclfvalidator, err := importCCLF0(cclf0)
		if err != nil {
			log.Errorf("failed to import CCLF0 file: %v, Skipping CCLF8 file: %v and CCLF9 file: %v ", cclf0, cclf8, cclf9)
			failure++
			skipped += 2
			continue
		} else {
			log.Infof("Successfully imported CCLF0 file: %v", cclf0)
			cclf0.imported = true
			success++
		}
		err = validate(cclf8, cclfvalidator)
		if err != nil {
			log.Errorf("failed to validate CCLF8 file: %v", cclf8)
			failure++
		} else {
			if err = importCCLF8(cclf8); err != nil {
				log.Errorf("failed to import CCLF8 file: %v ", cclf8)
				failure++
			} else {
				log.Infof("Successfully imported CCLF8 file: %v", cclf8)
				cclf8.imported = true
				success++
			}
		}
		err = validate(cclf9, cclfvalidator)
		if err != nil {
			log.Errorf("failed to validate CCLF9 file: %v", cclf9)
			failure++
		} else {
			if err = importCCLF9(cclf9); err != nil {
				log.Errorf("failed to import CCLF9 file: %v ", cclf9)
				failure++
			} else {
				log.Infof(fmt.Sprintf("Successfully imported CCLF9 file: %v", cclf9))
				cclf9.imported = true
				success++
			}
		}
	}
	cleanupCCLF(cclfmap)

	if failure > 0 {
		err = errors.New("one or more files failed to import correctly")
	} else {
		err = nil
	}
	return success, failure, skipped, err
}

func sortCCLFFiles(cclfmap *map[string][]*cclfFileMetadata, skipped *int) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error(err)
			return err
		}
		// Directories are not CCLF files
		if info.IsDir() {
			return nil
		}
		metadata, err := getCCLFFileMetadata(info.Name())
		metadata.filePath = path
		metadata.deliveryDate = info.ModTime()
		if err != nil {
			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			log.Error(err)
			*skipped = *skipped + 1

			newpath := fmt.Sprintf("%s/%s", os.Getenv("PENDING_DELETION_DIR"), info.Name())
			err = os.Rename(metadata.filePath, newpath)
			if err != nil {
				log.Error(err)
				return err
			}
			return nil
		}

		// if we get multiple sets of files relating to the same aco for attribution purposes, concat the year
		key := fmt.Sprintf(metadata.acoID+"_%d", metadata.perfYear)
		if (*cclfmap)[key] != nil {
			(*cclfmap)[key] = append((*cclfmap)[key], &metadata)
		} else {
			(*cclfmap)[key] = []*cclfFileMetadata{&metadata}
		}
		return nil
	}
}

func validate(fileMetadata *cclfFileMetadata, cclfFileValidator map[string]cclfFileValidator) error {
	if fileMetadata == nil {
		err := errors.New("file not found")
		log.Error(err)
		return err
	}

	file, err := os.Open(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		log.Error(err)
		return err
	}

	sc := bufio.NewScanner(file)
	count := 0
	var key string

	if fileMetadata.cclfNum == 8 {
		key = "CCLF8"
	} else if fileMetadata.cclfNum == 9 {
		key = "CCLF9"
	} else {
		err := fmt.Errorf("unknown file type when validating file: %v,", fileMetadata)
		log.Error(err)
		return err
	}

	validator := cclfFileValidator[key]

	for sc.Scan() {
		b := sc.Bytes()
		bytelength := len(bytes.TrimSpace(b))
		if bytelength > 0 && bytelength <= validator.maxRecordLength {
			count++

			// currently only errors if there are more records than we expect.
			if count > validator.totalRecordCount {
				err := fmt.Errorf("maximum record count reached for file %s, Expected record count: %d, Actual record count: %d ", key, validator.totalRecordCount, count)
				log.Error(err)
				return err
			}
		} else {
			err := fmt.Errorf("incorrect record length for file %s, Expected record length: %d, Actual record length: %d", key, validator.maxRecordLength, bytelength)
			log.Error(err)
			return err
		}
	}
	return nil
}

func deleteDirectoryContents(dirToDelete string) (filesDeleted int, err error) {
	log.Info(fmt.Sprintf("preparing to delete directory '%v'", dirToDelete))
	f, err := os.Open(filepath.Clean(dirToDelete))
	if err != nil {
		return 0, err
	}
	files, err := f.Readdir(-1)
	if err != nil {
		return 0, err
	}
	err = f.Close()
	if err != nil {
		return 0, err
	}

	for _, file := range files {
		log.Info(fmt.Sprintf("deleting %v", file.Name()))
		err = os.Remove(filepath.Join(dirToDelete, file.Name()))
		if err != nil {
			return 0, err
		}
	}

	return len(files), nil
}

func cleanupCCLF(cclfmap map[string][]*cclfFileMetadata) {
	for _, cclflist := range cclfmap {
		for _, cclf := range cclflist {
			newpath := fmt.Sprintf("%s/%s", os.Getenv("PENDING_DELETION_DIR"), cclf.name)
			if !cclf.imported {
				// check the timestamp on the failed files
				elapsed := time.Since(cclf.deliveryDate).Hours()
				if int(elapsed) > (24 - deleteThresholdHr) {
					err := os.Rename(cclf.filePath, newpath)
					if err != nil {
						err := fmt.Errorf("File: %v failed to cleanup properly", cclf)
						log.Error(err)
					}
				}
			} else {
				// move the successful files to the deletion dir
				err := os.Rename(cclf.filePath, newpath)
				if err != nil {
					err := fmt.Errorf("File: %v failed to cleanup properly", cclf)
					log.Error(err)
				}
			}
		}
	}
}
