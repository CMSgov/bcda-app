package beneprefs

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/pkg/errors"
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

// An BenePrefsImporter imports opt out files based on the provided file handler and saver.
type BenePrefsImporter struct {
	FileHandler BenePrefsFileHandler
	// Saver                Saver
	Repo                 models.Repository
	Logger               logrus.FieldLogger
	ImportStatusInterval int
}

func (importer BenePrefsImporter) ImportSuppressionDirectory(ctx context.Context, path string) (success, failure, skipped int, err error) {
	suppresslist, skipped, err := importer.FileHandler.LoadBenePrefsFiles(ctx, path)
	if err != nil {
		return 0, 0, 0, err
	}

	if len(*suppresslist) == 0 {
		importer.Logger.Info("Failed to find any suppression files in directory")
		return 0, 0, skipped, nil
	}

	for _, metadata := range *suppresslist {
		err = importer.validate(ctx, metadata)
		if err != nil {
			importer.Logger.Errorf("Failed to validate suppression file: %s", metadata)
			failure++
		} else {
			if err = importer.ImportSuppressionData(ctx, metadata); err != nil {
				importer.Logger.Errorf("Failed to import suppression file: %s ", metadata)
				failure++
			} else {
				metadata.Imported = true
				success++
			}
		}
	}
	err = importer.FileHandler.CleanupBenePrefsFiles(ctx, *suppresslist)
	if err != nil {
		importer.Logger.Error(err)
	}

	if failure > 0 {
		err = errors.New("one or more suppression files failed to import correctly")
		importer.Logger.Error(err)
	} else {
		err = nil
	}
	return success, failure, skipped, err
}

func (importer BenePrefsImporter) validate(ctx context.Context, metadata *models.BenePrefsFilenameMetadata) error {
	importer.Logger.Infof("Validating suppression file %s...", metadata)

	count := 0
	sc, close, err := importer.FileHandler.OpenFile(ctx, metadata)
	if err != nil {
		err = errors.Wrapf(err, "could not read file %s", metadata)
		importer.Logger.Error(err)
		return err
	}

	defer close()

	for sc.Scan() {
		b := sc.Bytes()
		metaInfo := string(bytes.TrimSpace(b[headTrailStart:headTrailEnd]))
		if count == 0 {
			if metaInfo != headerCode {
				// invalid file header found
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

	importer.Logger.Infof("Successfully validated suppression file %s.", metadata)
	return nil
}

func (importer BenePrefsImporter) ImportSuppressionData(ctx context.Context, metadata *models.BenePrefsFilenameMetadata) error {
	optOutCount := 0
	optInCount := 0
	err := importer.importSuppressionMetadata(ctx, metadata, func(fileID uint, b []byte) error {
		suppression, err := ParseRecord(metadata, b)

		if err != nil {
			importer.Logger.Error(err)
			return err
		}

		if err = importer.Repo.CreateBenePrefsRecord(ctx, *suppression); err != nil {
			err = errors.Wrap(err, "could not create suppression record")
			importer.Logger.Error(err)
			return err
		}
		switch suppression.PrefIndicator {
		case "Y":
			optInCount++
		case "N":
			optOutCount++
		}
		return nil
	})

	if err != nil {
		err2 := importer.Repo.UpdateBenePrefsImportStatus(ctx, metadata.FileID, constants.ImportFail)
		if err2 != nil {
			errMsg := errors.Wrapf(err2, "could not update suppression file import status for file: %s", metadata)
			importer.Logger.Error(errMsg)
		}
		return err
	}

	importer.Logger.WithFields(logrus.Fields{"created_opt_outs_count": optOutCount, "created_opt_ins_count": optInCount}).Infof("Successfully imported file: %s", metadata.Name)
	err = importer.Repo.UpdateBenePrefsImportStatus(ctx, metadata.FileID, constants.ImportComplete)
	if err != nil {
		err = errors.Wrapf(err, "could not update suppression file import status for file: %s", metadata)
		importer.Logger.Error(err)
		return err
	}

	return nil
}

func (importer BenePrefsImporter) importSuppressionMetadata(ctx context.Context, metadata *models.BenePrefsFilenameMetadata, importFunc func(uint, []byte) error) error {
	importer.Logger.Infof("Importing suppression file %s...", metadata)

	var (
		headTrailStart, headTrailEnd = 0, 15
		err                          error
	)
	bpFile := models.BenePrefsFile{
		Name:         metadata.Name,
		Timestamp:    metadata.Timestamp,
		ImportStatus: constants.ImportInprog,
	}

	if bpFile.ID, err = importer.Repo.CreateBenePrefsFile(ctx, bpFile); err != nil {
		err = errors.Wrapf(err, "could not create suppression file record for file: %s.", metadata)
		importer.Logger.Error(err)
		return err
	}

	metadata.FileID = bpFile.ID

	importedCount := 0

	sc, close, err := importer.FileHandler.OpenFile(ctx, metadata)
	if err != nil {
		err = errors.Wrapf(err, "could not read file %s", metadata)
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
			err = importFunc(bpFile.ID, b)
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

	importer.Logger.Infof("Successfully imported %d records from suppression file %s.", importedCount, metadata)
	return nil
}
