package database_test

import (
	"database/sql"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ConnectionTestSuite struct {
	suite.Suite
	db *sql.DB
	gormdb *gorm.DB
}

func (suite *ConnectionTestSuite) TestDbConnections() {
	suite.db = database.GetDbConnection()
	defer suite.db.Close()

	suite.gormdb = database.GetGORMDbConnection()
	defer suite.gormdb.Close()

	assert.NotNil(suite.T(), suite.db, fmt.Sprint("Error connecting to sql database"))
	assert.NotNil(suite.T(), suite.gormdb, fmt.Sprint("Error connecting to gorm database "))
}

func TestConnectionTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionTestSuite))
}