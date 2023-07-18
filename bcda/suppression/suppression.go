package suppression

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/log"
	"github.com/CMSgov/bcda-app/optout"

	"github.com/CMSgov/bcda-app/bcda/utils"

	"github.com/pkg/errors"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/conf"
)

const (
	headerCode  = "HDR_BENEDATASHR"
	trailerCode = "TRL_BENEDATASHR"
)

func ImportSuppressionDirectory(filePath string) (success, failure, skipped int, err error) {
	var suppresslist []*optout.OptOutFilenameMetadata

	err = filepath.Walk(filePath, getSuppressionFileMetadata(&suppresslist, &skipped))
	if err != nil {
		return 0, 0, 0, err
	}

	if len(suppresslist) == 0 {
		log.API.Info("Failed to find any suppression files in directory")
		return 0, 0, skipped, nil
	}

	for _, metadata := range suppresslist {
		err = validate(metadata)
		if err != nil {
			fmt.Printf("Failed to validate suppression file: %s.\n", metadata)
			log.API.Errorf("Failed to validate suppression file: %s", metadata)
			failure++
		} else {
			if err = importSuppressionData(metadata); err != nil {
				fmt.Printf("Failed to import suppression file: %s.\n", metadata)
				log.API.Errorf("Failed to import suppression file: %s ", metadata)
				failure++
			} else {
				metadata.Imported = true
				success++
			}
		}
	}
	err = cleanupSuppression(suppresslist)
	if err != nil {
		log.API.Error(err)
	}

	if failure > 0 {
		err = errors.New("one or more suppression files failed to import correctly")
		log.API.Error(err)
	} else {
		err = nil
	}
	return success, failure, skipped, err
}

func getSuppressionFileMetadata(suppresslist *[]*optout.OptOutFilenameMetadata, skipped *int) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			var fileName = "nil"
			if info != nil {
				fileName = info.Name()
			}
			fmt.Printf("Error in checking suppression file %s: %s.\n", fileName, err)
			err = errors.Wrapf(err, "error in checking suppression file: %s,", fileName)
			log.API.Error(err)
			return err
		}
		// Directories are not Suppression files
		if info.IsDir() {
			return nil
		}

		metadata, err := optout.ParseMetadata(info.Name())
		metadata.FilePath = path
		metadata.DeliveryDate = info.ModTime()
		if err != nil {
			log.API.Error(err)

			// skipping files with a bad name.  An unknown file in this dir isn't a blocker
			fmt.Printf("Unknown file found: %s.\n", metadata)
			log.API.Errorf("Unknown file found: %s", metadata)
			*skipped = *skipped + 1

			deleteThreshold := time.Hour * time.Duration(utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72))
			if metadata.DeliveryDate.Add(deleteThreshold).Before(time.Now()) {
				newpath := fmt.Sprintf("%s/%s", conf.GetEnv("PENDING_DELETION_DIR"), info.Name())
				err = os.Rename(metadata.FilePath, newpath)
				if err != nil {
					fmt.Printf("Error moving unknown file %s to pending deletion dir.\n", metadata)
					err = fmt.Errorf("error moving unknown file %s to pending deletion dir", metadata)
					log.API.Error(err)
					return err
				}
			}
			return nil
		}
		*suppresslist = append(*suppresslist, &metadata)
		return nil
	}
}

func validate(metadata *optout.OptOutFilenameMetadata) error {
	fmt.Printf("Validating suppression file %s...\n", metadata)
	log.API.Infof("Validating suppression file %s...", metadata)

	f, err := os.Open(metadata.FilePath)
	if err != nil {
		fmt.Printf("Could not read file %s.\n", metadata)
		err = errors.Wrapf(err, "could not read file %s", metadata)
		log.API.Error(err)
		return err
	}
	defer utils.CloseFileAndLogError(f)

	var (
		headTrailStart, headTrailEnd = 0, 15
		recCountStart, recCountEnd   = 23, 33
	)

	sc := bufio.NewScanner(f)
	count := 0
	for sc.Scan() {
		b := sc.Bytes()
		metaInfo := string(bytes.TrimSpace(b[headTrailStart:headTrailEnd]))
		if count == 0 {
			if metaInfo != headerCode {
				// invalid file header found
				fmt.Printf("Invalid file header for file: %s.\n", metadata.FilePath)
				err := fmt.Errorf("invalid file header for file: %s", metadata.FilePath)
				log.API.Error(err)
				return err
			}
			count++
			continue
		}

		if metaInfo != trailerCode {
			count++
		} else {
			// trailer info
			expectedCount, err := strconv.Atoi(string(bytes.TrimSpace(b[recCountStart:recCountEnd])))
			if err != nil {
				fmt.Printf("Failed to parse record count from file: %s.\n", metadata.FilePath)
				err = fmt.Errorf("failed to parse record count from file: %s", metadata.FilePath)
				log.API.Error(err)
				return err
			}
			// subtract the single count from the header
			count--
			if count != expectedCount {
				fmt.Printf("Incorrect number of records found from file: '%s'. Expected record count: %d, Actual record count: %d.\n", metadata.FilePath, expectedCount, count)
				err = fmt.Errorf("incorrect number of records found from file: '%s'. Expected record count: %d, Actual record count: %d", metadata.FilePath, expectedCount, count)
				log.API.Error(err)
				return err
			}
		}
	}
	fmt.Printf("Successfully validated suppression file %s.\n", metadata)
	log.API.Infof("Successfully validated suppression file %s.", metadata)
	return nil
}

