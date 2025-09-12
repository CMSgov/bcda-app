package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type TestDatabaseContainer struct {
	Container        *postgres.PostgresContainer
	ConnectionString string
	migrations       string
	testdata         string
}

// ExecuteFile will execute a *.sql file for a database container.
// Sql files for testing purposes should be under a package's testdata/ directory.
func (td *TestDatabaseContainer) ExecuteFile(path string) (int64, error) {
	ctx := context.Background()
	var rows int64

	file, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return rows, fmt.Errorf("failed to stat file: %w", err)
	}

	if filepath.Ext(file.Name()) != ".sql" {
		return rows, fmt.Errorf("failed execute file: not a .sql file")
	}

	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return rows, fmt.Errorf("failed to open file: %w", err)
	}

	sql := string(content)

	pgx, err := td.NewPgxConnection()
	if err != nil {
		return rows, fmt.Errorf("failed to connect to container database: %w", err)
	}
	defer pgx.Close(ctx)
	result, err := pgx.Exec(ctx, sql)

	if err != nil {
		return rows, fmt.Errorf("failed to execute sql: %ww", err)
	}
	rows = result.RowsAffected()
	if rows == 0 {
		return rows, fmt.Errorf("zero rows affected")
	}

	return rows, err
}

// ExecuteFile will execute all *.sql files for the provided dirpath.
// Is it recommended to use the package's testdata/ directory to add test files.
// A package's testdata/ dir can be retrieved with GetTestDataDir().
func (td *TestDatabaseContainer) ExecuteDir(dirpath string) error {
	var err error
	testDir, err := os.Stat(dirpath)
	if err != nil {
		return fmt.Errorf("failed to get testdata directory: %w", err)
	}

	if !testDir.IsDir() {
		return errors.New("failed to get directory; path is not a directory")
	}

	err = filepath.Walk(dirpath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}
		if !info.IsDir() && (filepath.Ext(info.Name()) == ".sql") {
			_, err = td.ExecuteFile(path)
			if err != nil {
				return fmt.Errorf("failed to execute sql file %s with error: %w", path, err)
			}
		}
		return err
	})
	return err
}

// CreateSnapshot will create a snapshot for a given name. Close any active connections to the database
// before taking a snapshot.
func (td *TestDatabaseContainer) CreateSnapshot(name string) error {
	err := td.Container.Snapshot(context.Background(), postgres.WithSnapshotName(name))
	if err != nil {
		return fmt.Errorf("failed to restore container database snapshot: %w", err)
	}
	return nil
}

// RestoreSnapshot will restore the snapshot that is taken after the database container
// has had the initial migrations and data seed applied. If no name is provided, it will restore
// the default snapshot. "Base" will restore the database to it's init state.
func (td *TestDatabaseContainer) RestoreSnapshot(name string) error {
	err := td.Container.Restore(context.Background(), postgres.WithSnapshotName(name))
	if err != nil {
		return fmt.Errorf("failed to restore container database snapshot: %w", err)
	}
	return nil
}

// Return a pgx connection for a given database container.
func (td *TestDatabaseContainer) NewPgxConnection() (*pgx.Conn, error) {
	pgx, err := pgx.Connect(context.Background(), td.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection to container database: %w", err)
	}
	return pgx, nil
}

// Return a sql/db connection for a given database container.
func (td *TestDatabaseContainer) NewSqlDbConnection() (*sql.DB, error) {
	db, err := sql.Open("postgres", td.ConnectionString+"sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to open connection to container database: %w", err)
	}
	return db, nil
}

// Return a pgx pool for a given database container.
func (td *TestDatabaseContainer) NewPgxPoolConnection() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), td.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool for container database: %w", err)
	}
	return pool, nil
}

