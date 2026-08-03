package postgres

import (
	"context"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
)

// PgxRepository provides repository methods that work with pgx transactions
type PgxRepository struct {
	pool *pgxv5Pool.Pool // Optional pool for stateful operations
}

// NewPgxRepositoryWithPool creates a new pgx repository instance with a connection pool
func NewPgxRepositoryWithPool(pool *pgxv5Pool.Pool) *PgxRepository {
	return &PgxRepository{pool: pool}
}

// GetCCLFFileExistsByNameTx checks if a CCLF file exists by name using pgx transaction
func (r *PgxRepository) GetCCLFFileExistsByNameTx(ctx context.Context, tx pgxv5.Tx, name string) (bool, error) {
	query := `SELECT COUNT(*) FROM cclf_files WHERE name = $1`
	var count int
	err := tx.QueryRow(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetCCLFFileExistsByName checks if a CCLF file exists by name using the repository's pool
func (r *PgxRepository) GetCCLFFileExistsByName(ctx context.Context, name string) (bool, error) {
	if r.pool == nil {
		return false, fmt.Errorf("pool not initialized")
	}

	query := `SELECT COUNT(*) FROM cclf_files WHERE name = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateCCLFFileTx creates a CCLF file record using pgx transaction
func (r *PgxRepository) CreateCCLFFileTx(ctx context.Context, tx pgxv5.Tx, cclfFile models.CCLFFile) (uint, error) {
	query := `
		INSERT INTO cclf_files (cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id uint
	err := tx.QueryRow(ctx, query,
		cclfFile.CCLFNum,
		cclfFile.Name,
		cclfFile.ACOCMSID,
		cclfFile.Timestamp,
		cclfFile.PerformanceYear,
		cclfFile.ImportStatus,
		cclfFile.Type).Scan(&id)

	return id, err
}

// CreateCCLFFile creates a CCLF file record using the repository's pool
func (r *PgxRepository) CreateCCLFFile(ctx context.Context, cclfFile models.CCLFFile) (uint, error) {
	if r.pool == nil {
		return 0, fmt.Errorf("pool not initialized")
	}

	query := `
		INSERT INTO cclf_files (cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	var id uint
	err := r.pool.QueryRow(ctx, query,
		cclfFile.CCLFNum,
		cclfFile.Name,
		cclfFile.ACOCMSID,
		cclfFile.Timestamp,
		cclfFile.PerformanceYear,
		cclfFile.ImportStatus,
		cclfFile.Type).Scan(&id)

	return id, err
}

// UpdateCCLFFileImportStatusTx updates the import status of a CCLF file using pgx transaction
func (r *PgxRepository) UpdateCCLFFileImportStatusTx(ctx context.Context, tx pgxv5.Tx, fileID uint, importStatus string) error {
	query := `UPDATE cclf_files SET import_status = $1 WHERE id = $2`
	result, err := tx.Exec(ctx, query, importStatus, fileID)
	if err != nil {
		return err
	}

	affected := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("failed to update file entry %d status to %s, no entry found", fileID, importStatus)
	}

	return nil
}

// UpdateCCLFFileImportStatus updates the import status of a CCLF file using the repository's pool
func (r *PgxRepository) UpdateCCLFFileImportStatus(ctx context.Context, fileID uint, importStatus string) error {
	if r.pool == nil {
		return fmt.Errorf("pool not initialized")
	}

	query := `UPDATE cclf_files SET import_status = $1 WHERE id = $2`
	result, err := r.pool.Exec(ctx, query, importStatus, fileID)
	if err != nil {
		return err
	}

	affected := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("failed to update file entry %d status to %s, no entry found", fileID, importStatus)
	}

	return nil
}

func (r *PgxRepository) FindOrCreateWarningAndInfoJobKey(ctx context.Context, jobID uint) error {
	if r.pool == nil {
		return fmt.Errorf("pool not initialized")
	}

	fname := constants.WarningsAndInfoFileName

	query := `
		SELECT id FROM job_keys
		WHERE job_id = $1 AND file_name = $2
		LIMIT 1`

	var id uint
	err := r.pool.QueryRow(ctx, query, jobID, fname).Scan(&id)
	if err != nil {
		if err != pgxv5.ErrNoRows {
			return fmt.Errorf("failed to check for existing job key for job %d, err: %w", jobID, err)
		}

		query = `
			INSERT INTO job_keys (job_id, file_name, resource_type)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
			RETURNING id`

		result, err := r.pool.Exec(ctx, query, jobID, fname, "OperationOutcome")
		affected := result.RowsAffected()
		if err != nil || affected == 0 {
			return fmt.Errorf("failed to create job key for job %d, err: %w", jobID, err)
		}
	}

	return nil
}
