package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
}

// ExecuteFile will execute a *.sql file for a database container.
// Sql files for testing purposes should be under a package's 'testdata' directory.
func (td *TestDatabaseContainer) ExecuteFile(filepath string) (int64, error) {
	ctx := context.Background()
	var rows int64
	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Sprintf("failed to open file: %s", err)
		return rows, err
	}

	sql := string(content)

	pgx, err := td.NewPgxConnection()
	if err != nil {
		fmt.Sprintf("failed to connect to container database: %s", err)
		return rows, err
	}
	defer pgx.Close(ctx)
	result, err := pgx.Exec(ctx, sql)

	if err != nil {
		fmt.Sprintf("failed to execute sql: %s", err)
		return rows, err
	} else {
		rows = result.RowsAffected()
	}

	return rows, err
}

// ExecuteFile will execute all *.sql files in a given dir for a database container.
// There must be a 'testdata' directory in the current working directory as the *_test.go file.
func (td *TestDatabaseContainer) ExecuteDir(dirpath string) error {
	// TODO: this executes all files in the testdata dir; need to update to execute child directories
	// within testdata
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	testDir := filepath.Join(currentDir, "testdata")
	_, err = os.Stat(testDir)
	if err != nil {
		return fmt.Errorf("failed to get testdata directory: %w", err)
	}

	err = filepath.Walk(testDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", path, err)
			return err
		}
		if !info.IsDir() {
			td.ExecuteFile(path)
		}
		return nil
	})
	return nil
}

// CreateSnapshot will create a snapshot for a given name. Close any active connections to the database
// before taking a snapshot.
func (td *TestDatabaseContainer) CreateSnapshot(name string) error {
	err := td.Container.Snapshot(context.Background(), postgres.WithSnapshotName(name))
	if err != nil {
		fmt.Sprintf("failed to restore container database snapshot: %s", err)
		return err
	}
	return nil
}

// RestoreSnapshot will restore the snapshot that is taken after the database container
// has had the initial migrations and data seed applied. If no name is provided, it will restore
// the default snapshot. "Base" will restore the database to it's init state.
func (td *TestDatabaseContainer) RestoreSnapshot(name string) error {
	err := td.Container.Restore(context.Background(), postgres.WithSnapshotName(name))
	if err != nil {
		fmt.Sprintf("failed to restore container database snapshot: %s", err)
		return err
	}
	return nil
}

// Return a pgx connection for a given database container.
func (td *TestDatabaseContainer) NewPgxConnection() (*pgx.Conn, error) {
	pgx, err := pgx.Connect(context.Background(), td.ConnectionString)
	if err != nil {
		fmt.Sprintf("failed to open connection to container database: %s", err)
		return nil, err
	}
	return pgx, nil
}

// Return a sql/db connection for a given database container.
func (td *TestDatabaseContainer) NewSqlDbConnection() (*sql.DB, error) {
	db, err := sql.Open("postgres", td.ConnectionString+"sslmode=disable")
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Return a pgx pool for a given database container.
func (td *TestDatabaseContainer) NewPgxPoolConnection() (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), td.ConnectionString)
	if err != nil {
		fmt.Sprintf("failed to create pool for container database: %s", err)
		return nil, err
	}
	return pool, nil
}

// runMigrations runs the production migrations to the local database so there is no drift between prod and local development.
func (td *TestDatabaseContainer) runMigrations() error {
	m, err := migrate.New("file://../../../db/migrations/bcda/", td.ConnectionString+"sslmode=disable")
	err = m.Up()
	if err != nil {
		fmt.Sprintf("failed to create database container: %s", err)
		return err
	}
	m.Close()
	return nil
}

// initSeed will apply the baseline data to the the database with newly run migrations.
// For applying test or scenario specific data, utilize ExecuteFile or ExecuteDir.
func (td *TestDatabaseContainer) initSeed() error {
	filePath, err := getSeedDir()
	if err != nil {
		fmt.Println("failed to get db/testdata dir", err)
		return err
	}

	rowsAffected, err := td.ExecuteFile(filepath.Join(filePath, "insert_acos.sql"))
	//rowsAffected, err := td.ExecuteFile(filepath.Join(filePath, "insert_acos.sql"))
	if err != nil {
		fmt.Sprintf("failed to seed database container: %s", err)
		return err
	}
	if rowsAffected == 0 {
		return errors.New("failed to seed init data; zero affected rows")
	}
	return nil
}

// Returns a new postgres container with migrations from db/migrations/bcda applied and seeed
// data from db/seeddata applied.
func NewTestDatabaseContainer() (TestDatabaseContainer, error) {
	ctx := context.Background()
	c, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("bcda"),
		postgres.WithUsername("toor"),
		postgres.WithPassword("foobar"),
		postgres.BasicWaitStrategies(),
	)

	if err != nil {
		fmt.Sprintf("failed to create database container: %s", err)
		return TestDatabaseContainer{}, err
	}

	conn, err := c.ConnectionString(ctx)
	if err != nil {
		fmt.Sprintf("failed to get connection string for container database: %s", err)
		return TestDatabaseContainer{}, err
	}

	tdc := TestDatabaseContainer{
		Container:        c,
		ConnectionString: conn,
	}

	err = tdc.runMigrations()
	if err != nil {
		fmt.Sprintf("failed to apply migrations to container database: %s", err)
		return TestDatabaseContainer{}, err
	}

	err = tdc.initSeed()
	if err != nil {
		fmt.Sprintf("failed to add test data to container database: %s", err)
		return TestDatabaseContainer{}, err
	}

	err = tdc.CreateSnapshot("Base")
	if err != nil {
		return TestDatabaseContainer{}, err
	}

	return tdc, nil

}

// getSeedDir ensures that we get the db/testdata folder no matter where NewTestDatabaseContainer is called.
func getSeedDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	for {
		targetPath := filepath.Join(currentDir, "db", "testdata")
		_, err := os.Stat(targetPath)
		if err == nil {
			return targetPath, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("error checking path %s: %w", targetPath, err)
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("file or directory '%s' not found in parent directories", "db/testdata")
		}
		currentDir = parentDir
	}
}
