package models

import (
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type ModelsTestSuite struct {
	suite.Suite

	// Re-initialized for every test
	db *gorm.DB
}

func (s *ModelsTestSuite) SetupTest() {
	s.db = database.GetGORMDbConnection()
}

func (s *ModelsTestSuite) TearDownTest() {
	database.Close(s.db)
}

func TestModelsTestSuite(t *testing.T) {
	suite.Run(t, new(ModelsTestSuite))
}

func (s *ModelsTestSuite) TestJobStatusMessage() {
	j := Job{Status: "In Progress", JobCount: 25, CompletedJobCount: 6}
	assert.Equal(s.T(), "In Progress (24%)", j.StatusMessage())

	j = Job{Status: "In Progress", JobCount: 0, CompletedJobCount: 0}
	assert.Equal(s.T(), "In Progress", j.StatusMessage())

	j = Job{Status: JobStatusCompleted, JobCount: 25, CompletedJobCount: 25}
	assert.Equal(s.T(), string(JobStatusCompleted), j.StatusMessage())
}

func (s *ModelsTestSuite) TestGetBlueButtonID_CCLFBeneficiary() {
	assert := s.Assert()
	cclfBeneficiary := CCLFBeneficiary{MBI: "MBI"}
	bbc := testUtils.BlueButtonClient{}
	bbc.MBI = &cclfBeneficiary.MBI

	bbc.On("GetPatientByIdentifierHash", client.HashIdentifier(cclfBeneficiary.MBI)).Return(bbc.GetData("Patient", "BB_VALUE"))

	cclfBeneficiary.BlueButtonID = ""
	// New never seen before mbi, asks the mock blue button client for the value
	blueButtonID, err := cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// The object has a BB ID set on it already, but we still ask mock blue button client for the value
	// We should receive the BB_VALUE since we are ignoring cached values
	cclfBeneficiary.BlueButtonID = "LOCAL_VAL"
	blueButtonID, err = cclfBeneficiary.GetBlueButtonID(&bbc)
	assert.Nil(err)
	assert.Equal("BB_VALUE", blueButtonID)

	// Should be making two calls to BB for the MBI_MODE attemptsm, but this number will be four with the earlier test in this method.
	// This is due to the fact that we are not relying on cached identifiers
	bbc.AssertNumberOfCalls(s.T(), "GetPatientByIdentifierHash", 2)
}

func (s *ModelsTestSuite) TestDuplicateCCLFFileNames() {
	tests := []struct {
		name     string
		fileName string
		acoIDs   []string
		errMsg   string
	}{
		{"Different ACO ID", uuid.New(), []string{"ACO1", "ACO2"},
			""},
		{"Duplicate ACO ID", uuid.New(), []string{"ACO3", "ACO3"},
			`pq: duplicate key value violates unique constraint "idx_cclf_files_name_aco_cms_id_key"`},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			var err error
			var expectedFileCount int64
			for _, acoID := range tt.acoIDs {
				cclfFile := &CCLFFile{
					Name:            tt.fileName,
					ACOCMSID:        acoID,
					Timestamp:       time.Now(),
					PerformanceYear: 20,
				}
				if err1 := s.db.Create(cclfFile).Error; err1 != nil {
					err = err1
					continue
				}
				expectedFileCount++
				defer func() {
					assert.Empty(t, cclfFile.Delete())
				}()
			}

			if tt.errMsg != "" {
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}

			var count int64
			s.db.Model(&CCLFFile{}).Where("name = ?", tt.fileName).Count(&count)
			assert.True(t, expectedFileCount > 0)
			assert.Equal(t, expectedFileCount, count)
		})
	}
}

// TestCMSID verifies that we can store and retrieve the CMS_ID as expected
// i.e. the value is not padded with any extra characters
func (s *ModelsTestSuite) TestCMSID() {
	cmsID := "V001"
	cclfFile := &CCLFFile{CCLFNum: 1, Name: "someName", ACOCMSID: cmsID, Timestamp: time.Now(), PerformanceYear: 20}
	aco := &ACO{UUID: uuid.NewUUID(), CMSID: &cmsID, Name: "someName"}

	assert.NoError(s.T(), s.db.Save(cclfFile).Error)
	defer s.db.Unscoped().Delete(cclfFile)
	assert.NoError(s.T(), s.db.Save(aco).Error)
	defer s.db.Unscoped().Delete(aco)

	var actualCMSID []string
	assert.NoError(s.T(), s.db.Model(&ACO{}).Where("id = ?", aco.ID).Pluck("cms_id", &actualCMSID).Error)
	assert.Equal(s.T(), 1, len(actualCMSID))
	assert.Equal(s.T(), cmsID, actualCMSID[0])

	assert.NoError(s.T(), s.db.Model(&CCLFFile{}).Where("id = ?", cclfFile.ID).Pluck("aco_cms_id", &actualCMSID).Error)
	assert.Equal(s.T(), 1, len(actualCMSID))
	assert.Equal(s.T(), cmsID, actualCMSID[0])
}

func (s *ModelsTestSuite) TestCCLFFileType() {
	noType := &CCLFFile{
		CCLFNum:         8,
		Name:            uuid.New(),
		ACOCMSID:        "T9999",
		Timestamp:       time.Now(),
		PerformanceYear: 20,
	}
	withType := &CCLFFile{
		CCLFNum:         8,
		Name:            uuid.New(),
		ACOCMSID:        "T9999",
		Timestamp:       time.Now(),
		PerformanceYear: 20,
		Type:            FileTypeRunout,
	}

	defer func() {
		s.db.Unscoped().Delete(noType)
		s.db.Unscoped().Delete(withType)
	}()

	assert.NoError(s.T(), s.db.Create(noType).Error)
	assert.NoError(s.T(), s.db.Create(withType).Error)

	var result CCLFFile
	assert.NoError(s.T(), s.db.First(&result, noType.ID).Error)
	assert.Equal(s.T(), FileTypeDefault, result.Type)

	result = CCLFFile{}
	assert.NoError(s.T(), s.db.First(&result, withType.ID).Error)
	assert.Equal(s.T(), withType.Type, result.Type)
}
