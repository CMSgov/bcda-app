package cclf

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

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
	fileID       uint
}

type cclfFileValidator struct {
	totalRecordCount int
	maxRecordLength  int
}

func importCCLF0(fileMetadata *cclfFileMetadata) (map[string]cclfFileValidator, error) {
	if fileMetadata == nil {
		fmt.Println("File CCLF0 not found.")
		err := errors.New("file CCLF0 not found")
		log.Error(err)
		return nil, err
	}

	fmt.Printf("Importing CCLF0 file %s...\n", fileMetadata)
	log.Infof("Importing CCLF0 file %s...", fileMetadata)

	r, err := zip.OpenReader(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		fmt.Printf("Could not read CCLF0 archive %s.\n", fileMetadata)
		err := errors.Wrapf(err, "could not read CCLF0 archive %s", fileMetadata)
		log.Error(err)
		return nil, err
	}
	defer r.Close()

	const (
		fileNumStart, fileNumEnd           = 0, 7
		totalRecordStart, totalRecordEnd   = 52, 63
		recordLengthStart, recordLengthEnd = 64, 69
	)

	var validator map[string]cclfFileValidator
	var rawFile *zip.File

	for _, f := range r.File {
		// iterate in this zipped folder until we find our cclf0 file
		if f.Name == fileMetadata.name {
			rawFile = f
			fmt.Printf("Reading file %s from archive %s.\n", fileMetadata.name, fileMetadata.filePath)
			log.Infof("Reading file %s from archive %s", fileMetadata.name, fileMetadata.filePath)
		}
	}

	if rawFile == nil {
		fmt.Printf("File %s not found in archive %s.\n", fileMetadata.name, fileMetadata.filePath)
		err = errors.Wrapf(err, "file %s not found in archive %s", fileMetadata.name, fileMetadata.filePath)
		log.Error(err)
		return nil, err
	}

	rc, err := rawFile.Open()
	if err != nil {
		fmt.Printf("Could not read file %s in CCLF0 archive %s.\n", fileMetadata.name, fileMetadata.filePath)
		err = errors.Wrapf(err, "could not read file %s in CCLF0 archive %s", fileMetadata.name, fileMetadata.filePath)
		log.Error(err)
		return nil, err
	}
	defer rc.Close()
	sc := bufio.NewScanner(rc)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			filetype := string(bytes.TrimSpace(b[fileNumStart:fileNumEnd]))

			if filetype == "CCLF8" {
				if validator == nil {
					validator = make(map[string]cclfFileValidator)
				}

				if _, ok := validator[filetype]; ok {
					fmt.Printf("Duplicate %v file type found from CCLF0 file.\n", filetype)
					err := fmt.Errorf("duplicate %v file type found from CCLF0 file", filetype)
					log.Error(err)
					return nil, err
				}

				count, err := strconv.Atoi(string(bytes.TrimSpace(b[totalRecordStart:totalRecordEnd])))
				if err != nil {
					fmt.Printf("Failed to parse %s record count from CCLF0 file.\n", filetype)
					err = errors.Wrapf(err, "failed to parse %s record count from CCLF0 file", filetype)
					log.Error(err)
					return nil, err
				}
				length, err := strconv.Atoi(string(bytes.TrimSpace(b[recordLengthStart:recordLengthEnd])))
				if err != nil {
					fmt.Printf("Failed to parse %s record length from CCLF0 file.\n", filetype)
					err = errors.Wrapf(err, "failed to parse %s record length from CCLF0 file", filetype)
					log.Error(err)
					return nil, err
				}
				validator[filetype] = cclfFileValidator{totalRecordCount: count, maxRecordLength: length}
			}
		}
	}

	if _, ok := validator["CCLF8"]; !ok {
		fmt.Printf("Failed to parse CCLF8 from CCLF0 file %s.\n", fileMetadata)
		err := fmt.Errorf("failed to parse CCLF8 from CCLF0 file %s", fileMetadata)
		log.Error(err)
		return nil, err
	}
	fmt.Printf("Successfully imported CCLF0 file %s.\n", fileMetadata)
	log.Infof("Successfully imported CCLF0 file %s.", fileMetadata)

	return validator, nil
}

