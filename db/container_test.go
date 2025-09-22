package db

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type DatabaseContainerTestSuite struct {
	suite.Suite
	ctr TestDatabaseContainer
}

func (s *DatabaseContainerTestSuite) SetupSuite() {
	var err error
	s.ctr, err = NewTestDatabaseContainer()
	require.NoError(s.T(), err)
}

func (s *DatabaseContainerTestSuite) SetupSubTest() {
	err := s.ctr.RestoreSnapshot("")
	require.NoError(s.T(), err)
}

func TestDatabaseContainerTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseContainerTestSuite))
}

func (s *DatabaseContainerTestSuite) TestExecuteFile() {
	uuid := uuid.New()
	validSql := fmt.Sprintf("INSERT INTO ACOS (uuid, cms_id, name, client_id, termination_details) VALUES ('%s', 'A0001', 'Test DB', '%s', null);", uuid, uuid)
	tests := []struct {
		name     string
		filename string
		text     string
		expRows  int64
		expErr   bool
	}{
		{"Execute valid SQL", "insert_acos-*.sql", validSql, int64(1), false},
		{"Execute empty file", "insert_empty-*.sql", "", int64(0), true},
		{"Execute invalid SQL", "insert_invalid-*.sql", "insert into foo (id) values ('bar')", int64(0), true},
		{"Execute non SQL files", "testexecutefile-*.foobar", validSql, int64(0), true},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tempDir := s.T().TempDir()

			tmpFile, err := os.CreateTemp(tempDir, tt.filename)
			if err != nil {
				s.T().Fatalf("failed to create temporary file: %v", err)
			}
			defer tmpFile.Close()

			data := []byte(tt.text)
			if _, err := tmpFile.Write(data); err != nil {
				s.T().Fatalf("failed to write to temporary file: %v", err)
			}

			rows, err := s.ctr.ExecuteFile(tmpFile.Name())
			if tt.expErr == false {
				assert.NoError(s.T(), err)
				assert.Equal(s.T(), tt.expRows, rows)
			} else {
				assert.NotNil(s.T(), err)
			}

		})
	}
}

func (s *DatabaseContainerTestSuite) TestExecuteDir() {

	uuid := uuid.New()
	validInsert := fmt.Sprintf("INSERT INTO ACOS (uuid, cms_id, name, client_id, termination_details) VALUES ('%s', 'A0001', 'Test DB', '%s', null);", uuid, uuid)
	validCreate := "select * from acos"
	invalidSQL := "insert into foo (id) values ('bar')"

	tests := []struct {
		name   string
		files  []string
		expErr bool
	}{
		{"Execute valid dir path with valid files", []string{validInsert, validCreate}, false},
		{"Execute valid dir path with invalid files", []string{validCreate, invalidSQL}, true},
		{"Execute empty dir", []string{}, false},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			tempDir := s.T().TempDir()

			for _, v := range tt.files {
				tmpFile, err := os.CreateTemp(tempDir, "test-*.sql")
				if err != nil {
					s.T().Fatalf("failed to create temporary file: %v", err)
				}
				defer tmpFile.Close()

				data := []byte(v)
				if _, err := tmpFile.Write(data); err != nil {
					s.T().Fatalf("failed to write to temporary file: %v", err)
				}
			}

			err := s.ctr.ExecuteDir(tempDir)
			if tt.expErr == false {
				assert.NoError(s.T(), err)

			} else {
				assert.NotNil(s.T(), err)
			}

		})
	}

}

func (s *DatabaseContainerTestSuite) TestExecuteDirInvalidPath() {
	tempDir := s.T().TempDir()

	tmpFile, err := os.CreateTemp(tempDir, "test-*.sql")
	if err != nil {
		s.T().Fatalf("failed to create temporary file: %v", err)
	}

	tests := []struct {
		name   string
		path   string
		expErr bool
	}{

		{"Execute dir with file path input", tmpFile.Name(), true},
		{"Execute dir with invalid path input", uuid.New(), true},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {

			err := s.ctr.ExecuteDir(tt.path)
			assert.NotNil(s.T(), err)
		})
	}

}

func (s *DatabaseContainerTestSuite) TestCreateSnapshot() {
	err := s.ctr.CreateSnapshot("test")
	assert.Nil(s.T(), err)
	err = s.ctr.CreateSnapshot("")
	assert.Nil(s.T(), err)
}

