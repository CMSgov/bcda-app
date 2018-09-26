package database

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type ConnectionTestSuite struct {
	suite.Suite
	db     *sql.DB
	gormdb *gorm.DB
}

func (suite *ConnectionTestSuite) TestDbConnections() {

	// after this test, replace the original log.Fatal() function
	origLogFatal := logFatal
	defer func() { logFatal = origLogFatal }()

	// create the mock version of log.Fatal()
	logFatal = func(args ...interface{}) {
		fmt.Println("FATAL (NO-OP)")
	}

	// get the real database URL
	actualDatabaseURL := os.Getenv("DATABASE_URL")

	// set the database URL to a bogus value to test negative scenarios
	os.Setenv("DATABASE_URL", "fake_db_url")

	// attempt to open DB connection swith the bogus DB string
	suite.db = GetDbConnection()
	suite.gormdb = GetGORMDbConnection()

	// asert that Ping returns an error
	assert.NotNil(suite.T(), suite.db.Ping(), fmt.Sprint("Database should fail to connect (negative scenario)"))
	assert.NotNil(suite.T(), suite.gormdb.DB().Ping(), fmt.Sprint("Gorm database should fail to connect (negative scenario)"))

	// close DBs to reset the test
	suite.db.Close()
	suite.gormdb.Close()

	// set the database URL back to the real value to test the positive scenarios
	os.Setenv("DATABASE_URL", actualDatabaseURL)

	suite.db = GetDbConnection()
	defer suite.db.Close()

	suite.gormdb = GetGORMDbConnection()
	defer suite.gormdb.Close()

	// assert that Ping() does not return an error
	assert.Nil(suite.T(), suite.db.Ping(), fmt.Sprint("Error connecting to sql database"))
	assert.Nil(suite.T(), suite.gormdb.DB().Ping(), fmt.Sprint("Error connecting to gorm database "))

}

func TestConnectionTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionTestSuite))
}
