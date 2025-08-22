package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	pgxv5 "github.com/jackc/pgx/v5"
	pgxv5Pool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func rollbackTx(ctx context.Context, tx pgxv5.Tx) {
	if err := tx.Rollback(ctx); err != nil {
		fmt.Printf("Warning: failed to rollback transaction: %v\n", err)
	}
}

type PgxRepositoryTestSuite struct {
	suite.Suite
	db   *sql.DB
	pool *pgxv5Pool.Pool
	repo *postgres.PgxRepository
}

func TestPgxRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PgxRepositoryTestSuite))
}

func (s *PgxRepositoryTestSuite) SetupSuite() {
	s.db, s.pool, _ = databasetest.CreateDatabase(s.T(), "../../../db/migrations/bcda/", true)
	s.repo = postgres.NewPgxRepositoryWithPool(s.pool)
}

func (s *PgxRepositoryTestSuite) TearDownSuite() {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.db != nil {
		s.db.Close()
	}
}

func (s *PgxRepositoryTestSuite) TearDownTest() {
	s.cleanupTestData("TEST123")
}

func (s *PgxRepositoryTestSuite) TestNewPgxRepositoryWithPool() {
	repo := postgres.NewPgxRepositoryWithPool(s.pool)
	require.NotNil(s.T(), repo)

	repoWithNilPool := postgres.NewPgxRepositoryWithPool(nil)
	require.NotNil(s.T(), repoWithNilPool)
}

func (s *PgxRepositoryTestSuite) TestGetCCLFFileExistsByNameTx() {
	ctx := context.Background()

	// non-existent file
	tx, err := s.pool.Begin(ctx)
	require.NoError(s.T(), err)
	defer rollbackTx(ctx, tx)

	exists, err := s.repo.GetCCLFFileExistsByNameTx(ctx, tx, "non-existent-file.ndjson")
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)

	// existing file
	cclfFile := createTestCCLFFile("test-file.ndjson", "TEST123")
	_, err = s.repo.CreateCCLFFileTx(ctx, tx, cclfFile)
	require.NoError(s.T(), err)

	exists, err = s.repo.GetCCLFFileExistsByNameTx(ctx, tx, "test-file.ndjson")
	require.NoError(s.T(), err)
	assert.True(s.T(), exists)
}

func (s *PgxRepositoryTestSuite) TestGetCCLFFileExistsByName() {
	ctx := context.Background()

	// non-existent file
	exists, err := s.repo.GetCCLFFileExistsByName(ctx, "non-existent-file.ndjson")
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)

	// existing file
	cclfFile := createTestCCLFFile("test-file-pool.ndjson", "TEST123")
	_, err = s.repo.CreateCCLFFile(ctx, cclfFile)
	require.NoError(s.T(), err)

	exists, err = s.repo.GetCCLFFileExistsByName(ctx, "test-file-pool.ndjson")
	require.NoError(s.T(), err)
	assert.True(s.T(), exists)

	// nil pool
	repoWithNilPool := &postgres.PgxRepository{}
	_, err = repoWithNilPool.GetCCLFFileExistsByName(ctx, "test-file.ndjson")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "pool not initialized")
}

func (s *PgxRepositoryTestSuite) TestCreateCCLFFileTx() {
	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	require.NoError(s.T(), err)
	defer rollbackTx(ctx, tx)

	cclfFile := createTestCCLFFile("test-create-tx.ndjson", "TEST123")

	id, err := s.repo.CreateCCLFFileTx(ctx, tx, cclfFile)
	require.NoError(s.T(), err)
	assert.Greater(s.T(), id, uint(0))

	exists, err := s.repo.GetCCLFFileExistsByNameTx(ctx, tx, "test-create-tx.ndjson")
	require.NoError(s.T(), err)
	assert.True(s.T(), exists)
}

