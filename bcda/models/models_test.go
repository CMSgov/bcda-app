package models

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testConstants"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
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

func (s *ModelsTestSuite) TestJobCompleted() {

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   1,
	}
	s.db.Save(&j)
	completed, err := j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
	assert.Nil(s.T(), err)
	completed, err = j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)
}
func (s *ModelsTestSuite) TestJobDefaultCompleted() {

	// Job is completed, but no keys exist.  This is fine, it is still complete
	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Completed",
		JobCount:   10,
	}
	s.db.Save(&j)

	completed, err := j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}
func (s *ModelsTestSuite) TestJobwithKeysCompleted() {

	j := Job{
		ACOID:      uuid.Parse("DBBD1CE1-AE24-435C-807D-ED45953077D3"),
		UserID:     uuid.Parse("82503A18-BF3B-436D-BA7B-BAE09B7FFD2F"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
		JobCount:   10,
	}
	s.db.Save(&j)
	completed, err := j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	// JobKeys exist, but not enough to make the job complete
	completed, err = j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.False(s.T(), completed)

	for i := 1; i <= 5; i++ {
		err = s.db.Create(&JobKey{JobID: j.ID, EncryptedKey: []byte("NOT A KEY"), FileName: "SOMETHING.ndjson"}).Error
		assert.Nil(s.T(), err)
	}
	completed, err = j.CheckCompletedAndCleanup()
	assert.Nil(s.T(), err)
	assert.True(s.T(), completed)
	s.db.Delete(&j)

}

func (s *ModelsTestSuite) TestGetEnqueJobs() {
	assert := s.Assert()

	j := Job{
		ACOID:      uuid.Parse(testConstants.DEVACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/Patient/$export",
		Status:     "Pending",
	}
	s.db.Save(&j)
	defer s.db.Delete(&j)

	enqueueJobs, err := j.GetEnqueJobs(true, "Patient")

	assert.Nil(err)
	assert.NotNil(enqueueJobs)
	assert.Equal(1, len(enqueueJobs))
	for _, queJob := range enqueueJobs {

		jobArgs := jobEnqueueArgs{}
		err := json.Unmarshal(queJob.Args, &jobArgs)
		if err != nil {
			s.T().Error(err)
		}
		assert.Equal(int(j.ID), jobArgs.ID)
		assert.Equal(testConstants.DEVACOUUID, jobArgs.ACOID)
		assert.Equal("6baf8254-2e8a-4808-b11d-0fa00c527d2e", jobArgs.UserID)
		assert.Equal("Patient", jobArgs.ResourceType)
		assert.Equal(true, jobArgs.Encrypt)
		assert.Equal(50, len(jobArgs.BeneficiaryIDs))
	}

	j = Job{
		ACOID:      uuid.Parse(testConstants.DEVACOUUID),
		UserID:     uuid.Parse("6baf8254-2e8a-4808-b11d-0fa00c527d2e"),
		RequestURL: "/api/v1/ExplanationOfBenefit/$export",
		Status:     "Pending",
	}

	s.db.Save(&j)
	defer s.db.Delete(&j)
	os.Setenv("BCDA_FHIR_MAX_RECORDS", "15")

	enqueueJobs, err = j.GetEnqueJobs(true, "ExplanationOfBenefit")
	assert.Nil(err)
	assert.NotNil(enqueueJobs)
	assert.Equal(4, len(enqueueJobs))
	enqueuedBenes := 0
	for _, queJob := range enqueueJobs {

		jobArgs := jobEnqueueArgs{}
		err := json.Unmarshal(queJob.Args, &jobArgs)
		if err != nil {
			s.T().Error(err)
		}
		enqueuedBenes += len(jobArgs.BeneficiaryIDs)
		assert.True(len(jobArgs.BeneficiaryIDs) <= 15)
	}
	assert.Equal(50, enqueuedBenes)

}

func (s *ModelsTestSuite) TestGetBeneficiaryIDs() {
	assert := s.Assert()
	var aco, smallACO, mediumACO, largeACO ACO
	acoUUID := uuid.Parse(testConstants.DEVACOUUID)

	err := s.db.Find(&aco, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err := aco.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(50, len(beneficiaryIDs))

	// small ACO has 10 benes
	acoUUID = uuid.Parse(testConstants.SMALLACOUUID)
	err = s.db.Debug().Find(&smallACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err = smallACO.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(10, len(beneficiaryIDs))

	// Medium ACO has 25 benes
	acoUUID = uuid.Parse(testConstants.MEDIUMACOUUID)
	err = s.db.Find(&mediumACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err = mediumACO.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(25, len(beneficiaryIDs))

	// Large ACO has 100 benes
	acoUUID = uuid.Parse(testConstants.LARGEACOUUID)
	err = s.db.Find(&largeACO, "UUID = ?", acoUUID).Error
	assert.Nil(err)
	beneficiaryIDs, err = largeACO.GetBeneficiaryIDs()
	assert.Nil(err)
	assert.NotNil(beneficiaryIDs)
	assert.Equal(100, len(beneficiaryIDs))

}

func (s *ModelsTestSuite) TestGroupStructs() {
	groupID := "A12345"
	name := "ACO Corp Systems"
	users := []string{"00uiqolo7fEFSfif70h7", "l0vckYyfyow4TZ0zOKek", "HqtEi2khroEZkH4sdIzj"}
	scopes := []string{"user-admin", "system-admin"}
	resources := []Resource{
		Resource{ID: "xxx", Name: "BCDA API", Scopes: []string{"bcda-api"}},
		Resource{ID: "eft", Name: "EFT CCLF", Scopes: []string{"eft-app:download", "eft-data:read"}},
	}
	systems := []System{
		System{ClientID: "4tuhiOIFIwriIOH3zn", SoftwareID: "4NRB1-0XZABZI9E6-5SM3R", ClientName: "ACO System A", ClientURI: "https://www.acocorpsite.com"},
	}
	groupData := GroupData{Name: name, Users: users, Scopes: scopes, Resources: resources, Systems: systems}

	rawGroupData, err := json.Marshal(groupData)
	assert.Nil(s.T(), err)

	group := Group{GroupID: groupID, Data: postgres.Jsonb{RawMessage: rawGroupData}}
	db := database.GetGORMDbConnection()
	defer database.Close(db)
	err = db.Save(&group).Error
	assert.Nil(s.T(), err)

	jsonData := group.Data
	byteData, err := jsonData.MarshalJSON()
	assert.Nil(s.T(), err)
	groupData2 := GroupData{}
	err = json.Unmarshal(byteData, &groupData2)
	assert.Nil(s.T(), err)
	assert.Equal(s.T(), groupData2, groupData)
}

// Sample values from https://confluence.cms.gov/pages/viewpage.action?spaceKey=BB&title=Getting+Started+with+Blue+Button+2.0%27s+Backend#space-menu-link-content
func (s *ModelsTestSuite) TestHashHICN() {
	cclfBeneficiary := CCLFBeneficiary{HICN: "1000067585", MBI: "NOTHING"}
	HICNHash := cclfBeneficiary.GetHashedHICN()
	assert.Equal(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
	cclfBeneficiary.HICN = "123456789"
	HICNHash = cclfBeneficiary.GetHashedHICN()
	assert.NotEqual(s.T(), "b67baee938a551f06605ecc521cc329530df4e088e5a2d84bbdcc047d70faff4", HICNHash)
}
