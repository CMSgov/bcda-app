package tests

import (
	"database/sql"
	"fmt"
	"github.com/CMSgov/bcda-app/ssas"
	"github.com/CMSgov/bcda-app/ssas/service"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
	"github.com/dhui/dktest"
	_ "github.com/lib/pq"
)

func pgReady(ctx context.Context, c dktest.ContainerInfo) bool {
	ip, port, err := c.FirstPort()
	if err != nil {
		return false
	}
	connStr := fmt.Sprintf("host=%s port=%s user=postgres dbname=postgres sslmode=disable", ip, port)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return false
	}
	defer db.Close()
	return db.PingContext(ctx) == nil
}

func Test(t *testing.T) {
	dktest.Run(t, "postgres:alpine", dktest.Options{PortRequired: true, ReadyFunc: pgReady},
		func(t *testing.T, c dktest.ContainerInfo) {
			ip, port, err := c.FirstPort()
			if err != nil {
				t.Fatal(err)
			}
			connStr := fmt.Sprintf("host=%s port=%s user=postgres dbname=postgres sslmode=disable", ip, port)
			db, err := sql.Open("postgres", connStr)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := db.Ping(); err != nil {
				t.Fatal(err)
			}
			// Test using db
		})
}

type MigrationTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (s *MigrationTestSuite) SetupSuite() {
	s.db = ssas.GetDbConnection()
	service.StartBlacklist()
}

func (s *MigrationTestSuite) TearDownSuite() {
	assert.Nil(s.T(), s.db.Close())
}

func (s *MigrationTestSuite) TestCreateGroup() {
	driver, err := postgres.WithInstance(s.db, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		"file:///migrations",
		"postgres", driver)
	m.Steps(2)
}

func TestMigrationTestSuite(t *testing.T) {
	suite.Run(t, new(MigrationTestSuite))
}