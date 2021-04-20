package cclf

import (
	"bufio"
	"bytes"
	"context"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/pgtype"
)

// A cclf8Importer is not safe for concurrent use by multiple goroutines.
// It should be scoped to a single *sql.Tx
type cclf8Importer struct {
	ctx context.Context

	scanner        *bufio.Scanner
	reportInterval int
	cclfFileID     uint // CCLFFile ID that will be associated with all created benes

	importCount   int
	processedMBIs map[string]struct{}
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

	// Use Int4 because we store file_id as an integer
	fileID := &pgtype.Int4{}
	if err := fileID.Set(importer.cclfFileID); err != nil {
		return nil, err
	}

	mbi := &pgtype.BPChar{}
	if err := mbi.Set(importer.getMBI()); err != nil {
		return nil, err
	}

	importer.importCount++
	if importer.importCount%importer.reportInterval == 0 {
		fmt.Printf("CCLF8 records imported: %d\n", importer.importCount)
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
func CopyFrom(ctx context.Context, tx *pgx.Tx, scanner *bufio.Scanner, fileID uint, reportInterval int) (int, error) {
	importer := &cclf8Importer{
		scanner:    scanner,
		ctx:        ctx,
		cclfFileID: fileID,

		reportInterval: reportInterval,
		processedMBIs:  make(map[string]struct{}),
	}
	tableName := pgx.Identifier([]string{"cclf_beneficiaries"})
	return tx.CopyFrom(tableName, []string{"file_id", "mbi"}, importer)
}