func importSuppressionData(metadata *optout.OptOutFilenameMetadata) error {
	err := importSuppressionMetadata(metadata, func(fileID uint, b []byte, r models.Repository) error {
		suppression, err := optout.ParseSuppressionLine(metadata, b)

		if err != nil {
			log.API.Error(err)
			return err
		}

		if err = r.CreateSuppression(context.Background(), *suppression); err != nil {
			fmt.Println("Could not create suppression record.")
			err = errors.Wrap(err, "could not create suppression record")
			log.API.Error(err)
			return err
		}
		return nil
	})

	if err != nil {
		updateImportStatus(metadata.FileID, optout.ImportFail)
		return err
	}
	updateImportStatus(metadata.FileID, optout.ImportComplete)
	return nil
}

func importSuppressionMetadata(metadata *optout.OptOutFilenameMetadata, importFunc func(uint, []byte, models.Repository) error) error {
	fmt.Printf("Importing suppression file %s...\n", metadata)
	log.API.Infof("Importing suppression file %s...", metadata)

	var (
		headTrailStart, headTrailEnd = 0, 15
		err                          error
	)

	suppressionMetaFile := optout.OptOutFile{
		Name:         metadata.Name,
		Timestamp:    metadata.Timestamp,
		ImportStatus: optout.ImportInprog,
	}

	db := database.Connection

	r := postgres.NewRepository(db)

	if suppressionMetaFile.ID, err = r.CreateSuppressionFile(context.Background(), suppressionMetaFile); err != nil {
		fmt.Printf("Could not create suppression file record for file: %s. \n", metadata)
		err = errors.Wrapf(err, "could not create suppression file record for file: %s.", metadata)
		log.API.Error(err)
		return err
	}

	metadata.FileID = suppressionMetaFile.ID

	importStatusInterval := utils.GetEnvInt("SUPPRESS_IMPORT_STATUS_RECORDS_INTERVAL", 1000)
	importedCount := 0
	f, err := os.Open(metadata.FilePath)
	if err != nil {
		fmt.Printf("Could not read file %s.\n", metadata)
		err = errors.Wrapf(err, "could not read file %s", metadata)
		log.API.Error(err)
		return err
	}
	defer utils.CloseFileAndLogError(f)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			metaInfo := string(bytes.TrimSpace(b[headTrailStart:headTrailEnd]))
			if metaInfo == headerCode || metaInfo == trailerCode {
				continue
			}
			err = importFunc(suppressionMetaFile.ID, b, r)
			if err != nil {
				log.API.Error(err)
				return err
			}
			importedCount++
			if importedCount%importStatusInterval == 0 {
				fmt.Printf("Suppression records imported: %d\n", importedCount)
			}
		}
	}

	successMsg := fmt.Sprintf("Successfully imported %d records from suppression file %s.", importedCount, metadata)
	fmt.Println(successMsg)
	log.API.Infof(successMsg)

	return nil
}

func cleanupSuppression(suppresslist []*optout.OptOutFilenameMetadata) error {
	errCount := 0
	for _, suppressionFile := range suppresslist {
		fmt.Printf("Cleaning up file %s.\n", suppressionFile)
		log.API.Infof("Cleaning up file %s", suppressionFile)
		newpath := fmt.Sprintf("%s/%s", conf.GetEnv("PENDING_DELETION_DIR"), suppressionFile.Name)
		if !suppressionFile.Imported {
			// check the timestamp on the failed files
			elapsed := time.Since(suppressionFile.DeliveryDate).Hours()
			deleteThreshold := utils.GetEnvInt("BCDA_ETL_FILE_ARCHIVE_THRESHOLD_HR", 72)
			if int(elapsed) > deleteThreshold {
				err := os.Rename(suppressionFile.FilePath, newpath)
				if err != nil {
					errCount++
					errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", suppressionFile, err)
					fmt.Println(errMsg)
					log.API.Error(errMsg)
				} else {
					fmt.Printf("File %s never ingested, moved to the pending deletion dir.\n", suppressionFile)
					log.API.Infof("File %s never ingested, moved to the pending deletion dir", suppressionFile)
				}
			}
		} else {
			// move the successful files to the deletion dir
			err := os.Rename(suppressionFile.FilePath, newpath)
			if err != nil {
				errCount++
				errMsg := fmt.Sprintf("File %s failed to clean up properly: %v", suppressionFile, err)
				fmt.Println(errMsg)
				log.API.Error(errMsg)
			} else {
				fmt.Printf("File %s successfully ingested, moved to the pending deletion dir.\n", suppressionFile)
				log.API.Infof("File %s successfully ingested, moved to the pending deletion dir", suppressionFile)
			}
		}
	}
	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}
	return nil
}

func updateImportStatus(fileID uint, status string) {
	db := database.Connection
	r := postgres.NewRepository(db)

	if err := r.UpdateSuppressionFileImportStatus(context.Background(), fileID, status); err != nil {
		fmt.Printf("Could not update suppression file record for file_id: %d. \n", fileID)
		err = errors.Wrapf(err, "could not update suppression file record for file_id: %d.", fileID)
		log.API.Error(err)
	}
}
