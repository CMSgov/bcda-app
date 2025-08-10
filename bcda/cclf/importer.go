package cclf

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
)

// A cclf8Importer is not safe for concurrent use by multiple goroutines.
// It should be scoped to a single *sql.Tx
type cclf8Importer struct {
	ctx context.Context

	scanner        *bufio.Scanner
	reportInterval int
	cclfFileID     uint // CCLFFile ID that will be associated with all created benes

	recordCount          int
	importCount          int
	processedMBIs        map[string]struct{}
	logger               logrus.FieldLogger
	expectedRecordLength int
}

func (importer *cclf8Importer) Next() bool {
	// Loops through the scanner until we either:
	// 1. Encounter the end of the data
	// 2. Find an MBI that we have not processed yet
	//
	// This logic exists in the Next() function because it
	// is the only way for us to ignore an already ingested MBI.
	// If we made this check in Values() and return an error or
	// return empty data, then the copy will fail.
	// NOTE: This choice was based on pgx v3.1.0.
	//
	// Since we're using the bufio.Scanner, we can make multiple calls
	// to scanner.Bytes() without advancing the cursor.
	for {
		hasNext := importer.scanner.Scan()
		if !hasNext {
			return hasNext
		}

		importer.recordCount++
		mbi := importer.getMBI()
		// We've already processed this MBI before
		if _, found := importer.processedMBIs[mbi]; found {
			continue
		}

		// We have an MBI we haven't processed yet
		importer.processedMBIs[mbi] = struct{}{}
		return true
	}
}

func (importer *cclf8Importer) Values() ([]interface{}, error) {
	close := metrics.NewChild(importer.ctx, "importCCLF8-benecreate")
	defer close()

	// Verify record length
	b := importer.scanner.Bytes()
	trimmed := bytes.TrimSpace(b)

	// Currently only errors if record is longer than expected
	if len(trimmed) == 0 || len(trimmed) > importer.expectedRecordLength {
		err := fmt.Errorf("incorrect record length for file (expected: %d, actual: %d)", importer.expectedRecordLength, len(trimmed))
		importer.logger.Error(err)
		return nil, err
	}

	// Use Int4 because we store file_id as an integer
	fileID := int32(importer.cclfFileID)

	mbi := importer.getMBI()

	importer.importCount++
	if importer.importCount%importer.reportInterval == 0 {
		importer.logger.Infof("CCLF8 records imported: %d\n", importer.importCount)
	}

	return []interface{}{fileID, mbi}, nil
}

// Err allows us to report back the the CopyFrom if and when
// the underlying context has been stopped.
// NOTE: This is specifc to pgx v3. In pgx v4, the CopyFrom function accepts a context argument
func (importer *cclf8Importer) Err() error {
	return importer.ctx.Err()
}

// getMBI returns the current MBI contained at the scanner's current cursor.
// Since calls the bufio.Scanner.Bytes() does not advance the cursor, this call
// can be invoked multiple times and return the same result.
func (importer *cclf8Importer) getMBI() string {
	const (
		mbiStart, mbiEnd = 0, 11
	)
	b := importer.scanner.Bytes()
	return string(bytes.TrimSpace(b[mbiStart:mbiEnd]))
}

// CopyFrom writes all of the beneficiary data captured in the scanner to the beneficiaries table.
// It returns the number of rows written along with any error that occurred.
func CopyFrom(ctx context.Context, tx pgx.Tx, scanner *bufio.Scanner, fileID uint, reportInterval int, logger logrus.FieldLogger, expectedRecordLength int) (int, int, error) {
	importer := &cclf8Importer{
		scanner:    scanner,
		ctx:        ctx,
		cclfFileID: fileID,

		reportInterval:       reportInterval,
		processedMBIs:        make(map[string]struct{}),
		logger:               logger,
		expectedRecordLength: expectedRecordLength,
	}
	tableName := pgx.Identifier{"cclf_beneficiaries"}
	importedCount, err := tx.CopyFrom(ctx, tableName, []string{"file_id", "mbi"}, importer)
	return int(importedCount), importer.recordCount, err
}

// CopyFromSql writes all of the beneficiary data captured in the scanner to the beneficiaries table using standard sql.
// It returns the number of rows written along with any error that occurred.
func CopyFromSql(ctx context.Context, tx *sql.Tx, scanner *bufio.Scanner, fileID uint, reportInterval int, logger logrus.FieldLogger, expectedRecordLength int) (int, int, error) {
	importer := &cclf8Importer{
		scanner:    scanner,
		ctx:        ctx,
		cclfFileID: fileID,

		reportInterval:       reportInterval,
		processedMBIs:        make(map[string]struct{}),
		logger:               logger,
		expectedRecordLength: expectedRecordLength,
	}

	// Prepare bulk insert statement
	stmt, err := tx.PrepareContext(ctx, "INSERT INTO cclf_beneficiaries (file_id, mbi) VALUES ($1, $2)")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var importedCount int
	for importer.Next() {
		values, err := importer.Values()
		if err != nil {
			return importedCount, importer.recordCount, err
		}

		_, err = stmt.ExecContext(ctx, values[0], values[1])
		if err != nil {
			return importedCount, importer.recordCount, fmt.Errorf("failed to insert row: %w", err)
		}
		importedCount++

		if importedCount%reportInterval == 0 {
			logger.Infof("CCLF8 records imported: %d\n", importedCount)
		}
	}

	return importedCount, importer.recordCount, nil
}