func importCCLF8(fileMetadata *cclfFileMetadata) error {
	err := importCCLF(fileMetadata, func(fileID uint, b []byte, db *gorm.DB) error {
		const (
			mbiStart, mbiEnd   = 0, 11
			hicnStart, hicnEnd = 11, 22
		)
		cclfBeneficiary := &models.CCLFBeneficiary{
			FileID: fileID,
			MBI:    string(bytes.TrimSpace(b[mbiStart:mbiEnd])),
			HICN:   string(bytes.TrimSpace(b[hicnStart:hicnEnd])),
		}
		err := db.Create(cclfBeneficiary).Error
		if err != nil {
			fmt.Println("Could not create CCLF8 beneficiary record.")
			err = errors.Wrap(err, "could not create CCLF8 beneficiary record")
			log.Error(err)
			return err
		}
		return nil
	})

	if err != nil {
		updateImportStatus(fileMetadata, constants.ImportFail)
		return err
	}
	updateImportStatus(fileMetadata, constants.ImportComplete)
	return nil
}

func importCCLF(fileMetadata *cclfFileMetadata, importFunc func(uint, []byte, *gorm.DB) error) error {
	if fileMetadata == nil {
		fmt.Println("CCLF file not found.")
		err := errors.New("CCLF file not found")
		log.Error(err)
		return err
	}

	fmt.Printf("Importing CCLF%d file %s...\n", fileMetadata.cclfNum, fileMetadata)
	log.Infof("Importing CCLF%d file %s...", fileMetadata.cclfNum, fileMetadata)

	r, err := zip.OpenReader(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		fmt.Printf("Could not read CCLF%d archive %s.\n", fileMetadata.cclfNum, fileMetadata.filePath)
		err := errors.Wrapf(err, "could not read CCLF%d archive %s", fileMetadata.cclfNum, fileMetadata.filePath)
		log.Error(err)
		return err
	}
	defer r.Close()

	if len(r.File) < 1 {
		fmt.Printf("No files found in CCLF%d archive %s.\n", fileMetadata.cclfNum, fileMetadata.filePath)
		err := fmt.Errorf("no files found in CCLF%d archive %s", fileMetadata.cclfNum, fileMetadata.filePath)
		log.Error(err)
		return err
	}

	cclfFile := models.CCLFFile{
		CCLFNum:         fileMetadata.cclfNum,
		Name:            fileMetadata.name,
		ACOCMSID:        fileMetadata.acoID,
		Timestamp:       fileMetadata.timestamp,
		PerformanceYear: fileMetadata.perfYear,
		ImportStatus:    constants.ImportInprog,
	}

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err = db.Create(&cclfFile).Error
	if err != nil {
		fmt.Printf("Could not create CCLF%d file record.\n", fileMetadata.cclfNum)
		err = errors.Wrapf(err, "could not create CCLF%d file record", fileMetadata.cclfNum)
		log.Error(err)
		return err
	}

	fileMetadata.fileID = cclfFile.ID

	importStatusInterval := utils.GetEnvInt("CCLF_IMPORT_STATUS_RECORDS_INTERVAL", 1000)
	importedCount := 0
	var rawFile *zip.File

	for _, f := range r.File {
		if f.Name == fileMetadata.name {
			rawFile = f
			fmt.Printf("Reading file %s from archive %s.\n", fileMetadata.name, fileMetadata.filePath)
			log.Infof("Reading file %s from archive %s", fileMetadata.name, fileMetadata.filePath)
		}
	}

	if rawFile == nil {
		fmt.Printf("File %s not found in archive %s.\n", fileMetadata.name, fileMetadata.filePath)
		err = errors.Wrapf(err, "file %s not found in archive %s", fileMetadata.name, fileMetadata.filePath)
		log.Error(err)
		return err
	}

	rc, err := rawFile.Open()
	if err != nil {
		fmt.Printf("Could not read file %s for CCLF%d in archive %s.\n", cclfFile.Name, fileMetadata.cclfNum, fileMetadata.filePath)
		err = errors.Wrapf(err, "could not read file %s for CCLF%d in archive %s", cclfFile.Name, fileMetadata.cclfNum, fileMetadata.filePath)
		log.Error(err)
		return err
	}
	defer rc.Close()
	sc := bufio.NewScanner(rc)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			err = importFunc(cclfFile.ID, b, db)
			if err != nil {
				log.Error(err)
				return err
			}
			importedCount++
			if importedCount%importStatusInterval == 0 {
				fmt.Printf("CCLF%d records imported: %d\n", fileMetadata.cclfNum, importedCount)
			}
		}
	}

	successMsg := fmt.Sprintf("Successfully imported %d records from CCLF%d file %s.", importedCount, fileMetadata.cclfNum, fileMetadata)
	fmt.Println(successMsg)
	log.Infof(successMsg)

	return nil
}