// runMigrations runs the production migrations to the local database so there is no drift between prod and local development.
func (td *TestDatabaseContainer) runMigrations() error {
	m, err := migrate.New("file:"+td.migrations, td.ConnectionString+"sslmode=disable")
	if err != nil {
		return fmt.Errorf("failed to get migrations: %w", err)
	}
	err = m.Up()
	if err != nil {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	err, _ = m.Close()
	if err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// initSeed will apply the baseline data to the the database with newly run migrations.
// For applying test or scenario specific data, utilize ExecuteFile or ExecuteDir.
func (td *TestDatabaseContainer) initSeed() error {
	err := td.ExecuteDir(td.testdata)
	if err != nil {
		return fmt.Errorf("failed to seed database container: %w", err)
	}
	return nil
}

// Returns a new postgres container with migrations from db/migrations/bcda applied and seed
// data from db/testdata applied.
func NewTestDatabaseContainer() (TestDatabaseContainer, error) {
	ctx := context.Background()
	c, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("bcda"),
		postgres.WithUsername("toor"),
		postgres.WithPassword("foobar"),
		postgres.BasicWaitStrategies())

	if err != nil {
		return TestDatabaseContainer{}, fmt.Errorf("failed to create database container: %w", err)
	}

	conn, err := c.ConnectionString(ctx)
	if err != nil {
		return TestDatabaseContainer{}, fmt.Errorf("failed to get connection string for container database: %w", err)
	}

	tdc := TestDatabaseContainer{
		Container:        c,
		ConnectionString: conn,
	}

	err = tdc.getSetupDirs()
	if err != nil {
		return TestDatabaseContainer{}, fmt.Errorf("failed to get testdata or migrations dirs: %w", err)
	}

	err = tdc.runMigrations()
	if err != nil {
		return TestDatabaseContainer{}, fmt.Errorf("failed to apply migrations to container database: %w", err)
	}

	err = tdc.initSeed()
	if err != nil {
		return TestDatabaseContainer{}, fmt.Errorf("failed to add test data to container database: %w", err)
	}

	err = tdc.CreateSnapshot("Base")
	if err != nil {
		return TestDatabaseContainer{}, err
	}

	return tdc, nil

}

// GetTestDataDir is a helper function that will return the testdata directory for package in which
// it is invoked. If the testdata directory has been created in another package or the files exist
// outside the package, they will not be found.
func GetTestDataDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	testDir := filepath.Join(filepath.Clean(currentDir), "testdata")
	_, err = os.Stat(testDir)
	if err != nil {
		return "", fmt.Errorf("failed to get testdata directory: %w", err)
	}

	return testDir, err
}

// getSetupDirs ensures that we get the db/testdata and migrations directories no matter where NewTestDatabaseContainer is called.
func (td *TestDatabaseContainer) getSetupDirs() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	testDir := filepath.Join("db", "testdata")
	migrationsDir := filepath.Join("db", "migrations", "bcda")
	dirPaths := []string{testDir, migrationsDir}

	for _, v := range dirPaths {
		for {
			targetPath := filepath.Join(filepath.Clean(currentDir), filepath.Clean(v))
			_, err := os.Stat(targetPath)
			if err == nil {
				if strings.Contains(v, "testdata") {
					td.testdata = targetPath
				}
				if strings.Contains(v, "migrations") {
					td.migrations = targetPath
				}
				break
			}
			if !os.IsNotExist(err) {
				return fmt.Errorf("error checking path %s: %w", targetPath, err)
			}

			parentDir := filepath.Dir(currentDir)
			if parentDir == currentDir {
				return fmt.Errorf("file or directory '%s' not found in parent directories", "db/testdata")
			}
			currentDir = parentDir
		}
	}
	return nil
	// for {
	// 	targetPath := filepath.Join(filepath.Clean(currentDir), "db", "testdata")
	// 	_, err := os.Stat(targetPath)
	// 	if err == nil {
	// 		return targetPath, nil
	// 	}
	// 	if !os.IsNotExist(err) {
	// 		return "", fmt.Errorf("error checking path %s: %w", targetPath, err)
	// 	}

	// 	parentDir := filepath.Dir(currentDir)
	// 	if parentDir == currentDir {
	// 		return "", fmt.Errorf("file or directory '%s' not found in parent directories", "db/testdata")
	// 	}
	// 	currentDir = parentDir
	// }
}
