package migrations

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/huandu/go-sqlbuilder"
	"github.com/pborman/uuid"

	"github.com/CMSgov/bcda-app/conf"

	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/stretchr/testify/suite"

	_ "github.com/jackc/pgx"
)

const (
	sqlFlavor = sqlbuilder.PostgreSQL
	nullValue = "NULL"
)

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

	databaseURL := conf.GetEnv("DATABASE_URL")
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

	migration7Tables := []string{"acos", "cclf_beneficiaries", "cclf_files",
		"job_keys", "jobs", "suppressions", "suppression_files"}

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
				assertColumnExists(t, true, db, "cclf_files", "type")
				assertColumnDefaultValue(t, db, "type", "0", []interface{}{"cclf_files"})
			},
		},
		{
			"Add blacklisted column to acos",
			func(t *testing.T) {
				migrator.runMigration(t, "3")
				assertColumnExists(t, true, db, "acos", "blacklisted")
				assertColumnDefaultValue(t, db, "blacklisted", "false", []interface{}{"acos"})
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
			"Add default now() timestamp to updated_at columns",
			func(t *testing.T) {
				migrator.runMigration(t, "6")
				assertColumnDefaultValue(t, db, "updated_at", "now()", migration5Tables)

				type tables struct {
					tableName string
					fields    map[string]interface{}
				}

				// Setup tables to test with fields that cant be Null
				testTables := []tables{
					{
						"acos",
						map[string]interface{}{
							"uuid": uuid.NewUUID(),
							"name": uuid.New(),
						},
					},
					{
						"cclf_files",
						map[string]interface{}{
							"cclf_num":         rand.Intn(10),
							"name":             uuid.New(),
							"timestamp":        time.Now(),
							"performance_year": rand.Intn(4),
						},
					},
					// cclf_beneficiaries must be written AFTER cclf_files to ensure
					// we satisfy the foreign key constraint
					{
						"cclf_beneficiaries",
						map[string]interface{}{
							"file_id": 1, // Set this to 1 to ensure that we reference the cclf_file created above
							"mbi":     "test1234",
						},
					},
					{
						"job_keys",
						map[string]interface{}{
							"file_name": uuid.New(), // This is not required but we need a unique identifier to identify the row
						},
					},
					{
						"suppressions",
						map[string]interface{}{
							"file_id": rand.Intn(25),
						},
					},
					{
						"suppression_files",
						map[string]interface{}{
							"name":      uuid.New(),
							"timestamp": time.Now(),
						},
					},
				}

				var createdAt, updatedAt time.Time
				for _, v := range testTables {
					createdAt, updatedAt = createTestRow(t, db, v.tableName, v.fields)
					assert.Equal(t, createdAt, updatedAt) // Created and Updated at will be same value on create
					createdAt, updatedAt = updateTestRow(t, db, v.tableName, v.fields)
					assert.True(t, updatedAt.After(createdAt)) // Updated at will be more recent than Created at after update
				}
			},
		},
		{
			"Remove deleted_at columns from database tables",
			func(t *testing.T) {
				for _, table := range migration7Tables {
					assertColumnExists(t, true, db, table, "deleted_at")
				}
				migrator.runMigration(t, "7")
				for _, table := range migration7Tables {
					assertColumnExists(t, false, db, table, "deleted_at")
				}
			},
		},
		{
			"Add termination_details column to acos",
			func(t *testing.T) {
				migrator.runMigration(t, "8")
				assertColumnExists(t, true, db, "acos", "termination_details")
				assertColumnDefaultValue(t, db, "termination_details", nullValue, []interface{}{"acos"})
			},
		},
		{
			"Remove termination_details column to acos",
			func(t *testing.T) {
				migrator.runMigration(t, "7")
				assertColumnExists(t, false, db, "acos", "termination_details")
			},
		},
		{
			"Add deleted_at columns to database tables",
			func(t *testing.T) {
				for _, table := range migration7Tables {
					assertColumnExists(t, false, db, table, "deleted_at")
				}
				migrator.runMigration(t, "6")
				for _, table := range migration7Tables {
					assertColumnExists(t, true, db, table, "deleted_at")
				}
			},
		},
		{
			"Remove default now() timestamp to updated_at columns",
			func(t *testing.T) {
				migrator.runMigration(t, "5")
				assertColumnDefaultValue(t, db, "updated_at", "", migration5Tables)
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
		// Ensure each test runs with a clean state
		cleanData(s.T(), db)
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
		// Ensure each test runs with a clean state
		cleanData(s.T(), db)
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
		// A default value of NULL is written as a NULL string field
		// So we need to check NullString to ensure it was set to a null value
		if expectedDefault == nullValue {
			assert.False(t, actualDefault.Valid, "%s default value is supposed to be null. Actual value is %s",
				tableName, actualDefault.String)
			continue
		}
		assert.Equal(t, expectedDefault, actualDefault.String, "%s default value is %s; actual value should be %s", tableName, actualDefault.String, expectedDefault)
	}
}

func createTestRow(t *testing.T, db *sql.DB, tableName string, fields map[string]interface{}) (createdAt, updatedAt time.Time) {
	ib := sqlFlavor.NewInsertBuilder()
	ib.InsertInto(tableName)

	columns := []string{}
	values := []interface{}{}
	for k, v := range fields {
		columns = append(columns, k)
		values = append(values, v)
	}

	ib.Cols(columns...)
	ib.Values(values...)

	query, args := ib.Build()
	query = fmt.Sprintf("%s RETURNING created_at, updated_at", query) // Force created_at and updated_at to be returned
	err := db.QueryRow(query, args...).Scan(&createdAt, &updatedAt)
	assert.NoError(t, err)

	return
}

func updateTestRow(t *testing.T, db *sql.DB, tableName string, fields map[string]interface{}) (createdAt, updatedAt time.Time) {
	ub := sqlFlavor.NewUpdateBuilder()
	ub.Update(tableName)

	for k, v := range fields {
		ub.SetMore(
			ub.Assign(k, v),
		)
		ub.Where(
			ub.Equal(k, v),
		)
	}

	query, args := ub.Build()
	query = fmt.Sprintf("%s RETURNING created_at, updated_at", query) // Force created_at and updated_at to be returned
	err := db.QueryRow(query, args...).Scan(&createdAt, &updatedAt)
	assert.NoError(t, err)

	return
}

// cleanData removes all the non migration related tables
func cleanData(t *testing.T, db *sql.DB) {
	sb := sqlFlavor.NewSelectBuilder()
	sb.Select("table_name").From("information_schema.tables").Where(
		sb.NotLike("table_name", "%schema_migrations%"), // Ensure we skip the tables related to migration
		sb.Equal("table_schema", "public"))
	query, args := sb.Build()
	rows, err := db.Query(query, args...)
	assert.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var tableName string
		assert.NoError(t, rows.Scan(&tableName))
		// Use TRUNCATE ... CASCADE to ensure that all foreign data is removed.
		_, err := db.Exec(fmt.Sprintf("TRUNCATE %s CASCADE", tableName))
		assert.NoError(t, err)
	}
}
