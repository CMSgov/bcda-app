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
	assert := s.Assert()

	const ACOName = "ACO Name"
	cmsID := "A0000"
	acoUUID, err := CreateACO(ACOName, &cmsID)

	assert.Nil(err)
	assert.NotNil(acoUUID)

	var aco ACO
	s.db.Find(&aco, "UUID = ?", acoUUID)
	assert.NotNil(aco)
	assert.Equal(ACOName, aco.Name)
	assert.Equal("", aco.ClientID)
	assert.Equal(cmsID, *aco.CMSID)
	assert.NotNil(aco.GetPublicKey())
	assert.NotNil(GetATOPrivateKey())
	// should confirm the keys are a matched pair? i.e., encrypt something with one and decrypt with the other
	// the auth provider determines what the clientID contains (formatting, alphabet used, etc).
	// we require that it be representable in a string of less than 255 characters
	const ClientID = "Alpha client id"
	aco.ClientID = ClientID
	s.db.Save(aco)
	s.db.Find(&aco, "UUID = ?", acoUUID)
	assert.NotNil(aco)
	assert.Equal(ACOName, aco.Name)
	assert.NotNil(aco.ClientID)
	assert.Equal(ClientID, aco.ClientID)

	// make sure we can't duplicate the ACO UUID
	aco = ACO{
		UUID: acoUUID,
		Name: "Duplicate UUID Test",
	}
	err = s.db.Save(&aco).Error
	assert.NotNil(err)

	// Duplicate CMS ID
	aco = ACO{
		UUID:  uuid.NewRandom(),
		CMSID: &cmsID,
		Name:  "Duplicate CMS ID Test",
	}
	err = s.db.Save(&aco).Error
	assert.NotNil(err)
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