func getCCLFFileMetadata(fileName string) (cclfFileMetadata, error) {
	var metadata cclfFileMetadata
	// CCLF filename convention for SSP with BCD identifier: P.BCD.A****.ZC[0|8][Y]**.Dyymmdd.Thhmmsst
	filenameRegexp := regexp.MustCompile(`(T|P)\.BCD\.((?:A|T)\d{4})\.ZC(0|8)Y(\d{2})\.(D\d{6}\.T\d{6})\d`)
	filenameMatches := filenameRegexp.FindStringSubmatch(fileName)

	if len(filenameMatches) < 5 {
		fmt.Printf("Invalid filename for file: %s.\n", fileName)
		err := fmt.Errorf("invalid filename for file: %s", fileName)
		log.Error(err)
		return metadata, err
	}

	cclfNum, err := strconv.Atoi(filenameMatches[3])
	if err != nil {
		fmt.Printf("Failed to parse CCLF number from file: %s.\n", fileName)
		err = errors.Wrapf(err, "failed to parse CCLF number from file: %s", fileName)
		log.Error(err)
		return metadata, err
	}

	perfYear, err := strconv.Atoi(filenameMatches[4])
	if err != nil {
		fmt.Printf("Failed to parse performance year from file: %s.\n", fileName)
		err = errors.Wrapf(err, "failed to parse performance year from file: %s", fileName)
		log.Error(err)
		return metadata, err
	}

	filenameDate := filenameMatches[5]
	t, err := time.Parse("D060102.T150405", filenameDate)
	if err != nil || t.IsZero() {
		fmt.Printf("Failed to parse date '%s' from file: %s.\n", filenameDate, fileName)
		err = errors.Wrapf(err, "failed to parse date '%s' from file: %s", filenameDate, fileName)
		log.Error(err)
		return metadata, err
	}

	maxFileDays := utils.GetEnvInt("CCLF_MAX_AGE", 45)
	refDateString := os.Getenv("CCLF_REF_DATE")
	refDate, err := time.Parse("060102", refDateString)
	if err != nil {
		refDate = time.Now()
	}

	// Files must not be too old
	filesNotBefore := refDate.Add(-1 * time.Duration(int64(maxFileDays*24)*int64(time.Hour)))
	filesNotAfter := refDate
	if t.Before(filesNotBefore) || t.After(filesNotAfter) {
		fmt.Printf("Date '%s' from file %s is out of range; comparison date %s\n", filenameDate, fileName, refDate.Format("060102"))
		err = errors.New(fmt.Sprintf("date '%s' from file %s out of range; comparison date %s", filenameDate, fileName, refDate.Format("060102")))
		log.Error(err)
		return metadata, err
	}

	acoID := filenameMatches[2]
	if len(acoID) < 5 {
		fmt.Printf("Failed to parse aco id '%s' from file: %s.\n", acoID, fileName)
		err = errors.Wrapf(err, "failed to parse aco id '%s' from file: %s", acoID, fileName)
		log.Error(err)
		return metadata, err
	}

	if filenameMatches[1] == "T" {
		metadata.env = "test"
	} else if filenameMatches[1] == "P" {
		metadata.env = "production"
	}

	metadata.name = filenameMatches[0]
	metadata.cclfNum = cclfNum
	metadata.acoID = acoID
	metadata.timestamp = t
	metadata.perfYear = perfYear

	return metadata, nil
}

