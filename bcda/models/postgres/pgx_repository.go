package postgres

import (
	"context"
	"fmt"

	"github.com/CMSgov/bcda-app/bcda/models"
	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
)

// PgxRepository provides repository methods that work with pgx transactions
type PgxRepository struct {
	pool *pgxv5Pool.Pool // Optional pool for stateful operations
}

// NewPgxRepository creates a new stateless pgx repository instance
func NewPgxRepository() *PgxRepository {
	return &PgxRepository{}
}

// NewPgxRepositoryWithPool creates a new pgx repository instance with a connection pool
func NewPgxRepositoryWithPool(pool *pgxv5Pool.Pool) *PgxRepository {
	return &PgxRepository{pool: pool}
}

// GetCCLFFileExistsByName checks if a CCLF file exists by name using pgx transaction
func (r *PgxRepository) GetCCLFFileExistsByName(ctx context.Context, tx pgxv5.Tx, name string) (bool, error) {
	query := `SELECT COUNT(*) FROM cclf_files WHERE name = $1`
	var count int
	err := tx.QueryRow(ctx, query, name).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetCCLFFileExistsByNameWithPool checks if a CCLF file exists by name using the repository's pool
func (r *PgxRepository) GetCCLFFileExistsByNameWithPool(ctx context.Context, name string) (bool, error) {
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

// CreateCCLFFile creates a CCLF file record using pgx transaction
func (r *PgxRepository) CreateCCLFFile(ctx context.Context, tx pgxv5.Tx, cclfFile models.CCLFFile) (uint, error) {
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

// CreateCCLFFileWithPool creates a CCLF file record using the repository's pool
func (r *PgxRepository) CreateCCLFFileWithPool(ctx context.Context, cclfFile models.CCLFFile) (uint, error) {
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

// UpdateCCLFFileImportStatus updates the import status of a CCLF file using pgx transaction
func (r *PgxRepository) UpdateCCLFFileImportStatus(ctx context.Context, tx pgxv5.Tx, fileID uint, importStatus string) error {
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

// UpdateCCLFFileImportStatusWithPool updates the import status of a CCLF file using the repository's pool
func (r *PgxRepository) UpdateCCLFFileImportStatusWithPool(ctx context.Context, fileID uint, importStatus string) error {
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

// DeleteCCLFFile deletes a CCLF file record using pgx transaction
func (r *PgxRepository) DeleteCCLFFile(ctx context.Context, tx pgxv5.Tx, fileID uint) error {
	query := `DELETE FROM cclf_files WHERE id = $1`
	result, err := tx.Exec(ctx, query, fileID)
	if err != nil {
		return err
	}

	affected := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("failed to delete file entry %d, no entry found", fileID)
	}

	return nil
}

// DeleteCCLFFileWithPool deletes a CCLF file record using the repository's pool
func (r *PgxRepository) DeleteCCLFFileWithPool(ctx context.Context, fileID uint) error {
	if r.pool == nil {
		return fmt.Errorf("pool not initialized")
	}

	query := `DELETE FROM cclf_files WHERE id = $1`
	result, err := r.pool.Exec(ctx, query, fileID)
	if err != nil {
		return err
	}

	affected := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("failed to delete file entry %d, no entry found", fileID)
	}

	return nil
}
