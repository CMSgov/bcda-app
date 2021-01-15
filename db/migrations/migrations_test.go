package migrations

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/stretchr/testify/assert"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/stretchr/testify/suite"
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

func (s *MigrationTestSuite) TestBCDAMigration() {
	migrator := migrator{
		migrationPath: "./bcda/",
		dbURL:         s.bcdaDBURL,
	}
	db, err := gorm.Open(postgres.Open(s.bcdaDBURL), &gorm.Config{})
	if err != nil {
		assert.FailNowf(s.T(), "Failed to open postgres connection", err.Error())
	}
	defer close(db)

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
					assert.True(t, db.Migrator().HasTable(table), fmt.Sprintf("Table %s should exist", table))
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

				assert.NoError(t, db.Create(noType).Error)
				assert.NoError(t, db.Create(withType).Error)

				var result models.CCLFFile
				assert.NoError(t, db.First(&result, noType.ID).Error)
				assert.Equal(t, models.FileTypeDefault, result.Type)

				result = models.CCLFFile{}
				assert.NoError(t, db.First(&result, withType.ID).Error)
				assert.Equal(t, withType.Type, result.Type)
			},
		},
		{
			"Add blacklisted column to acos",
			func(t *testing.T) {
				// Verify that existing ACOs have blacklisted equal to false
				beforeMigration := &models.ACO{
					UUID:        uuid.NewUUID(),
					Name:        uuid.New(),
					Blacklisted: true,
				}
				assert.NoError(t, db.Select("uuid", "name").Create(beforeMigration).Error)

				migrator.runMigration(t, "3")
				blacklisted := &models.ACO{
					UUID:        uuid.NewUUID(),
					Name:        uuid.New(),
					Blacklisted: true,
				}
				notBlacklisted := &models.ACO{
					UUID:        uuid.NewUUID(),
					Name:        uuid.New(),
					Blacklisted: false,
				}
				notSet := &models.ACO{
					UUID: uuid.NewUUID(),
					Name: uuid.New(),
				}

				assert.NoError(t, db.Create(blacklisted).Error)
				assert.NoError(t, db.Create(notBlacklisted).Error)
				assert.NoError(t, db.Create(notSet).Error)

				var result models.ACO
				assert.NoError(t, db.First(&result, blacklisted.ID).Error)
				assert.True(t, result.Blacklisted)

				result = models.ACO{}
				assert.NoError(t, db.First(&result, notBlacklisted.ID).Error)
				assert.False(t, result.Blacklisted)

				result = models.ACO{}
				assert.NoError(t, db.First(&result, notSet.ID).Error)
				assert.False(t, result.Blacklisted)

				result = models.ACO{}
				assert.NoError(t, db.First(&result, beforeMigration.ID).Error)
				assert.False(t, result.Blacklisted)
			},
		},
		{
			"Remove HICN column from cclf_beneficiaries and suppressions",
			func(t *testing.T) {
				assert.True(t, db.Migrator().HasColumn(&models.CCLFBeneficiary{}, "hicn"))
				migrator.runMigration(t, "4")
				assert.False(t, db.Migrator().HasColumn(&models.CCLFBeneficiary{}, "hicn"))
			},
		},
		{
			"Add HICN column to cclf_beneficiaries and suppressions",
			func(t *testing.T) {
				assert.False(t, db.Migrator().HasColumn(&models.CCLFBeneficiary{}, "hicn"))
				migrator.runMigration(t, "3")
				assert.True(t, db.Migrator().HasColumn(&models.CCLFBeneficiary{}, "hicn"))
			},
		},
		{
			"Remove blacklisted column from acos",
			func(t *testing.T) {
				migrator.runMigration(t, "2")

				blacklisted := &models.ACO{
					UUID:        uuid.NewUUID(),
					Name:        uuid.New(),
					Blacklisted: true,
				}

				err := db.Create(blacklisted).Error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "column \"blacklisted\" of relation \"acos\" does not exist")
			},
		},
		{
			"Remove type column from cclf_files",
			func(t *testing.T) {
				migrator.runMigration(t, "1")

				withType := &models.CCLFFile{
					CCLFNum:         8,
					Name:            "CCLFFile_with_type_no_column",
					ACOCMSID:        "T9999",
					Timestamp:       time.Now(),
					PerformanceYear: 20,
					Type:            models.FileTypeRunout,
				}

				err := db.Create(withType).Error
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "column \"type\" of relation \"cclf_files\" does not exist")
			},
		},
		{
			"Revert initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "0")
				for _, table := range migration1Tables {
					assert.False(t, db.Migrator().HasTable(table), fmt.Sprintf("Table %s should not exist", table))
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
	db, err := gorm.Open(postgres.Open(s.bcdaQueueDBURL), &gorm.Config{})
	if err != nil {
		assert.FailNowf(s.T(), "Failed to open postgres connection", err.Error())
	}
	defer close(db)

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
					assert.True(t, db.Migrator().HasTable(table), fmt.Sprintf("Table %s should exist", table))
				}
			},
		},
		{
			"Revert initial schema",
			func(t *testing.T) {
				migrator.runMigration(t, "0")
				for _, table := range migration1Tables {
					assert.False(t, db.Migrator().HasTable(table), fmt.Sprintf("Table %s should not exist", table))
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

func close(db *gorm.DB) {
	dbc, err := db.DB()
	if err != nil {
		log.Infof("failed to retrieve db connection: %v", err)
		return
	}
	if err := dbc.Close(); err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Infof("failed to close db connection at %s#%d because %s", file, line, err)
	}
}
