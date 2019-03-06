package models

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/jinzhu/gorm"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ModelsTestSuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *ModelsTestSuite) SetupTest() {
	InitializeGormModels()
	s.db = database.GetGORMDbConnection()
}

func (s *ModelsTestSuite) TearDownTest() {
	database.Close(s.db)
}

func (s *ModelsTestSuite) TestCreateACO() {
	const ACOName = "ACO Name"
	acoUUID, err := CreateACO(ACOName)

	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), acoUUID)

	var aco ACO
	s.db.Find(&aco, "UUID = ?", acoUUID)
	assert.NotNil(s.T(), aco)
	assert.Equal(s.T(), ACOName, aco.Name)
	assert.Equal(s.T(), "", aco.ClientID)
	assert.NotNil(s.T(), aco.GetPublicKey())
	assert.NotNil(s.T(), GetATOPrivateKey())
	// should confirm the keys are a matched pair? i.e., encrypt something with one and decrypt with the other
	// the auth provider determines what the clientID contains (formatting, alphabet used, etc).
	// we require that it be representable in a string of less than 255 characters
	const ClientID = "Alpha client id"
	aco.ClientID = ClientID
	s.db.Save(aco)
	s.db.Find(&aco, "UUID = ?", acoUUID)
	assert.NotNil(s.T(), aco)
	assert.Equal(s.T(), ACOName, aco.Name)
	assert.NotNil(s.T(), aco.ClientID)
	assert.Equal(s.T(), ClientID, aco.ClientID)

	// make sure we can't duplicate the ACO UUID
	aco = ACO{
		UUID: acoUUID,
		Name: "Duplicate UUID Test",
	}
	err = s.db.Save(&aco).Error
	assert.NotNil(s.T(), err)
}

func (s *ModelsTestSuite) TestCreateUser() {
	name, email, sampleUUID, duplicateName := "First Last", "firstlast@example.com", "DBBD1CE1-AE24-435C-807D-ED45953077D3", "Duplicate Name"

	// Make a user for an ACO that doesn't exist
	badACOUser, err := CreateUser(name, email, uuid.NewRandom())
	//No ID because it wasn't saved
	assert.True(s.T(), badACOUser.ID == 0)
	// Should get an error
	assert.NotNil(s.T(), err)

	// Make a good user
	user, err := CreateUser(name, email, uuid.Parse(sampleUUID))
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), user.UUID)
	assert.NotNil(s.T(), user.ID)

	// Try making a duplicate user for the same E-mail address
	duplicateUser, err := CreateUser(duplicateName, email, uuid.Parse(sampleUUID))
	// Got a user, not the one that was requested
	assert.True(s.T(), duplicateUser.Name == name)
	assert.NotNil(s.T(), err)
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

func (s *ModelsTestSuite) TestJobCompleted() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   1,
	}
	db.Save(&j)
	completed, err := j.CheckCompleted()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	err = db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
	assert.Nil(s.T(), err)
	completed, err = j.CheckCompleted()
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)
}
func (s *ModelsTestSuite) TestJobDefaultCompleted() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	// Job is completed, but no keys exist.  This is fine, it is still complete
	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Completed",
		JobCount:   10,
	}
	db.Save(&j)

	completed, err := j.CheckCompleted()
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}
func (s *ModelsTestSuite) TestJobwithKeysCompleted() {
	db := database.GetGORMDbConnection()
	defer database.Close(db)

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   10,
	}
	db.Save(&j)
	completed, err := j.CheckCompleted()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	// JobKeys exist, but not enough to make the job complete
	completed, err = j.CheckCompleted()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	completed, err = j.CheckCompleted()
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}