func (s *PgxRepositoryTestSuite) TestCreateCCLFFile() {
	ctx := context.Background()

	cclfFile := createTestCCLFFile("test-create-pool.ndjson", "TEST123")

	id, err := s.repo.CreateCCLFFile(ctx, cclfFile)
	require.NoError(s.T(), err)
	assert.Greater(s.T(), id, uint(0))

	exists, err := s.repo.GetCCLFFileExistsByName(ctx, "test-create-pool.ndjson")
	require.NoError(s.T(), err)
	assert.True(s.T(), exists)

	repoWithNilPool := &postgres.PgxRepository{}
	_, err = repoWithNilPool.CreateCCLFFile(ctx, cclfFile)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "pool not initialized")
}

func (s *PgxRepositoryTestSuite) TestUpdateCCLFFileImportStatusTx() {
	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	require.NoError(s.T(), err)
	defer rollbackTx(ctx, tx)

	cclfFile := createTestCCLFFile("test-update-tx.ndjson", "TEST123")
	fileID, err := s.repo.CreateCCLFFileTx(ctx, tx, cclfFile)
	require.NoError(s.T(), err)

	err = s.repo.UpdateCCLFFileImportStatusTx(ctx, tx, fileID, "COMPLETED")
	require.NoError(s.T(), err)

	err = s.repo.UpdateCCLFFileImportStatusTx(ctx, tx, 99999, "COMPLETED")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to update file entry 99999 status to COMPLETED, no entry found")
}

func (s *PgxRepositoryTestSuite) TestUpdateCCLFFileImportStatus() {
	ctx := context.Background()

	cclfFile := createTestCCLFFile("test-update-pool.ndjson", "TEST123")
	fileID, err := s.repo.CreateCCLFFile(ctx, cclfFile)
	require.NoError(s.T(), err)

	err = s.repo.UpdateCCLFFileImportStatus(ctx, fileID, "COMPLETED")
	require.NoError(s.T(), err)

	err = s.repo.UpdateCCLFFileImportStatus(ctx, 99999, "COMPLETED")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to update file entry 99999 status to COMPLETED, no entry found")

	repoWithNilPool := &postgres.PgxRepository{}
	err = repoWithNilPool.UpdateCCLFFileImportStatus(ctx, fileID, "COMPLETED")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "pool not initialized")
}

func (s *PgxRepositoryTestSuite) TestPgxRepository_Integration() {
	ctx := context.Background()

	tx, err := s.pool.Begin(ctx)
	require.NoError(s.T(), err)
	defer rollbackTx(ctx, tx)

	exists, err := s.repo.GetCCLFFileExistsByNameTx(ctx, tx, "integration-test.ndjson")
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)

	cclfFile := createTestCCLFFile("integration-test.ndjson", "TEST123")
	fileID, err := s.repo.CreateCCLFFileTx(ctx, tx, cclfFile)
	require.NoError(s.T(), err)
	assert.Greater(s.T(), fileID, uint(0))

	exists, err = s.repo.GetCCLFFileExistsByNameTx(ctx, tx, "integration-test.ndjson")
	require.NoError(s.T(), err)
	assert.True(s.T(), exists)

	err = s.repo.UpdateCCLFFileImportStatusTx(ctx, tx, fileID, "IN_PROGRESS")
	require.NoError(s.T(), err)

	err = s.repo.UpdateCCLFFileImportStatusTx(ctx, tx, fileID, "COMPLETED")
	require.NoError(s.T(), err)
}

func (s *PgxRepositoryTestSuite) TestPgxRepository_ErrorHandling() {
	ctx := context.Background()

	// invalid transaction (closed)
	tx, err := s.pool.Begin(ctx)
	require.NoError(s.T(), err)
	rollbackTx(ctx, tx)

	// closed transaction
	_, err = s.repo.GetCCLFFileExistsByNameTx(ctx, tx, "test.ndjson")
	assert.Error(s.T(), err)

	cclfFile := createTestCCLFFile("test.ndjson", "TEST123")
	_, err = s.repo.CreateCCLFFileTx(ctx, tx, cclfFile)
	assert.Error(s.T(), err)

	err = s.repo.UpdateCCLFFileImportStatusTx(ctx, tx, 1, "COMPLETED")
	assert.Error(s.T(), err)
}

