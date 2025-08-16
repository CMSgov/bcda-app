package cclf

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"
)

type cclfZipMetadata struct {
	acoID         string
	cclf0Metadata cclfFileMetadata
	cclf8Metadata cclfFileMetadata
	cclf0File     zip.File
	cclf8File     zip.File
	zipReader     *zip.Reader
	zipCloser     func()
	filePath      string
	imported      bool
}

type cclfFileMetadata struct {
	name         string
	env          string
	acoID        string
	cclfNum      int
	perfYear     int
	timestamp    time.Time
	deliveryDate time.Time
	fileID       uint
	fileType     models.CCLFFileType
}

type cclfFileValidator struct {
	totalRecordCount int
	maxRecordLength  int
}

// Manages the interaction of CCLF files from a given source
type CclfFileProcessor interface {
	// Load a list of valid CCLF files to be imported
	LoadCclfFiles(path string) (cclfList map[string][]*cclfZipMetadata, skipped int, failed int, err error)
	// Clean up CCLF files after failed or successful import runs
	CleanUpCCLF(ctx context.Context, cclfMap map[string][]*cclfZipMetadata) (deletedCount int, err error)
	// Open a zip archive
	OpenZipArchive(name string) (*zip.Reader, func(), error)
}

// Manages the import process for CCLF files from a given source
type CclfImporter struct {
	logger        logrus.FieldLogger
	fileProcessor CclfFileProcessor
	db            *sql.DB
}

func NewCclfImporter(logger logrus.FieldLogger, fileProcessor CclfFileProcessor, db *sql.DB) CclfImporter {
	return CclfImporter{logger: logger, fileProcessor: fileProcessor, db: db}
}

func (importer CclfImporter) importCCLF0(ctx context.Context, zipMetadata *cclfZipMetadata) (*cclfFileValidator, error) {
	fileMetadata := zipMetadata.cclf0Metadata
	importer.logger.Infof("Importing CCLF0 file %s...", fileMetadata)

	const (
		fileNumStart, fileNumEnd           = 0, 7
		totalRecordStart, totalRecordEnd   = 52, 63
		recordLengthStart, recordLengthEnd = 64, 69
	)

	close := metrics.NewChild(ctx, "importCCLF0")
	defer close()

	rc, err := zipMetadata.cclf0File.Open()
	if err != nil {
		err = errors.Wrapf(err, "could not read file %s in CCLF0 archive %s", fileMetadata.name, zipMetadata.filePath)
		importer.logger.Error(err)
		return nil, err
	}
	defer rc.Close()
	sc := bufio.NewScanner(rc)

	var validator *cclfFileValidator

	for sc.Scan() {
		b := sc.Bytes()
		if len(bytes.TrimSpace(b)) > 0 {
			filetype := string(bytes.TrimSpace(b[fileNumStart:fileNumEnd]))

			if filetype == "CCLF8" {
				if validator != nil {
					err := fmt.Errorf("duplicate %v file type found from CCLF0 file", filetype)
					importer.logger.Error(err)
					return nil, err
				}

				count, err := strconv.Atoi(string(bytes.TrimSpace(b[totalRecordStart:totalRecordEnd])))
				if err != nil {
					err = errors.Wrapf(err, "failed to parse %s record count from CCLF0 file", filetype)
					importer.logger.Error(err)
					return nil, err
				}
				length, err := strconv.Atoi(string(bytes.TrimSpace(b[recordLengthStart:recordLengthEnd])))
				if err != nil {
					err = errors.Wrapf(err, "failed to parse %s record length from CCLF0 file", filetype)
					importer.logger.Error(err)
					return nil, err
				}

				validator = &cclfFileValidator{totalRecordCount: count, maxRecordLength: length}
			}
		}
	}

	if validator != nil {
		importer.logger.Infof("Successfully imported CCLF0 file %s.", fileMetadata)
		return validator, nil
	}

	err = fmt.Errorf("failed to parse CCLF8 from CCLF0 file %s", fileMetadata.name)
	importer.logger.Error(err)
	return nil, err
}

