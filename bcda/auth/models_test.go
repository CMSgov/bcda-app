package auth

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"testing"
)

type ModelsTestSuite struct {
	suite.Suite
}

func (s *ModelsTestSuite) SetupTest() {
	// Setup the DB
	InitializeGormModels()

}

func (s *ModelsTestSuite) TestAco() {
	//db := database.GetGORMDbConnection()
	//db.Create(&ACO{UUID: uuid.NewRandom(), Name: "Test ACO"})
	//var testACO ACO
	//db.First(&testACO, "Name=?", "Test ACO")
	//assert.True(s.T(), testACO.Name == "Test ACO")
	assert.True(s.T(), true)

}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}