func ImportCCLFDirectory(filePath string) (success, failure, skipped int, err error) {
	var cclfMap = make(map[string]map[int][]*cclfFileMetadata)

	err = filepath.Walk(filePath, sortCCLFArchives(&cclfMap, &skipped))
	if err != nil {
		return 0, 0, 0, err
	}

	if len(cclfMap) == 0 {
		log.Info("Failed to find any CCLF files in directory")
		return 0, 0, skipped, nil
	}

	acoOrder := orderACOs(&cclfMap)

	for _, acoID := range acoOrder {
		for _, cclfFiles := range cclfMap[acoID] {
			var cclf0, cclf8 *cclfFileMetadata
			for _, cclf := range cclfFiles {
				if cclf.cclfNum == 0 {
					cclf0 = cclf
				} else if cclf.cclfNum == 8 {
					cclf8 = cclf
				}
			}
			cclfvalidator, err := importCCLF0(cclf0)
			if err != nil {
				fmt.Printf("Failed to import CCLF0 file: %s, Skipping CCLF8 file: %s.\n ", cclf0, cclf8)
				log.Errorf("Failed to import CCLF0 file: %s, Skipping CCLF8 file: %s ", cclf0, cclf8)
				failure++
				skipped += 2
				continue
			} else {
				success++
			}
			err = validate(cclf8, cclfvalidator)
			if err != nil {
				fmt.Printf("Failed to validate CCLF8 file: %s.\n", cclf8)
				log.Errorf("Failed to validate CCLF8 file: %s", cclf8)
				failure++
			} else {
				if err = importCCLF8(cclf8); err != nil {
					fmt.Printf("Failed to import CCLF8 file: %s.\n", cclf8)
					log.Errorf("Failed to import CCLF8 file: %s ", cclf8)
					failure++
				} else {
					cclf8.imported = true
					success++
				}
			}
			cclf0.imported = cclf8 != nil && cclf8.imported
		}
	}

	err = cleanUpCCLF(cclfMap)
	if err != nil {
		log.Error(err)
	}

	if failure > 0 {
		err = errors.New("one or more files failed to import correctly")
		log.Error(err)
	} else {
		err = nil
	}
	return success, failure, skipped, err
}

func sortCCLFArchives(cclfMap *map[string]map[int][]*cclfFileMetadata, skipped *int) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			var fileName = "nil"
			if info != nil {
				fileName = info.Name()
			}
			fmt.Printf("Error in sorting CCLF file %s: %s.\n", fileName, err)
			err = errors.Wrapf(err, "error in sorting cclf file: %v,", fileName)
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
			*skipped = *skipped + 1
			msg := fmt.Sprintf("Skipping %s: file is not a CCLF archive.", path)
			fmt.Println(msg)
			log.Warn(msg)
			return nil
		}
		_ = zipReader.Close()

		// validate the top level zipped folder
		err = validateCCLFFolderName(info.Name()); if err != nil {
			*skipped = *skipped + 1
			msg := fmt.Sprintf("Skipping CCLF archive: %s.", info.Name())
			fmt.Println(msg)
			log.Warn(msg)
			err = checkDeliveryDate(path, info.ModTime())
			if err != nil {
				fmt.Printf("Error moving unknown file %s to pending deletion dir.\n", path)
				err = fmt.Errorf("error moving unknown file %s to pending deletion dir", path)
				log.Error(err)
				return err
			}
			return nil
		}

		for _, f := range zipReader.File {
			metadata, err := getCCLFFileMetadata(f.Name)
			metadata.filePath = path
			metadata.deliveryDate = info.ModTime()

			if err != nil {
				// skipping files with a bad name.  An unknown file in this dir isn't a blocker
				fmt.Printf("Unknown file found: %s.\n", f.Name)
				log.Errorf("Unknown file found: %s", f.Name)
				continue
			}

			if (*cclfMap)[metadata.acoID] != nil {
				if (*cclfMap)[metadata.acoID][metadata.perfYear] != nil {
					(*cclfMap)[metadata.acoID][metadata.perfYear] = append((*cclfMap)[metadata.acoID][metadata.perfYear], &metadata)
				} else {
					(*cclfMap)[metadata.acoID][metadata.perfYear] = []*cclfFileMetadata{&metadata}
				}
			} else {
				(*cclfMap)[metadata.acoID] = map[int][]*cclfFileMetadata{metadata.perfYear: []*cclfFileMetadata{&metadata}}
			}
		}
		return nil
	}
}

