package cclf

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/cclf/metrics"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type importer interface {
	// do(ctx context.Context, tx *sql.Tx, cclfBeneficiary models.CCLFBeneficiary) error
	do(ctx context.Context, tx *sql.Tx, data interface{}) error

	// flush should be called once the import process is complete.
	// This will guarantee any remaining work involved with the importer is complete.
	flush(ctx context.Context) error
}

// A cclf8Importer is not safe for concurrent use by multiple goroutines.
// It should be scoped to a single *sql.Tx
type cclf8Importer struct {
	logger *logrus.Logger

	inprogress *sql.Stmt

	pendingQueries    int
	maxPendingQueries int
}

// validates that cclf8Importer implements the interface
// var _ importer = &cclf8Importer{}

func (cclfImporter *cclf8Importer) do(ctx context.Context, tx *sql.Tx, data interface{}) error {
	bene, ok := data.(models.CCLFBeneficiary)
	if !ok {
		return errors.New("invalid type sent, expected models.CCLFBeneficiary")
	}
	if cclfImporter.inprogress == nil {
		if err := cclfImporter.refreshStatement(ctx, tx); err != nil {
			return errors.Wrap(err, "failed to refresh statement")
		}
	}

	if cclfImporter.pendingQueries >= cclfImporter.maxPendingQueries {
		if err := cclfImporter.flush(ctx); err != nil {
			return errors.Wrap(err, "failed to flush statement")
		}
		if err := cclfImporter.refreshStatement(ctx, tx); err != nil {
			return errors.Wrap(err, "failed to refresh statement")
		}
		cclfImporter.pendingQueries = 0
	}

	close := metrics.NewChild(ctx, "importCCLF8-benecreate")
	defer close()

	_, err := cclfImporter.inprogress.Exec(bene.FileID, bene.HICN, bene.MBI)
	if err != nil {
		fmt.Println("Could not create CCLF8 beneficiary record.")
		err = errors.Wrap(err, "could not create CCLF8 beneficiary record")
		cclfImporter.logger.Error(err)
		return err
	}
	cclfImporter.pendingQueries++
	return nil
}

func (cclfImporter *cclf8Importer) flush(ctx context.Context) error {
	stmt := cclfImporter.inprogress
	if stmt == nil {
		cclfImporter.logger.Warn("No statement to flush.")
		return nil
	}

	if _, err := stmt.Exec(); err != nil {
		return err
	}

	if err := stmt.Close(); err != nil {
		return err
	}

	return nil
}

func (cclfImporter *cclf8Importer) refreshStatement(ctx context.Context, tx *sql.Tx) error {
	stmt, err := tx.PrepareContext(ctx, pq.CopyIn("cclf_beneficiaries", "file_id", "hicn", "mbi"))
	if err != nil {
		return err
	}

	cclfImporter.inprogress = stmt
	return nil
}
