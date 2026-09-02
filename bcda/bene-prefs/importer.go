package beneprefs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"

	bcdaaws "github.com/CMSgov/bcda-app/bcda/aws"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/sirupsen/logrus"
)

const (
	headerCode  = "HDR_BENEDATASHR"
	trailerCode = "TRL_BENEDATASHR"
)

const (
	headTrailStart, headTrailEnd = 0, 15
	recCountStart, recCountEnd   = 23, 33
)

type BenePrefsImporter struct {
	FileClient           bcdaaws.CustomS3Client
	Repo                 models.Repository
	Logger               logrus.FieldLogger
	ImportStatusInterval int
}

// ImportDirectory takes a dir path and processes all bene-prefs files, creating a suppression_files db record for each file and a suppressions db record for each entry in that file.
// returns the number of files successfully imported, the number of files that failed to import, and the number of files that were skipped (not bene-prefs files).
func (importer BenePrefsImporter) ImportDirectory(ctx context.Context, path string) (success, failure, skipped int, err error) {
	suppresslist, skipped, err := importer.loadBenePrefsFiles(ctx, path)
	if err != nil {
		return 0, 0, 0, err
	}

	if len(*suppresslist) == 0 {
		importer.Logger.Error("failed to find any bene-prefs files in directory")
		return 0, 0, skipped, nil
	}

	// validate then import each file
	for _, metadata := range *suppresslist {
		err = importer.validate(ctx, metadata)
		if err != nil {
			importer.Logger.Errorf("failed to validate bene-prefs file: %s", metadata)
			failure++
		} else {
			if err = importer.importFile(ctx, metadata); err != nil {
				importer.Logger.Errorf("failed to import bene-prefs file: %s ", metadata)
				failure++
			} else {
				metadata.Imported = true
				success++
			}
		}
	}

	err = importer.cleanupBenePrefsFiles(ctx, *suppresslist)
	if err != nil {
		importer.Logger.Error(err)
	}

	if failure > 0 {
		err = errors.New("one or more bene-prefs files failed to import correctly")
		importer.Logger.Error(err)
	} else {
		err = nil
	}

	return success, failure, skipped, err
}

// validate scans a bene prefs file by checking header and trailer codes and ensuring the record count matches the number of records in the file
func (importer BenePrefsImporter) validate(ctx context.Context, metadata *models.BenePrefsFilenameMetadata) error {
	importer.Logger.Infof("validating bene-prefs file %s...", metadata)

	count := 0
	sc, close, err := bcdaaws.OpenFileAsScanner(ctx, importer.FileClient, metadata.FilePath)
	if err != nil {
		err = fmt.Errorf("could not read file %s, err: %w", metadata, err)
		importer.Logger.Error(err)
		return err
	}

	defer close()

	for sc.Scan() {
		b := sc.Bytes()
		metaInfo := string(bytes.TrimSpace(b[headTrailStart:headTrailEnd]))
		if count == 0 {
			if metaInfo != headerCode {
				err := fmt.Errorf("invalid file header for file: %s", metadata.FilePath)
				importer.Logger.Error(err)
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
				err = fmt.Errorf("failed to parse record count from file: %s", metadata.FilePath)
				importer.Logger.Error(err)
				return err
			}
			// subtract the single count from the header
			count--
			if count != expectedCount {
				err = fmt.Errorf("incorrect number of records found from file: '%s'. Expected record count: %d, Actual record count: %d", metadata.FilePath, expectedCount, count)
				importer.Logger.Error(err)
				return err
			}
		}
	}

	importer.Logger.Infof("Successfully validated bene-prefs file %s.", metadata)
	return nil
}

// importFile handles importing the file, creating the file record, then processing individual entries
func (importer BenePrefsImporter) importFile(ctx context.Context, metadata *models.BenePrefsFilenameMetadata) error {
	importer.Logger.Infof("importing bene-prefs file %s...", metadata)

	err := importer.createBenePrefsFileRecord(ctx, metadata)
	if err != nil {
		err = fmt.Errorf("failed to create bene-prefs file record for file: %s, err: %w", metadata, err)
		importer.Logger.Error(err)
		return err
	}

	err = importer.scanAndImport(ctx, metadata)
	if err != nil {
		err := fmt.Errorf("error scanning and importing records from file: %s, err: %w", metadata, err)
		importer.Logger.Error(err)

		repoErr := importer.Repo.UpdateBenePrefsImportStatus(ctx, metadata.FileID, constants.ImportFail)
		if repoErr != nil {
			repoErrMsg := fmt.Errorf("could not update bene-prefs file import status for file: %s, err: %w", metadata, repoErr)
			importer.Logger.Error(repoErrMsg)
		}
		return err
	}

	importer.Logger.Infof("Successfully imported file: %s", metadata.Name)

	err = importer.Repo.UpdateBenePrefsImportStatus(ctx, metadata.FileID, constants.ImportComplete)
	if err != nil {
		err = fmt.Errorf("could not update bene-prefs file import status for file: %s, err: %w", metadata, err)
		importer.Logger.Error(err)
		return err
	}

	return nil
}