func checkDeliveryDate(folderPath string, deliveryDate time.Time) error {
	deleteThreshold := time.Hour * time.Duration(utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72))
	if deliveryDate.Add(deleteThreshold).Before(time.Now()) {
		folderName := filepath.Base(folderPath)
		newpath := fmt.Sprintf("%s/%s", os.Getenv("PENDING_DELETION_DIR"), folderName)
		err := os.Rename(folderPath, newpath)
		if err != nil {
			return err
		}
	}
	return nil
}

func orderACOs(cclfMap *map[string]map[int][]*cclfFileMetadata) []string {
	var acoOrder []string

	if priorityACOList := os.Getenv("CCLF_PRIORITY_ACO_CMS_IDS"); priorityACOList != "" {
		priorityACOs := strings.Split(priorityACOList, ",")
		for _, acoID := range priorityACOs {
			acoID = strings.TrimSpace(acoID)
			if (*cclfMap)[acoID] != nil {
				acoOrder = append(acoOrder, acoID)
			}
		}
	}

	for acoID := range *cclfMap {
		if !utils.ContainsString(acoOrder, acoID) {
			acoOrder = append(acoOrder, acoID)
		}
	}

	return acoOrder
}

func validate(fileMetadata *cclfFileMetadata, cclfFileValidator map[string]cclfFileValidator) error {
	if fileMetadata == nil {
		fmt.Printf("File not found.\n")
		err := errors.New("file not found")
		log.Error(err)
		return err
	}

	fmt.Printf("Validating CCLF%d file %s...\n", fileMetadata.cclfNum, fileMetadata)
	log.Infof("Validating CCLF%d file %s...", fileMetadata.cclfNum, fileMetadata)

	var key string
	if fileMetadata.cclfNum == 8 {
		key = "CCLF8"
	} else {
		fmt.Printf("Unknown file type when validating file: %s.\n", fileMetadata)
		err := fmt.Errorf("unknown file type when validating file: %s", fileMetadata)
		log.Error(err)
		return err
	}

	r, err := zip.OpenReader(filepath.Clean(fileMetadata.filePath))
	if err != nil {
		fmt.Printf("Could not read archive %s.\n", fileMetadata.filePath)
		err := errors.Wrapf(err, "could not read archive %s", fileMetadata.filePath)
		log.Error(err)
		return err
	}
	defer r.Close()

	count := 0
	validator := cclfFileValidator[key]
	var rawFile *zip.File

	for _, f := range r.File {
		if f.Name == fileMetadata.name {
			rawFile = f
			fmt.Printf("Reading file %s from archive %s.\n", fileMetadata.name, fileMetadata.filePath)
			log.Infof("Reading file %s from archive %s", fileMetadata.name, fileMetadata.filePath)
		}
	}

	if rawFile == nil {
		fmt.Printf("File %s not found in archive %s.\n", fileMetadata.name, fileMetadata.filePath)
		err = errors.Wrapf(err, "file %s not found in archive %s", fileMetadata.name, fileMetadata.filePath)
		log.Error(err)
		return err
	}

	rc, err := rawFile.Open()
	if err != nil {
		fmt.Printf("Could not read file %s in archive %s.\n", fileMetadata.name, fileMetadata.filePath)
		err = errors.Wrapf(err, "could not read file %s in archive %s", fileMetadata.name, fileMetadata.filePath)
		log.Error(err)
		return err
	}
	defer rc.Close()
	sc := bufio.NewScanner(rc)
	for sc.Scan() {
		b := sc.Bytes()
		bytelength := len(bytes.TrimSpace(b))
		if bytelength > 0 && bytelength <= validator.maxRecordLength {
			count++

			// currently only errors if there are more records than we expect.
			if count > validator.totalRecordCount {
				fmt.Printf("Maximum record count reached for file %s, Expected record count: %d, Actual record count: %d.\n", key, validator.totalRecordCount, count)
				err := fmt.Errorf("maximum record count reached for file %s (expected: %d, actual: %d)", key, validator.totalRecordCount, count)
				log.Error(err)
				return err
			}
		} else {
			fmt.Printf("Incorrect record length for file %s, Expected record length: %d, Actual record length: %d.\n", key, validator.maxRecordLength, bytelength)
			err := fmt.Errorf("incorrect record length for file %s (expected: %d, actual: %d)", key, validator.maxRecordLength, bytelength)
			log.Error(err)
			return err
		}
	}
	fmt.Printf("Successfully validated CCLF%d file %s.\n", fileMetadata.cclfNum, fileMetadata)
	log.Infof("Successfully validated CCLF%d file %s.", fileMetadata.cclfNum, fileMetadata)
	return nil
}