func (importer CclfImporter) importCCLF8(ctx context.Context, zipMetadata *cclfZipMetadata, validator cclfFileValidator) (err error) {
	fileMetadata := zipMetadata.cclf8Metadata

	repository := postgres.NewRepository(importer.db)
	exists, err := repository.GetCCLFFileExistsByName(ctx, fileMetadata.name)
	if err != nil {
		err = errors.Wrapf(err, "failed to check existence of CCLF%d file", fileMetadata.cclfNum)
		importer.logger.Error(err)
		return err
	}

	if exists {
		importer.logger.Infof("CCL%d file %s already exists in database, skipping import...", fileMetadata.cclfNum, fileMetadata)
		return nil
	}

	importer.logger.Infof("Importing CCLF%d file %s...", fileMetadata.cclfNum, fileMetadata)

	tx, err := importer.db.BeginTx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("failed to start transaction: %w", err)
		importer.logger.Error(err)
		return err
	}

	rtx := postgres.NewRepositoryPgxTx(tx)

	defer func() {
		if err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				importer.logger.Warnf("Failed to rollback transaction %s", err.Error())
			}
			return
		}
	}()

	close := metrics.NewChild(ctx, fmt.Sprintf("importCCLF%d", fileMetadata.cclfNum))
	defer close()

	cclfFile := models.CCLFFile{
		CCLFNum:         fileMetadata.cclfNum,
		Name:            fileMetadata.name,
		ACOCMSID:        fileMetadata.acoID,
		Timestamp:       fileMetadata.timestamp,
		PerformanceYear: fileMetadata.perfYear,
		ImportStatus:    constants.ImportInprog,
		Type:            fileMetadata.fileType,
	}
	cclfFile.ID, err = rtx.CreateCCLFFile(ctx, cclfFile)
	if err != nil {
		err = errors.Wrapf(err, "could not create CCLF%d file record", fileMetadata.cclfNum)
		importer.logger.Error(err)
		return err
	}

	fileMetadata.fileID = cclfFile.ID

	rc, err := zipMetadata.cclf8File.Open()
	if err != nil {
		err = errors.Wrapf(err, "could not read file %s for CCLF%d in archive %s", cclfFile.Name, fileMetadata.cclfNum, zipMetadata.filePath)
		importer.logger.Error(err)
		return err
	}
	defer rc.Close()
	sc := bufio.NewScanner(rc)

	importedCount, recordCount, err := CopyFrom(ctx, tx, sc, cclfFile.ID, utils.GetEnvInt("CCLF_IMPORT_STATUS_RECORDS_INTERVAL", 10000), importer.logger, validator.maxRecordLength)
	if err != nil {
		return errors.Wrap(err, "failed to copy data to beneficiaries table")
	}

	if recordCount > validator.totalRecordCount {
		err := fmt.Errorf("unexpected number of records imported for file %s (expected: %d, actual: %d)", fileMetadata.name, validator.totalRecordCount, recordCount)
		importer.logger.Error(err)
		return err
	}

	err = rtx.UpdateCCLFFileImportStatus(ctx, fileMetadata.fileID, constants.ImportComplete)
	if err != nil {
		err = errors.Wrapf(err, "could not update cclf file record for file: %s.", fileMetadata.name)
		importer.logger.Error(err)
	}

	if err = tx.Commit(); err != nil {
		importer.logger.Error(err.Error())
		failMsg := fmt.Sprintf("failed to commit transaction for CCLF%d import file %s", fileMetadata.cclfNum, fileMetadata.name)
		return errors.Wrap(err, failMsg)
	}

	successMsg := fmt.Sprintf("Successfully imported %d records from CCLF%d file %s.", importedCount, fileMetadata.cclfNum, fileMetadata.name)
	importer.logger.WithFields(logrus.Fields{"imported_count": importedCount}).Info(successMsg)

	return nil
}

func (importer CclfImporter) ImportCCLFDirectory(filePath string) (success, failure, skipped int, err error) {
	t := metrics.GetTimer()
	defer t.Close()
	ctx := metrics.NewContext(context.Background(), t)

	// We are not going to create any children from this parent so we can
	// safely ignored the returned context.
	_, c := metrics.NewParent(ctx, "ImportCCLFDirectory#sortCCLFArchives")
	cclfMap, skipped, failure, err := importer.fileProcessor.LoadCclfFiles(filePath)
	c()

	if err != nil {
		return 0, 0, 0, err
	}

	if len(cclfMap) == 0 {
		importer.logger.Info("Did not find any CCLF files in directory -- returning safely.")
		return 0, failure, skipped, err
	}

	for acoID := range cclfMap {
		for _, zipMetadata := range cclfMap[acoID] {
			func() {
				ctx, c := metrics.NewParent(ctx, "ImportCCLFDirectory#processACOs")
				defer c()
				defer zipMetadata.zipCloser()

				cclfvalidator, err := importer.importCCLF0(ctx, zipMetadata)
				if err != nil {
					importer.logger.Errorf("Failed to import CCLF0 file: %s, Skipping CCLF8 file: %s ", zipMetadata.cclf0Metadata, zipMetadata.cclf8Metadata)
					failure++
					skipped += 2
				} else {
					success++
				}

				if err = importer.importCCLF8(ctx, zipMetadata, *cclfvalidator); err != nil {
					importer.logger.Errorf("Failed to import CCLF8 file: %s %s", zipMetadata.cclf8Metadata, err)
					failure++
				} else {
					zipMetadata.imported = true
					success++
				}
			}()
		}
	}

	if err = func() error {
		ctx, c := metrics.NewParent(ctx, "ImportCCLFDirectory#cleanupCCLF")
		defer c()
		_, err := importer.fileProcessor.CleanUpCCLF(ctx, cclfMap)
		return err
	}(); err != nil {
		importer.logger.Error(err)
	}

	if failure > 0 {
		err = errors.New(fmt.Sprintf("Failed to import %d files", failure))
		importer.logger.Error(err)
	} else {
		err = nil
	}

	return success, failure, skipped, err
}

func (m cclfFileMetadata) String() string {
	return m.name
}
