package cclf

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/stdlib"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
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
	fileType     models.CCLFFileType
}

type cclfFileValidator struct {
	totalRecordCount int
	maxRecordLength  int
}

// Manages the interaction of CCLF files from a given source
type CclfFileProcessor interface {
	// Load a list of valid CCLF files to be imported
	LoadCclfFiles(path string) (cclfList map[string]map[metadataKey][]*cclfFileMetadata, skipped int, failed int, err error)
	// Clean up CCLF files after failed or successful import runs
	CleanUpCCLF(ctx context.Context, cclfMap map[string]map[metadataKey][]*cclfFileMetadata) error
	// Open a zip archive
	OpenZipArchive(name string) (*zip.Reader, func(), error)
}

// Manages the import process for CCLF files from a given source
type CclfImporter struct {
	Logger        logrus.FieldLogger
	FileProcessor CclfFileProcessor
}

func (importer CclfImporter) importCCLF0(ctx context.Context, fileMetadata *cclfFileMetadata) (map[string]cclfFileValidator, error) {
	if fileMetadata == nil {
		err := errors.New("file CCLF0 not found")
		importer.Logger.Error(err)
		return nil, err
	}

	importer.Logger.Infof("Importing CCLF0 file %s...", fileMetadata)

	r, closeReader, err := importer.FileProcessor.OpenZipArchive(fileMetadata.filePath)
	if err != nil {
		err := errors.Wrapf(err, "could not read CCLF0 archive %s", fileMetadata)
		importer.Logger.Error(err)
		return nil, err
	}
	defer closeReader()

	const (
		fileNumStart, fileNumEnd           = 0, 7
		totalRecordStart, totalRecordEnd   = 52, 63
		recordLengthStart, recordLengthEnd = 64, 69
	)

	close := metrics.NewChild(ctx, "importCCLF0")
	defer close()

	var validator map[string]cclfFileValidator
	var rawFile *zip.File

	for _, f := range r.File {
		// iterate in this zipped folder until we find our cclf0 file
		if f.Name == fileMetadata.name {
			rawFile = f
			importer.Logger.Infof("Reading file %s from archive %s", fileMetadata.name, fileMetadata.filePath)
		}
	}

	if rawFile == nil {
		err = errors.Wrapf(err, constants.FileNotFound, fileMetadata.name, fileMetadata.filePath)
		importer.Logger.Error(err)
		return nil, err
	}

	rc, err := rawFile.Open()
	if err != nil {
		err = errors.Wrapf(err, "could not read file %s in CCLF0 archive %s", fileMetadata.name, fileMetadata.filePath)
		importer.Logger.Error(err)
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
					err := fmt.Errorf("duplicate %v file type found from CCLF0 file", filetype)
					importer.Logger.Error(err)
					return nil, err
				}

				count, err := strconv.Atoi(string(bytes.TrimSpace(b[totalRecordStart:totalRecordEnd])))
				if err != nil {
					err = errors.Wrapf(err, "failed to parse %s record count from CCLF0 file", filetype)
					importer.Logger.Error(err)
					return nil, err
				}
				length, err := strconv.Atoi(string(bytes.TrimSpace(b[recordLengthStart:recordLengthEnd])))
				if err != nil {
					err = errors.Wrapf(err, "failed to parse %s record length from CCLF0 file", filetype)
					importer.Logger.Error(err)
					return nil, err
				}
				validator[filetype] = cclfFileValidator{totalRecordCount: count, maxRecordLength: length}
			}
		}
	}

	if _, ok := validator["CCLF8"]; !ok {
		err := fmt.Errorf("failed to parse CCLF8 from CCLF0 file %s", fileMetadata)
		importer.Logger.Error(err)
		return nil, err
	}
	importer.Logger.Infof("Successfully imported CCLF0 file %s.", fileMetadata)

	return validator, nil
}

