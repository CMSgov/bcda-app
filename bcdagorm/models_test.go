package bcdagorm_test

import (
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcdagorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ModelsTestSuite struct {
	suite.Suite
}

func (s *ModelsTestSuite) SetupTest() {
	// Setup the DB
	bcdagorm.Initialize()

}

func (s *ModelsTestSuite) TestAco() {
	db := database.GetGORMDbConnection()
	db.Create(&bcdagorm.ACO{UUID: uuid.NewRandom(), Name: "Test ACO"})
	var testACO bcdagorm.ACO
	db.First(&testACO, "Name=?", "Test ACO")
	assert.True(s.T(), testACO.Name == "Test ACO")

}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
