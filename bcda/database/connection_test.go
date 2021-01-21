package database

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
    
    configuration "github.com/CMSgov/bcda-app/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ConnectionTestSuite struct {
	suite.Suite
	db     *sql.DB
	gormdb *gorm.DB
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
	actualDatabaseURL := configuration.GetEnv("DATABASE_URL")

	// set the database URL to a bogus value to test negative scenarios
	os.Setenv("DATABASE_URL", "fake_db_url")

	// attempt to open DB connection swith the bogus DB string
	suite.db = GetDbConnection()
	suite.gormdb = GetGORMDbConnection()

	// asert that Ping returns an error
	assert.NotNil(suite.T(), suite.db.Ping(), "Database should fail to connect (negative scenario)")
	gdb, _ := suite.gormdb.DB()
	assert.NotNil(suite.T(), gdb.Ping(), "Gorm database should fail to connect (negative scenario)")

	// close DBs to reset the test
	suite.db.Close()
	Close(suite.gormdb)

	// set the database URL back to the real value to test the positive scenarios
	os.Setenv("DATABASE_URL", actualDatabaseURL)

	suite.db = GetDbConnection()
	defer suite.db.Close()

	suite.gormdb = GetGORMDbConnection()
	defer Close(suite.gormdb)

	// assert that Ping() does not return an error
	assert.Nil(suite.T(), suite.db.Ping(), "Error connecting to sql database")
	gdb, _ = suite.gormdb.DB()
	assert.Nil(suite.T(), gdb.Ping(), "Error connecting to gorm database")

}

func TestConnectionTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionTestSuite))
}
