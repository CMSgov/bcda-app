package cclf

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	f "path/filepath"
	"time"

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
	Database      *sql.DB
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
	repository := postgres.NewRepository(importer.Database)
	exists, err := repository.GetCCLFFileExistsByName(ctx, csv.metadata.name)
	if err != nil {
		return fmt.Errorf("database query returned an error: %s", err)
	}
	if exists {
		return &ers.AttributionFileAlreadyExists{Filename: csv.metadata.name}
	}

	importer.Logger.Infof("Importing CSV file %s...", csv.metadata.name)

	tx, err := importer.Database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	rtx := postgres.NewRepositoryPgxTx(tx)
	var records int
	defer func() {
		if err != nil {
			if err1 := tx.Rollback(); err1 != nil {
				importer.Logger.Errorf("Failed to rollback transaction: %s, %s", err.Error(), err1.Error())
			}
			return
		}
	}()

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

	record.ID, err = rtx.CreateCCLFFile(ctx, record)
	if err != nil {
		err := fmt.Errorf("database error when calling CreateCCLFFile(): %s", err)
		return err
	}

	csv.metadata.fileID = record.ID

	rows, count, err := importer.prepareCSVData(csv.data, record.ID)
	if err != nil {
		return err
	}

	// Insert beneficiaries using bulk insert
	records, err = importer.insertBeneficiaries(ctx, tx, rows)
	if count != records {
		return fmt.Errorf("unexpected number of records imported (expected: %d, actual: %d)", count, records)
	}
	if err != nil {
		return errors.New("failed to write attribution beneficiaries to database")
	}

	err = rtx.UpdateCCLFFileImportStatus(ctx, csv.metadata.fileID, constants.ImportComplete)
	if err != nil {
		return fmt.Errorf("database error when calling UpdateCCLFFileImportStatus(): %s", csv.metadata.name)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit database transaction: %s", err)
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

func (importer CSVImporter) insertBeneficiaries(ctx context.Context, tx *sql.Tx, rows [][]interface{}) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}

	// Prepare bulk insert statement
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO cclf_beneficiaries (file_id, mbi) VALUES ($1, $2)")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var count int
	for _, row := range rows {
		_, err = stmt.ExecContext(ctx, row[0], row[1])
		if err != nil {
			return count, fmt.Errorf("failed to insert row: %w", err)
		}
		count++
	}

	return count, nil
}
