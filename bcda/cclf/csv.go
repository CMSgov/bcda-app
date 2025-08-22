package cclf

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	f "path/filepath"
	"time"

	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	ers "github.com/CMSgov/bcda-app/bcda/errors"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/optout"
)

// FileProcessors for attribution are created as interfaces so that they can be passed in place of the implementation; local development and other envs will require different processors.
// This interface has two implementations; one for ingesting and testing locally, and one for ingesting in s3.
type CSVFileProcessor interface {
	// Fetch the csv attribution file to be imported.
	LoadCSV(path string) (*bytes.Reader, func(), error)
	// Remove csv attribution file that was successfully imported.
	CleanUpCSV(file csvFile) (err error)
}

type csvFile struct {
	metadata csvFileMetadata
	data     *bytes.Reader
	imported bool
	filepath string
}

type csvFileMetadata struct {
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

type CSVImporter struct {
	Logger        logrus.FieldLogger
	FileProcessor CSVFileProcessor
	PgxPool       *pgxv5Pool.Pool
}

func (importer CSVImporter) ImportCSV(filepath string) error {

	file := csvFile{filepath: filepath}

	optOut, _ := optout.IsOptOut(filepath)
	if optOut {
		return &ers.IsOptOutFile{}
	}

	short := f.Base(filepath)

	metadata, err := GetCSVMetadata(short)
	if err != nil {
		return &ers.InvalidCSVMetadata{Msg: err.Error()}
	}
	file.metadata = metadata

	data, _, err := importer.FileProcessor.LoadCSV(filepath)
	if err != nil {
		if errors.Is(err, &ers.AttributionFileMismatchedEnv{}) {
			importer.Logger.WithFields(logrus.Fields{"file": filepath}).Info(err)
			return nil
		} else {
			return err
		}
	}

	file.data = data

	err = importer.ProcessCSV(file)
	if err != nil {
		return err
	}

	err = importer.FileProcessor.CleanUpCSV(file)
	if err != nil {
		return err
	}
	return nil
}

// ProcessCSV() will take provided metadata and write a new record to the cclf_files table and the contents of the file and write new record(s) to the cclf_beneficiaries table.
// If any step of writing to the database should fail, the whole transaction will fail. If the new records are written successfully, then the new record in the cclf_files
// table will have it's import status updated.
func (importer CSVImporter) ProcessCSV(csv csvFile) error {
	ctx := context.Background()
	if importer.PgxPool == nil {
		return errors.New("pgx pool is required for import operations")
	}

	pgxRepo := postgres.NewPgxRepositoryWithPool(importer.PgxPool)
	exists, err := pgxRepo.GetCCLFFileExistsByName(ctx, csv.metadata.name)
	if err != nil {
		return fmt.Errorf("database query returned an error: %s", err)
	}
	if exists {
		return &ers.AttributionFileAlreadyExists{Filename: csv.metadata.name}
	}

	importer.Logger.Infof("Importing CSV file %s...", csv.metadata.name)

	pgxTx, err := importer.PgxPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start pgx transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := pgxTx.Rollback(ctx); rollbackErr != nil {
				importer.Logger.Errorf("Failed to rollback pgx transaction: %s, %s", err.Error(), rollbackErr.Error())
			}
			return
		}
	}()

	var records int

	// CCLF model corresponds with a database record
	record := models.CCLFFile{
		CCLFNum:         csv.metadata.cclfNum,
		Name:            csv.metadata.name,
		ACOCMSID:        csv.metadata.acoID,
		Timestamp:       csv.metadata.timestamp,
		PerformanceYear: csv.metadata.perfYear,
		ImportStatus:    constants.ImportInprog,
		Type:            csv.metadata.fileType,
	}

	record.ID, err = pgxRepo.CreateCCLFFileTx(ctx, pgxTx, record)
	if err != nil {
		err := fmt.Errorf("database error when calling CreateCCLFFile(): %s", err)
		return err
	}

	csv.metadata.fileID = record.ID

	rows, count, err := importer.prepareCSVData(csv.data, record.ID)
	if err != nil {
		return err
	}

	importedCount, err := pgxTx.CopyFrom(ctx, pgxv5.Identifier{"cclf_beneficiaries"}, []string{"file_id", "mbi"}, pgxv5.CopyFromRows(rows))
	records = int(importedCount)
	if err != nil {
		return fmt.Errorf("failed to write attribution beneficiaries to database: %w", err)
	}
	if count != records {
		return fmt.Errorf("unexpected number of records imported (expected: %d, actual: %d)", count, records)
	}

	err = pgxRepo.UpdateCCLFFileImportStatusTx(ctx, pgxTx, csv.metadata.fileID, constants.ImportComplete)
	if err != nil {
		return fmt.Errorf("database error when calling UpdateCCLFFileImportStatus(): %s", csv.metadata.name)
	}

	if err = pgxTx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit pgx transaction: %s", err)
	}

	successMsg := fmt.Sprintf("successfully imported %d records from csv file %s.", records, csv.metadata.name)
	importer.Logger.WithFields(logrus.Fields{"imported_count": records}).Info(successMsg)
	return nil
}

func (importer CSVImporter) prepareCSVData(csvfile *bytes.Reader, id uint) ([][]interface{}, int, error) {
	var rows [][]interface{}
	r := csv.NewReader(csvfile)

	_, err := r.Read()
	if err == io.EOF {
		return nil, 0, errors.New("empty attribution file")
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read csv attribution header: %s", err)
	}

	count := 0

	for {
		var record []string
		record, err = r.Read()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read csv attribution file: %s", err)
		}
		row := make([]interface{}, 2)
		row[0] = id
		row[1] = record[0]

		rows = append(rows, row)
		count++
	}
	return rows, count, err

}
