package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"

	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/stretchr/testify/suite"

	_ "github.com/jackc/pgx/stdlib"
)

const sqlFlavor = sqlbuilder.PostgreSQL

// These tests relies on migrate tool being installed
// See: https://github.com/golang-migrate/migrate/tree/v4.13.0/cmd/migrate
type MigrationTestSuite struct {
	suite.Suite

	db *sql.DB

	bcdaDB    string
	bcdaDBURL string

	bcdaQueueDB    string
	bcdaQueueDBURL string
}

func (s *MigrationTestSuite) SetupSuite() {
	// We expect that the DB URL follows
	// postgres://<USER_NAME>:<PASSWORD>@<HOST>:<PORT>/<DB_NAME>
	re := regexp.MustCompile(`(postgresql\:\/\/\S+\:\S+\@\S+\:\d+\/)(.*)(\?.*)`)

	s.db = database.GetDbConnection()

	databaseURL := os.Getenv("DATABASE_URL")
	s.bcdaDB = fmt.Sprintf("migrate_test_bcda_%d", time.Now().Nanosecond())
	s.bcdaQueueDB = fmt.Sprintf("migrate_test_bcda_queue_%d", time.Now().Nanosecond())
	s.bcdaDBURL = re.ReplaceAllString(databaseURL, fmt.Sprintf("${1}%s${3}", s.bcdaDB))
	s.bcdaQueueDBURL = re.ReplaceAllString(databaseURL, fmt.Sprintf("${1}%s${3}", s.bcdaQueueDB))

	if _, err := s.db.Exec("CREATE DATABASE " + s.bcdaDB); err != nil {
		assert.FailNowf(s.T(), "Could not create bcda db", err.Error())
	}

	if _, err := s.db.Exec("CREATE DATABASE " + s.bcdaQueueDB); err != nil {
		assert.FailNowf(s.T(), "Could not create bcda_queue db", err.Error())
	}
}

func (s *MigrationTestSuite) TearDownSuite() {
	if _, err := s.db.Exec("DROP DATABASE " + s.bcdaDB); err != nil {
		assert.FailNowf(s.T(), "Could not drop bcda db", err.Error())
	}

	if _, err := s.db.Exec("DROP DATABASE " + s.bcdaQueueDB); err != nil {
		assert.FailNowf(s.T(), "Could not drop bcda_queue db", err.Error())
	}
}

func TestMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}

var migration5Tables = []interface{}{
	"acos",
	"cclf_beneficiaries",
	"cclf_files",
	"job_keys",
	"suppressions",
	"suppression_files",
}

