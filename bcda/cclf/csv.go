package cclf

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/log"
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
	data, _, err := importer.FileProcessor.LoadCSV(filepath)
	if err != nil {
		importer.Logger.Error(err)
	}
	file.data = data

	metadata, err := GetCSVMetadata(filepath)
	if err != nil {
		err = errors.Wrapf(err, "failed to get metadata from CSV S3 import parth")
		importer.Logger.Error(err)
		return err
	}
	file.metadata = metadata

	err = importer.ProcessCSV(file)
	if err != nil {
		importer.Logger.Error()
	}

	err = importer.FileProcessor.CleanUpCSV(file)
	if err != nil {
		log.API.Error("error!")
	}
	return nil
}

// ImportCSV() will take provided metadata and write a new record to the cclf_files table and the contents of the file and write new record(s) to the cclf_beneficiaries table.
// If any step of writing to the database should fail, the whole transaction will fail. If the new records are written successfully, then the new record in the cclf_files
// table will have it's import status updated.
func (importer CSVImporter) ProcessCSV(csv csvFile) error {
	ctx := context.Background()
	repository := postgres.NewRepository(importer.Database)
	exists, err := repository.GetCCLFFileExistsByName(ctx, csv.metadata.name)
	if err != nil {
		err = errors.Wrapf(err, "failed to check existence of CSV file")
		importer.Logger.Error(err)
		return err
	}
	if exists {
		importer.Logger.Infof("CSV File %s already exists in database, skipping import...", csv.metadata.name)
		return errors.New("Attribution file already exists") // make this a type
	}

	importer.Logger.Infof("Importing CSV file %s...", csv.metadata.name)
	conn, err := stdlib.AcquireConn(importer.Database)
	if err != nil {
		return err
	}

	defer utils.CloseAndLog(logrus.WarnLevel, func() error { return stdlib.ReleaseConn(importer.Database, conn) })

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

	// technically not a cclf file, but this model corresponds with a database record
	record := models.CCLFFile{
		Name:            csv.metadata.name,
		ACOCMSID:        csv.metadata.acoID,
		Timestamp:       csv.metadata.timestamp,
		PerformanceYear: csv.metadata.perfYear,
		ImportStatus:    constants.ImportInprog,
	}

	record.ID, err = rtx.CreateCCLFFile(ctx, record)
	if err != nil {
		err = errors.Wrapf(err, "could not create CSV %s file record", csv.metadata.name)
		importer.Logger.Error(err)
		return err
	}

	csv.metadata.fileID = record.ID

	rows, count, err := importer.prepareCSVData(csv.data, record.ID)
	if err != nil {
		return err
	}

	num, err := tx.CopyFrom(pgx.Identifier{"cclf_beneficiaries"}, []string{"file_id", "mbi"}, pgx.CopyFromRows(rows))
	if count != num {
		importer.Logger.Error("unexpected record count")
	}

	err = rtx.UpdateCCLFFileImportStatus(ctx, csv.metadata.fileID, constants.ImportComplete)
	if err != nil {
		err = errors.Wrapf(err, "could not update CSV file record for file: %s.", csv.metadata.name)
		importer.Logger.Error(err)
	}

	if err = tx.Commit(); err != nil {
		importer.Logger.Error(err.Error())
		failMsg := fmt.Sprintf("failed to commit transaction for CSV%d import file %s", csv.metadata.cclfNum, csv.metadata.name)
		return errors.Wrap(err, failMsg)
	}
	return nil
}

func (importer CSVImporter) prepareCSVData(csvfile *bytes.Reader, id uint) ([][]interface{}, int, error) {
	var rows [][]interface{}
	r := csv.NewReader(csvfile)

	_, err := r.Read()
	if err != nil {
		return nil, 0, err
	}

	count := 0

	for {
		var record []string
		record, err = r.Read()
		if err == io.EOF {
			err = nil
			break
		}
		// should there be additional validation here so that we know the mbi is a valid mbi?
		row := make([]interface{}, 2)
		row[0] = id
		row[1] = record[0]

		rows = append(rows, row)
		count++
	}
	return rows, count, err

}
