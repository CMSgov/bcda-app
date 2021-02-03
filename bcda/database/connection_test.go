package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/conf"
	"github.com/jackc/pgx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ConnectionTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (suite *ConnectionTestSuite) TestDbConnections() {

	// after this test, replace the original log.Fatal() function
	origLogFatal := LogFatal
	defer func() { LogFatal = origLogFatal }()

	// create the mock version of log.Fatal()
	LogFatal = func(args ...interface{}) {
		fmt.Println("FATAL (NO-OP)")
	}

	// get the real database URL
	actualDatabaseURL := conf.GetEnv("DATABASE_URL")

	// set the database URL to a bogus value to test negative scenarios
	conf.SetEnv(suite.T(), "DATABASE_URL", "fake_db_url")

	// attempt to open DB connection swith the bogus DB string
	suite.db = GetDbConnection()

	// asert that Ping returns an error
	assert.NotNil(suite.T(), suite.db.Ping(), "Database should fail to connect (negative scenario)")

	// close DBs to reset the test
	suite.db.Close()

	// set the database URL back to the real value to test the positive scenarios
	conf.SetEnv(suite.T(), "DATABASE_URL", actualDatabaseURL)

	suite.db = GetDbConnection()
	defer suite.db.Close()

	// assert that Ping() does not return an error
	assert.Nil(suite.T(), suite.db.Ping(), "Error connecting to sql database")
}

// TestHealthCheck verifies that we are able to start the health check
// and the checks do not cause a panic by waiting some amount of time
// to ensure that health checks are being executed.
func (suite *ConnectionTestSuite) TestHealthCheck() {
	cfg, err := pgx.ParseURI(os.Getenv("DATABASE_URL"))
	assert.NoError(suite.T(), err)

	pool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig: cfg,
	})
	assert.NoError(suite.T(), err)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	StartHealthCheck(ctx, pool, time.Millisecond)
	time.Sleep(10 * time.Millisecond)
}

func TestConnectionTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionTestSuite))
}