// createBenePrefsFileRecord creates the suppression_files db record and updates the metadata with returned file_id
func (importer BenePrefsImporter) createBenePrefsFileRecord(ctx context.Context, metadata *models.BenePrefsFilenameMetadata) error {
	var err error

	bpFile := models.BenePrefsFile{
		Name:         metadata.Name,
		Timestamp:    metadata.Timestamp,
		ImportStatus: constants.ImportInprog,
	}

	bpFile.ID, err = importer.Repo.CreateBenePrefsFile(ctx, bpFile)
	if err != nil {
		errMsg := fmt.Errorf("could not create bene-prefs file record for file: %s, err: %w", metadata, err)
		importer.Logger.Error(errMsg)
		return err
	}

	metadata.FileID = bpFile.ID

	return nil
}

// scanAndImport scans the file and creates a suppression record for each entry in the file
func (importer BenePrefsImporter) scanAndImport(ctx context.Context, metadata *models.BenePrefsFilenameMetadata) error {
	var (
		headTrailStart, headTrailEnd = 0, 15
		err                          error
	)

	importedCount := 0

	sc, close, err := bcdaaws.OpenFileAsScanner(ctx, importer.FileClient, metadata.FilePath)
	if err != nil {
		err = fmt.Errorf("could not read file %s, err: %w", metadata, err)
		importer.Logger.Error(err)
		return err
	}
	defer close()

	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			metaInfo := string(bytes.TrimSpace(b[headTrailStart:headTrailEnd]))
			if metaInfo == headerCode || metaInfo == trailerCode {
				continue
			}
			err = importer.createBenePrefsRecord(ctx, metadata, b)
			if err != nil {
				importer.Logger.Error(err)
				return err
			}
			importedCount++
			if importedCount%importer.ImportStatusInterval == 0 {
				importer.Logger.Infof("Suppression records imported: %d\n", importedCount)
			}
		}
	}
	if err := sc.Err(); err != nil {
		importer.Logger.Errorf("error encountered during scanning: %v", err)
		return err
	}

	importer.Logger.Infof("Successfully imported %d records from bene-prefs file %s.", importedCount, metadata)

	return nil
}

// createBenePrefsRecord creates a suppression record for each entry in the file
func (importer BenePrefsImporter) createBenePrefsRecord(ctx context.Context, metadata *models.BenePrefsFilenameMetadata, b []byte) error {
	suppression, err := parseRecord(metadata, b)
	if err != nil {
		importer.Logger.Error(err)
		return err
	}

	if err = importer.Repo.CreateBenePrefsRecord(ctx, *suppression); err != nil {
		err = fmt.Errorf("failed to create bene-prefs record, err: %w", err)
		importer.Logger.Error(err)
		return err
	}

	return nil
}

func (importer BenePrefsImporter) loadBenePrefsFiles(ctx context.Context, path string) (suppressList *[]*models.BenePrefsFilenameMetadata, skipped int, err error) {
	var result []*models.BenePrefsFilenameMetadata

	bucket, prefix := bcdaaws.ParseS3Uri(path)
	s3Objects, err := bcdaaws.ListFiles(ctx, importer.FileClient, bucket, prefix)
	if err != nil {
		return &result, skipped, err
	}

	for _, obj := range s3Objects {
		metadata, err := parseMetadata(*obj.Key)
		metadata.FilePath = fmt.Sprintf("s3://%s/%s", bucket, *obj.Key)
		metadata.DeliveryDate = *obj.LastModified

		if err != nil {
			// Skip files with a bad name.  An unknown file in this dir isn't a blocker
			importer.Logger.Errorf("Issue parsing filename into metadata: %v", err)
			skipped = skipped + 1
			continue
		}

		result = append(result, &metadata)
	}

	return &result, skipped, err
}

func (importer BenePrefsImporter) cleanupBenePrefsFiles(ctx context.Context, suppresslist []*models.BenePrefsFilenameMetadata) error {
	errCount := 0

	for _, bpFile := range suppresslist {
		if !bpFile.Imported {
			// Don't do anything. The S3 bucket should have a retention policy that
			// automatically cleans up files after a specified period of time,
			importer.Logger.Warningf("File %s was not imported successfully. Skipping cleanup", bpFile)
			continue
		}

		importer.Logger.Infof("Cleaning up file %s\n", bpFile)
		err := bcdaaws.Delete(ctx, importer.FileClient, bpFile.FilePath)

		if err != nil {
			errCount++
			continue
		}

		importer.Logger.Infof("File %s successfully ingested and deleted from S3", bpFile)
	}

	if errCount > 0 {
		return fmt.Errorf("%d files could not be cleaned up", errCount)
	}

	return nil
}