func (s *MigrationTestSuite) TestBCDAMigration() {
	migrator := migrator{
		migrationPath: "./bcda/",
		dbURL:         s.bcdaDBURL,
	}
	db, err := sql.Open("pgx", s.bcdaDBURL)
	if err != nil {
		assert.FailNowf(s.T(), "Failed to open postgres connection", err.Error())
	}
	defer db.Close()

	migration1Tables := []string{"acos", "cclf_beneficiaries", "cclf_beneficiary_xrefs",
		"cclf_files", "job_keys", "jobs", "suppression_files", "suppressions"}

	// Tests should begin with "up" migrations, in order, followed by "down" migrations in reverse order
	tests := []struct {
		name  string
		tFunc func(t *testing.T)
	}{
		{
			"Apply initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "1")
				for _, table := range migration1Tables {
					assertTableExists(t, true, db, table)
				}
			},
		},
		{
			"Add type column to cclf_files",
			func(t *testing.T) {
				migrator.runMigration(t, "2")
				noType := &models.CCLFFile{
					CCLFNum:         8,
					Name:            "CCLFFile_no_type",
					ACOCMSID:        "T9999",
					Timestamp:       time.Now(),
					PerformanceYear: 20,
				}
				withType := &models.CCLFFile{
					CCLFNum:         8,
					Name:            "CCLFFile_with_type",
					ACOCMSID:        "T9999",
					Timestamp:       time.Now(),
					PerformanceYear: 20,
					Type:            models.FileTypeRunout,
				}
				postgrestest.CreateCCLFFile(t, db, noType)
				postgrestest.CreateCCLFFile(t, db, withType)

				assert.Equal(t, noType.Type, postgrestest.GetCCLFFilesByName(t, db, noType.Name)[0].Type)
				assert.Equal(t, withType.Type, postgrestest.GetCCLFFilesByName(t, db, withType.Name)[0].Type)
			},
		},
		{
			"Add blacklisted column to acos",
			func(t *testing.T) {
				// Verify that existing ACOs have blacklisted equal to false
				beforeMigration := models.ACO{
					UUID:        uuid.NewUUID(),
					Name:        uuid.New(),
					Blacklisted: true,
				}
				// Need to manually build the insert query since the column does not exist yet.
				ib := sqlFlavor.NewInsertBuilder().InsertInto("acos").
					Cols("uuid", "name").Values(beforeMigration.UUID, beforeMigration.Name)
				query, args := ib.Build()
				_, err := db.Exec(query, args...)
				assert.NoError(t, err)

				migrator.runMigration(t, "3")
				blacklisted := models.ACO{
					UUID:        uuid.NewUUID(),
					Name:        uuid.New(),
					Blacklisted: true,
				}
				notBlacklisted := models.ACO{
					UUID:        uuid.NewUUID(),
					Name:        uuid.New(),
					Blacklisted: false,
				}
				notSet := models.ACO{
					UUID: uuid.NewUUID(),
					Name: uuid.New(),
				}
				postgrestest.CreateACO(t, db, blacklisted)
				postgrestest.CreateACO(t, db, notBlacklisted)
				postgrestest.CreateACO(t, db, notSet)

				assert.True(t, postgrestest.GetACOByUUID(t, db, blacklisted.UUID).Blacklisted)
				assert.False(t, postgrestest.GetACOByUUID(t, db, notBlacklisted.UUID).Blacklisted)
				assert.False(t, postgrestest.GetACOByUUID(t, db, notSet.UUID).Blacklisted)
				assert.False(t, postgrestest.GetACOByUUID(t, db, beforeMigration.UUID).Blacklisted)
			},
		},
		{
			"Remove HICN column from cclf_beneficiaries and suppressions",
			func(t *testing.T) {
				assertColumnExists(t, true, db, "cclf_beneficiaries", "hicn")
				migrator.runMigration(t, "4")
				assertColumnExists(t, false, db, "cclf_beneficiaries", "hicn")
			},
		},
		{
			"Add default now() timestamp to created_at columns",
			func(t *testing.T) {
				migrator.runMigration(t, "5")
				assertColumnDefaultValue(t, db, "created_at", "now()", migration5Tables)
			},
		},
		{
			"Remove default now() timestamp to created_at columns",
			func(t *testing.T) {
				assertColumnDefaultValue(t, db, "created_at", "now()", migration5Tables)
				migrator.runMigration(t, "4")
				assertColumnDefaultValue(t, db, "created_at", "", migration5Tables)
			},
		},
		{
			"Add HICN column to cclf_beneficiaries and suppressions",
			func(t *testing.T) {
				assertColumnExists(t, false, db, "cclf_beneficiaries", "hicn")
				migrator.runMigration(t, "3")
				assertColumnExists(t, true, db, "cclf_beneficiaries", "hicn")
			},
		},
		{
			"Remove blacklisted column from acos",
			func(t *testing.T) {
				migrator.runMigration(t, "2")
				assertColumnExists(t, false, db, "acos", "blacklisted")
			},
		},
		{
			"Remove type column from cclf_files",
			func(t *testing.T) {
				migrator.runMigration(t, "1")
				assertColumnExists(t, false, db, "cclf_files", "type")
			},
		},
		{
			"Revert initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "0")
				for _, table := range migration1Tables {
					assertTableExists(t, false, db, table)
				}
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, tt.tFunc)
	}
}

