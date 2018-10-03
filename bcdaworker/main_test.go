package main

import (
	"encoding/json"
	"fmt"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/bgentry/que-go"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"os"
	"testing"
)

type MainTestSuite struct {
	testUtils.AuthTestSuite
}

func (s *MainTestSuite) SetupTest() {

}

func (s *MainTestSuite) TearDownTest() {
	testUtils.PrintSeparator()
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func (s *MainTestSuite) TestProcessJob() {
	db := database.GetGORMDbConnection()
	defer db.Close()

	j := models.Job{
		AcoID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
	}
	db.Save(&j)

	jobArgs := new(jobEnqueueArgs)
	jobArgs.ID = int(j.ID)
	jobArgs.AcoID = j.AcoID.String()
	jobArgs.UserID = j.UserID.String()
	jobArgs.BeneficiaryIDs = []string{"foo", "bar", "baz"}
	args, _ := json.Marshal(jobArgs)

	job := &que.Job{
		Type: "ProcessJob",
		Args: args,
	}
	fmt.Println("About to queue up the job")
	err := processJob(job)
	assert.Nil(s.T(), err)
	var completedJob models.Job
	err = db.First(&completedJob, "ID = ?", jobArgs.ID).Error
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), completedJob.Status, "Completed")
}

func (s *MainTestSuite) TestSetupQueue() {
	setupQueue()
	os.Setenv("WORKER_POOL_SIZE", "7")
	setupQueue()
}