// TestPgxRepository_ConcurrentAccess tests concurrent access to the repository
func (s *PgxRepositoryTestSuite) TestPgxRepository_ConcurrentAccess() {
	ctx := context.Background()

	const numFiles = 5
	results := make(chan error, numFiles)

	for i := 0; i < numFiles; i++ {
		go func(index int) {
			cclfFile := createTestCCLFFile(fmt.Sprintf("concurrent-test-%d.ndjson", index), "TEST123")
			_, err := s.repo.CreateCCLFFile(ctx, cclfFile)
			results <- err
		}(i)
	}

	for i := 0; i < numFiles; i++ {
		err := <-results
		assert.NoError(s.T(), err)
	}

	for i := 0; i < numFiles; i++ {
		exists, err := s.repo.GetCCLFFileExistsByName(ctx, fmt.Sprintf("concurrent-test-%d.ndjson", i))
		require.NoError(s.T(), err)
		assert.True(s.T(), exists)
	}
}

func (s *PgxRepositoryTestSuite) TestPgxRepository_EdgeCases() {
	ctx := context.Background()

	// empty filename
	exists, err := s.repo.GetCCLFFileExistsByName(ctx, "")
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)

	// very long filename
	longName := strings.Repeat("a", 1000)
	exists, err = s.repo.GetCCLFFileExistsByName(ctx, longName)
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)

	// special characters in filename
	exists, err = s.repo.GetCCLFFileExistsByName(ctx, "test-file-with-special-chars-!@#$%^&*().ndjson")
	require.NoError(s.T(), err)
	assert.False(s.T(), exists)

	// zero file ID
	err = s.repo.UpdateCCLFFileImportStatus(ctx, 0, "COMPLETED")
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "failed to update file entry 0 status to COMPLETED, no entry found")
}

func createTestCCLFFile(name, acoCMSID string) models.CCLFFile {
	return models.CCLFFile{
		CCLFNum:         8,
		Name:            name,
		ACOCMSID:        acoCMSID,
		Timestamp:       time.Now().Round(time.Millisecond),
		PerformanceYear: 2024,
		ImportStatus:    "PENDING",
		Type:            models.FileTypeDefault,
	}
}

func (s *PgxRepositoryTestSuite) cleanupTestData(cmsID string) {
	// Delete CCLF beneficiaries first due to foreign key constraints
	_, err := s.db.Exec("DELETE FROM cclf_beneficiaries WHERE file_id IN (SELECT id FROM cclf_files WHERE aco_cms_id = $1)", cmsID)
	assert.NoError(s.T(), err)

	// Delete CCLF files
	_, err = s.db.Exec("DELETE FROM cclf_files WHERE aco_cms_id = $1", cmsID)
	assert.NoError(s.T(), err)
}

func TestPgxRepository_CCLFFileOperations(t *testing.T) {
	pool := database.ConnectPool()
	defer pool.Close()

	repo := postgres.NewPgxRepositoryWithPool(pool)
	assert.NotNil(t, repo)
}

func TestPgxRepository_NewPgxRepositoryWithPool(t *testing.T) {
	pool := database.ConnectPool()
	defer pool.Close()

	repo := postgres.NewPgxRepositoryWithPool(pool)
	require.NotNil(t, repo)

	_, ok := interface{}(repo).(*postgres.PgxRepository)
	assert.True(t, ok, "NewPgxRepositoryWithPool should return *PgxRepository")
}

func TestPgxRepository(t *testing.T) {
	repo := postgres.NewPgxRepositoryWithPool(nil)
	assert.NotNil(t, repo)

	cclfFile := createTestCCLFFile("test-file.ndjson", "TEST123")
	assert.Equal(t, "test-file.ndjson", cclfFile.Name)
	assert.Equal(t, "TEST123", cclfFile.ACOCMSID)
	assert.Equal(t, 8, cclfFile.CCLFNum)
	assert.Equal(t, 2024, cclfFile.PerformanceYear)
	assert.Equal(t, "PENDING", cclfFile.ImportStatus)
	assert.Equal(t, models.FileTypeDefault, cclfFile.Type)
}