func (s *MigrationTestSuite) TestBCDAQueueMigration() {
	migrator := migrator{
		migrationPath: "./bcda_queue/",
		dbURL:         s.bcdaQueueDBURL,
	}
	db, err := sql.Open("pgx", s.bcdaQueueDBURL)
	if err != nil {
		assert.FailNowf(s.T(), "Failed to open postgres connection", err.Error())
	}
	defer db.Close()

	migration1Tables := []string{"que_jobs"}

	// Tests should begin with "up" migrations, in order, followed by "down" migrations in reverse order
	tests := []struct {
		name  string
		tFunc func(t *testing.T)
	}{
		{
			"Apply initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "1")
				for _, table := range migration1Tables {
					assertTableExists(t, true, db, table)
				}
			},
		},
		{
			"Revert initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "0")
				for _, table := range migration1Tables {
					assertTableExists(t, false, db, table)
				}
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, tt.tFunc)
	}
}

type migrator struct {
	migrationPath string
	dbURL         string
}

func (m migrator) runMigration(t *testing.T, idx string) {
	args := []string{"goto", idx}
	expVersion := idx
	// Since we do not have a 0 index, this is interpreted
	// as revert the last migration (1)
	if idx == "0" {
		args = []string{"down", "1"}
	}

	args = append([]string{"-database", m.dbURL, "-path",
		m.migrationPath}, args...)

	_, err := exec.Command("migrate", args...).CombinedOutput()
	if err != nil {
		t.Errorf("Failed to run migration %s", err.Error())
	}

	// If we're going down past the first schema, we won't be able
	// to check the version since there's no active schema version
	if idx == "0" {
		return
	}

	// Expected output:
	// <VERSION>
	// If there's a failure (i.e. dirty migration)
	// <VERSION> (dirty)
	out, err := exec.Command("migrate", "-database", m.dbURL, "-path",
		m.migrationPath, "version").CombinedOutput()
	if err != nil {
		t.Errorf("Failed to retrieve version information %s", err.Error())
	}
	str := strings.TrimSpace(string(out))

	assert.Contains(t, expVersion, str)
	assert.NotContains(t, str, "dirty")
}

func assertColumnExists(t *testing.T, shouldExist bool, db *sql.DB, tableName, columnName string) {
	sb := sqlFlavor.NewSelectBuilder().Select("COUNT(1)").From("information_schema.columns ")
	sb.Where(sb.Equal("table_name", tableName), sb.Equal("column_name", columnName))
	query, args := sb.Build()
	var count int
	assert.NoError(t, db.QueryRow(query, args...).Scan(&count))

	var expected int
	if shouldExist {
		expected = 1
	}
	assert.Equal(t, expected, count)
}

func assertTableExists(t *testing.T, shouldExist bool, db *sql.DB, tableName string) {
	sb := sqlFlavor.NewSelectBuilder().Select("COUNT(1)").From("information_schema.tables ")
	sb.Where(sb.Equal("table_name", tableName))
	query, args := sb.Build()
	var count int
	assert.NoError(t, db.QueryRow(query, args...).Scan(&count))

	var expected int
	if shouldExist {
		expected = 1
	}
	assert.Equal(t, expected, count)
}

func assertColumnDefaultValue(t *testing.T, db *sql.DB, columnName, expectedDefault string, tables []interface{}) {
	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("table_name", "column_default").
		From("information_schema.columns").
		Where(
			sb.NotIn("table_schema", "information_schema", "pg_catalog"), // Ignore postgres internal schemas
			sb.Equal("column_name", columnName),                          // Filter desired column
			sb.In("table_name", tables...),                               // Only check specific tables
		)

	query, args := sb.Build()
	rows, err := db.Query(query, args...)
	assert.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var tableName string
		var actualDefault sql.NullString
		assert.NoError(t, rows.Scan(&tableName, &actualDefault))
		assert.Equal(t, expectedDefault, actualDefault.String, "%s default value is %s; actual value should be %s", tableName, actualDefault.String, expectedDefault)
	}
}