func (s *DatabaseContainerTestSuite) TestRestoreSnapshot() {
	tests := []struct {
		name     string
		snapshot string
		expErr   bool
	}{

		{"Restore snapshot with name", "test", false},
		{"Restore snapshot with empty string", "", false},
		{"Restore snapshot with invalid name", "invalidname", true},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var err error
			if tt.expErr == false {
				err = s.ctr.CreateSnapshot(tt.snapshot)
				assert.Nil(s.T(), err)
			}

			ctx := context.Background()

			c, err := s.ctr.NewPgxConnection()
			assert.Nil(s.T(), err)

			_, err = c.Exec(ctx, "CREATE TABLE foobar (id int)")
			assert.Nil(s.T(), err)

			c.Close(ctx)

			err = s.ctr.RestoreSnapshot(tt.snapshot)
			if tt.expErr == true {
				assert.NotNil(s.T(), err)
			} else {
				assert.Nil(s.T(), err)

				c, err = s.ctr.NewPgxConnection()
				assert.Nil(s.T(), err)
				_, err := c.Query(context.Background(), "select count(*) from foobar")
				assert.NotNil(s.T(), err)
				assert.Contains(s.T(), err.Error(), "does not exist")

			}
			c.Close(ctx)

		})
	}
}

func (s *DatabaseContainerTestSuite) TestNewPgxConnection() {
	ctx := context.Background()
	c, err := s.ctr.NewPgxConnection()
	assert.Nil(s.T(), err)
	row := c.QueryRow(ctx, "select count(*) from acos;")
	assert.Nil(s.T(), err)

	var count int
	err = row.Scan(&count)
	assert.Nil(s.T(), err)

	assert.Nil(s.T(), err)
	assert.Greater(s.T(), count, 0)
	c.Close(ctx)
}

func (s *DatabaseContainerTestSuite) TestNewSqlDbConnection() {
	db, err := s.ctr.NewSqlDbConnection()
	assert.Nil(s.T(), err)
	var count int
	err = db.QueryRow("select count(*) from acos;").Scan(&count)
	assert.Nil(s.T(), err)
	db.Close()
}

func (s *DatabaseContainerTestSuite) TestNewPgxPoolConnection() {
	pool, err := s.ctr.NewPgxPoolConnection()
	assert.Nil(s.T(), err)
	row := pool.QueryRow(context.Background(), "select count(*) from acos;")
	assert.Nil(s.T(), err)

	var count int
	err = row.Scan(&count)
	assert.Nil(s.T(), err)

	assert.Nil(s.T(), err)
	assert.Greater(s.T(), count, 0)
	pool.Close()

}

type UnexportedMethodsTestSuite struct {
	suite.Suite
	postgresCtr *postgres.PostgresContainer
	connString  string
}

func (s *UnexportedMethodsTestSuite) SetupTest() {
	ctx := context.Background()
	c, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("bcda"),
		postgres.WithUsername("toor"),
		postgres.WithPassword("foobar"),
		postgres.BasicWaitStrategies())

	require.NoError(s.T(), err)
	s.postgresCtr = c

	s.connString, err = c.ConnectionString(ctx)
	require.NoError(s.T(), err)

}

func TestUnexportedMethodsTestSuiteTestSuite(t *testing.T) {
	suite.Run(t, new(UnexportedMethodsTestSuite))
}

func (s *UnexportedMethodsTestSuite) TestRunMigrations() {
	tdc := TestDatabaseContainer{
		Container:        s.postgresCtr,
		ConnectionString: s.connString,
	}

	err := tdc.getSetupDirs()
	assert.Nil(s.T(), err)
	err = tdc.runMigrations()
	assert.Nil(s.T(), err)

}

func (s *UnexportedMethodsTestSuite) TestInitSeed() {
	tdc := TestDatabaseContainer{
		Container:        s.postgresCtr,
		ConnectionString: s.connString,
	}

	err := tdc.getSetupDirs()
	assert.Nil(s.T(), err)
	err = tdc.runMigrations()
	assert.Nil(s.T(), err)
	err = tdc.initSeed()
	assert.Nil(s.T(), err)
}

func TestNewTestDatabaseContainer(t *testing.T) {
	tdc, err := NewTestDatabaseContainer()
	assert.NoError(t, err)
	assert.NotNil(t, tdc.migrations)
	assert.NotNil(t, tdc.testdata)
	assert.NotNil(t, tdc.ConnectionString)
	assert.NotNil(t, tdc.Container)
}