func (importer CclfImporter) importCCLF8(ctx context.Context, fileMetadata *cclfFileMetadata) (err error) {
	db := database.Connection
	repository := postgres.NewRepository(db)
	exists, err := repository.GetCCLFFileExistsByName(ctx, fileMetadata.name)
	if err != nil {
		err = errors.Wrapf(err, "failed to check existence of CCLF%d file", fileMetadata.cclfNum)
		importer.Logger.Error(err)
		return err
	}

	if exists {
		importer.Logger.Infof("CCL%d file %s already exists in database, skipping import...", fileMetadata.cclfNum, fileMetadata)
		return nil
	}

	importer.Logger.Infof("Importing CCLF%d file %s...", fileMetadata.cclfNum, fileMetadata)

	conn, err := stdlib.AcquireConn(db)
	defer utils.CloseAndLog(logrus.WarnLevel, func() error { return stdlib.ReleaseConn(db, conn) })

	tx, err := conn.BeginEx(ctx, nil)
	if err != nil {
		err = fmt.Errorf("failed to start transaction: %w", err)
		importer.Logger.Error(err)

		return err
	}

	rtx := postgres.NewRepositoryPgxTx(tx)

	defer func() {
		if err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				importer.Logger.Warnf("Failed to rollback transaction %s", err.Error())
			}
			return
		}
	}()

	r, closeReader, err := importer.FileProcessor.OpenZipArchive(fileMetadata.filePath)
	if err != nil {
		err = errors.Wrapf(err, "could not read CCLF%d archive %s", fileMetadata.cclfNum, fileMetadata.filePath)
		importer.Logger.Error(err)
		return err
	}
	defer closeReader()

	if len(r.File) < 1 {
		err = fmt.Errorf("no files found in CCLF%d archive %s", fileMetadata.cclfNum, fileMetadata.filePath)
		importer.Logger.Error(err)
		return err
	}

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
		importer.Logger.Error(err)
		return err
	}

	fileMetadata.fileID = cclfFile.ID

	var rawFile *zip.File

	for _, f := range r.File {
		if f.Name == fileMetadata.name {
			rawFile = f
			importer.Logger.Infof("Reading file %s from archive %s", fileMetadata.name, fileMetadata.filePath)
		}
	}

	if rawFile == nil {
		err = fmt.Errorf(constants.FileNotFound, fileMetadata.name, fileMetadata.filePath)
		importer.Logger.Error(err)
		return err
	}

	rc, err := rawFile.Open()
	if err != nil {
		err = errors.Wrapf(err, "could not read file %s for CCLF%d in archive %s", cclfFile.Name, fileMetadata.cclfNum, fileMetadata.filePath)
		importer.Logger.Error(err)
		return err
	}
	defer rc.Close()
	sc := bufio.NewScanner(rc)

	importedCount, err := CopyFrom(ctx, tx, sc, cclfFile.ID, utils.GetEnvInt("CCLF_IMPORT_STATUS_RECORDS_INTERVAL", 10000), importer.Logger)
	if err != nil {
		return errors.Wrap(err, "failed to copy data to beneficiaries table")
	}
	err = rtx.UpdateCCLFFileImportStatus(ctx, fileMetadata.fileID, constants.ImportComplete)
	if err != nil {
		err = errors.Wrapf(err, "could not update cclf file record for file: %s.", fileMetadata)
		importer.Logger.Error(err)
	}

	if err = tx.Commit(); err != nil {
		importer.Logger.Error(err.Error())
		failMsg := fmt.Sprintf("failed to commit transaction for CCLF%d import file %s", fileMetadata.cclfNum, fileMetadata)
		return errors.Wrap(err, failMsg)
	}

	successMsg := fmt.Sprintf("Successfully imported %d records from CCLF%d file %s.", importedCount, fileMetadata.cclfNum, fileMetadata)
	importer.Logger.WithFields(logrus.Fields{"imported_count": importedCount}).Infof(successMsg)

	return nil
}

