package suppression

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/CMSgov/bcda-app/optout"
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

type OptOutImporter struct {
	FileHandler          optout.OptOutFileHandler
	Saver                optout.Saver
	Logger               logrus.FieldLogger
	ImportStatusInterval int
}

func (importer OptOutImporter) ImportSuppressionDirectory(path string) (success, failure, skipped int, err error) {
	suppresslist, skipped, err := importer.FileHandler.LoadOptOutFiles(path)
	if err != nil {
		return 0, 0, 0, err
	}

	if len(suppresslist) == 0 {
		importer.Logger.Info("Failed to find any suppression files in directory")
		return 0, 0, skipped, nil
	}

	for _, metadata := range suppresslist {
		err = importer.validate(metadata)
		if err != nil {
			fmt.Printf("Failed to validate suppression file: %s.\n", metadata)
			importer.Logger.Errorf("Failed to validate suppression file: %s", metadata)
			failure++
		} else {
			if err = importer.ImportSuppressionData(metadata); err != nil {
				fmt.Printf("Failed to import suppression file: %s.\n", metadata)
				importer.Logger.Errorf("Failed to import suppression file: %s ", metadata)
				failure++
			} else {
				metadata.Imported = true
				success++
			}
		}
	}
	err = importer.FileHandler.CleanupOptOutFiles(suppresslist)
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

func (importer OptOutImporter) validate(metadata *optout.OptOutFilenameMetadata) error {
	fmt.Printf("Validating suppression file %s...\n", metadata)
	importer.Logger.Infof("Validating suppression file %s...", metadata)

	count := 0
	sc, close, err := importer.FileHandler.OpenFile(metadata)
	if err != nil {
		fmt.Printf("Could not read file %s.\n", metadata)
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
				fmt.Printf("Invalid file header for file: %s.\n", metadata.FilePath)
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
				fmt.Printf("Failed to parse record count from file: %s.\n", metadata.FilePath)
				err = fmt.Errorf("failed to parse record count from file: %s", metadata.FilePath)
				importer.Logger.Error(err)
				return err
			}
			// subtract the single count from the header
			count--
			if count != expectedCount {
				fmt.Printf("Incorrect number of records found from file: '%s'. Expected record count: %d, Actual record count: %d.\n", metadata.FilePath, expectedCount, count)
				err = fmt.Errorf("incorrect number of records found from file: '%s'. Expected record count: %d, Actual record count: %d", metadata.FilePath, expectedCount, count)
				importer.Logger.Error(err)
				return err
			}
		}
	}

	fmt.Printf("Successfully validated suppression file %s.\n", metadata)
	importer.Logger.Infof("Successfully validated suppression file %s.", metadata)
	return nil
}

func (importer OptOutImporter) ImportSuppressionData(metadata *optout.OptOutFilenameMetadata) error {
	err := importer.importSuppressionMetadata(metadata, func(fileID uint, b []byte) error {
		suppression, err := optout.ParseRecord(metadata, b)

		if err != nil {
			importer.Logger.Error(err)
			return err
		}

		if err = importer.Saver.SaveOptOutRecord(*suppression); err != nil {
			fmt.Println("Could not create suppression record.")
			err = errors.Wrap(err, "could not create suppression record")
			importer.Logger.Error(err)
			return err
		}
		return nil
	})

	if err != nil {
		importer.updateImportStatus(metadata, optout.ImportFail)
		return err
	}
	importer.updateImportStatus(metadata, optout.ImportComplete)
	return nil
}

func (importer OptOutImporter) importSuppressionMetadata(metadata *optout.OptOutFilenameMetadata, importFunc func(uint, []byte) error) error {
	fmt.Printf("Importing suppression file %s...\n", metadata)
	importer.Logger.Infof("Importing suppression file %s...", metadata)

	var (
		headTrailStart, headTrailEnd = 0, 15
		err                          error
	)
	suppressionMetaFile := optout.OptOutFile{
		Name:         metadata.Name,
		Timestamp:    metadata.Timestamp,
		ImportStatus: optout.ImportInprog,
	}

	if suppressionMetaFile.ID, err = importer.Saver.SaveFile(suppressionMetaFile); err != nil {
		fmt.Printf("Could not create suppression file record for file: %s. \n", metadata)
		err = errors.Wrapf(err, "could not create suppression file record for file: %s.", metadata)
		importer.Logger.Error(err)
		return err
	}

	metadata.FileID = suppressionMetaFile.ID

	importedCount := 0

	sc, close, err := importer.FileHandler.OpenFile(metadata)
	if err != nil {
		fmt.Printf("Could not read file %s.\n", metadata)
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
			err = importFunc(suppressionMetaFile.ID, b)
			if err != nil {
				importer.Logger.Error(err)
				return err
			}
			importedCount++
			if importedCount%importer.ImportStatusInterval == 0 {
				fmt.Printf("Suppression records imported: %d\n", importedCount)
			}
		}
	}

	successMsg := fmt.Sprintf("Successfully imported %d records from suppression file %s.", importedCount, metadata)
	fmt.Println(successMsg)
	importer.Logger.Infof(successMsg)
	return nil
}

func (importer OptOutImporter) updateImportStatus(metadata *optout.OptOutFilenameMetadata, status string) {
	if err := importer.Saver.UpdateImportStatus(*metadata, status); err != nil {
		fmt.Printf("Could not update suppression file record for file_id: %s. \n", metadata.String())
		err = errors.Wrapf(err, "could not update suppression file record for file_id: %s.", metadata.String())
		importer.Logger.Error(err)
	}
}