func validateCCLFFolderName(folderName string) error {
	// CCLF foldername convention for SSP with BCD identifier: P.BCD.A****.ZCY**.Dyymmdd.Thhmmsst
	folderNameRegexp := regexp.MustCompile(`(T|P)\.BCD\.((?:A|T)\d{4})\.ZCY(\d{2})\.(D\d{6})\.T(\d{4})\d{3}`)
	valid := folderNameRegexp.MatchString(folderName)
	if !valid {
		fmt.Printf("Invalid foldername for CCLF archive: %s.\n", folderName)
		err := fmt.Errorf("invalid foldername for CCLF archive: %s", folderName)
		log.Error(err)
		return err
	}
	return nil
}

func cleanUpCCLF(cclfMap map[string]map[int][]*cclfFileMetadata) error {
	errCount := 0
	for _, perfYearCCLFFileList := range cclfMap {
		for _, cclfFileList := range perfYearCCLFFileList {
			for _, cclf := range cclfFileList {
				fmt.Printf("Cleaning up file %s.\n", cclf.filePath)
				log.Infof("Cleaning up file %s", cclf.filePath)
				folderName := filepath.Base(cclf.filePath)
				newpath := fmt.Sprintf("%s/%s", os.Getenv("PENDING_DELETION_DIR"), folderName)
				if !cclf.imported {
					// check the timestamp on the failed files
					elapsed := time.Since(cclf.deliveryDate).Hours()
					deleteThreshold := utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72)
					if int(elapsed) > deleteThreshold {
						if _, err := os.Stat(newpath); err == nil {
							break
						}
						// move the (un)successful files to the deletion dir
						err := os.Rename(cclf.filePath, newpath)
						if err != nil {
							errCount++
							errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", cclf.filePath, err)
							fmt.Println(errMsg)
							log.Error(errMsg)
						} else {
							fmt.Printf("File %s never ingested, moved to the pending deletion dir.\n", cclf.filePath)
							log.Infof("File %s never ingested, moved to the pending deletion dir", cclf.filePath)
						}
					}
				} else {
					if _, err := os.Stat(newpath); err == nil {
						break
					}
					// move the successful files to the deletion dir
					err := os.Rename(cclf.filePath, newpath)
					if err != nil {
						errCount++
						errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", cclf.filePath, err)
						fmt.Println(errMsg)
						log.Error(errMsg)
					} else {
						fmt.Printf("File %s successfully ingested, moved to the pending deletion dir.\n", cclf.filePath)
						log.Infof("File %s successfully ingested, moved to the pending deletion dir", cclf.filePath)
					}
				}
			}
		}
	}
	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}
	return nil
}

func (m cclfFileMetadata) String() string {
	if m.name != "" {
		return m.name
	}
	return m.filePath
}

func updateImportStatus(m *cclfFileMetadata, status string) {
	if m == nil {
		return
	}
	var cclfFile models.CCLFFile

	db := database.GetGORMDbConnection()
	defer database.Close(db)

	err := db.Model(&cclfFile).Where("id = ?", m.fileID).Update("import_status", status).Error
	if err != nil {
		fmt.Printf("Could not update cclf file record for file: %s. \n", m)
		err = errors.Wrapf(err, "could not update cclf file record for file: %s.", m)
		log.Error(err)
	}
}