func (importer CclfImporter) ImportCCLFDirectory(filePath string) (success, failure, skipped int, err error) {
	t := metrics.GetTimer()
	defer t.Close()
	ctx := metrics.NewContext(context.Background(), t)

	// We are not going to create any children from this parent so we can
	// safely ignored the returned context.
	_, c := metrics.NewParent(ctx, "ImportCCLFDirectory#sortCCLFArchives")
	cclfMap, skipped, failure, err := importer.FileProcessor.LoadCclfFiles(filePath)
	c()

	if err != nil {
		return 0, 0, 0, err
	}

	if len(cclfMap) == 0 {
		importer.Logger.Info("Did not find any CCLF files in directory -- returning safely.")
		return 0, 0, skipped, nil
	}

	for acoID := range cclfMap {
		func() {
			ctx, c := metrics.NewParent(ctx, "ImportCCLFDirectory#processACOs")
			defer c()
			for _, cclfFiles := range cclfMap[acoID] {
				var cclf0, cclf8 *cclfFileMetadata
				for _, cclf := range cclfFiles {
					if cclf.cclfNum == 0 {
						cclf0 = cclf
					} else if cclf.cclfNum == 8 {
						cclf8 = cclf
					}
				}
				cclfvalidator, err := importer.importCCLF0(ctx, cclf0)
				if err != nil {
					importer.Logger.Errorf("Failed to import CCLF0 file: %s, Skipping CCLF8 file: %s ", cclf0, cclf8)
					failure++
					skipped += 2
					continue
				} else {
					success++
				}
				err = importer.validate(ctx, cclf8, cclfvalidator)
				if err != nil {
					importer.Logger.Errorf("Failed to validate CCLF8 file: %s", cclf8)
					failure++
				} else {
					if err = importer.importCCLF8(ctx, cclf8); err != nil {
						importer.Logger.Errorf("Failed to import CCLF8 file: %s %s", cclf8, err)
						failure++
					} else {
						cclf8.imported = true
						success++
					}
				}
				cclf0.imported = cclf8 != nil && cclf8.imported
			}
		}()
	}

	if err = func() error {
		ctx, c := metrics.NewParent(ctx, "ImportCCLFDirectory#cleanupCCLF")
		defer c()
		return importer.FileProcessor.CleanUpCCLF(ctx, cclfMap)
	}(); err != nil {
		importer.Logger.Error(err)
	}

	if failure > 0 {
		err = errors.New(fmt.Sprintf("Failed to import %d files", failure))
		importer.Logger.Error(err)
	} else {
		err = nil
	}

	return success, failure, skipped, err
}

func (importer CclfImporter) validate(ctx context.Context, fileMetadata *cclfFileMetadata, cclfFileValidator map[string]cclfFileValidator) error {
	if fileMetadata == nil {
		err := errors.New("file not found")
		importer.Logger.Error(err)
		return err
	}

	importer.Logger.Infof("Validating CCLF%d file %s...", fileMetadata.cclfNum, fileMetadata)

	var key string
	if fileMetadata.cclfNum == 8 {
		key = "CCLF8"
	} else {
		err := fmt.Errorf("unknown file type when validating file: %s", fileMetadata)
		importer.Logger.Error(err)
		return err
	}

	r, closeReader, err := importer.FileProcessor.OpenZipArchive(fileMetadata.filePath)
	if err != nil {
		err := errors.Wrapf(err, "could not read archive %s", fileMetadata.filePath)
		importer.Logger.Error(err)
		return err
	}
	defer closeReader()

	close := metrics.NewChild(ctx, "validate")
	defer close()

	count := 0
	validator := cclfFileValidator[key]
	var rawFile *zip.File

	for _, f := range r.File {
		if f.Name == fileMetadata.name {
			rawFile = f
			importer.Logger.Infof("Reading file %s from archive %s", fileMetadata.name, fileMetadata.filePath)
		}
	}

	if rawFile == nil {
		err = errors.Wrapf(err, constants.FileNotFound, fileMetadata.name, fileMetadata.filePath)
		importer.Logger.Error(err)
		return err
	}

	rc, err := rawFile.Open()
	if err != nil {
		err = errors.Wrapf(err, "could not read file %s in archive %s", fileMetadata.name, fileMetadata.filePath)
		importer.Logger.Error(err)
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
				err := fmt.Errorf("maximum record count reached for file %s (expected: %d, actual: %d)", key, validator.totalRecordCount, count)
				importer.Logger.Error(err)
				return err
			}
		} else {
			err := fmt.Errorf("incorrect record length for file %s (expected: %d, actual: %d)", key, validator.maxRecordLength, bytelength)
			importer.Logger.Error(err)
			return err
		}
	}
	importer.Logger.Infof("Successfully validated CCLF%d file %s.", fileMetadata.cclfNum, fileMetadata)
	return nil
}

func (m cclfFileMetadata) String() string {
	if m.name != "" {
		return m.name
	}
	return m.filePath
}
