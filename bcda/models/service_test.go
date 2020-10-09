package models

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"

	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"
)

func TestSupportedACOs(t *testing.T) {
	tests := []struct {
		name        string
		cmsID       string
		isSupported bool
	}{
		{"SSP too short", "A999", false},
		{"SSP too long", "A99999", false},
		{"SSP invalid characters", "A999A", false},
		{"valid SSP", "A9999", true},

		{"NGACO too short", "V99", false},
		{"NGACO too long", "V9999", false},
		{"NGACO invalid characters", "V99V", false},
		{"valid NGACO", "V999", true},

		{"CEC too short", "E999", false},
		{"CEC too long", "E99999", false},
		{"CEC invalid characters", "E999E", false},
		{"valid CEC", "E9999", true},

		{"Unregisted ACO", "Z1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(sub *testing.T) {
			match := IsSupportedACO(tt.cmsID)
			assert.Equal(sub, tt.isSupported, match)
		})
	}
}

type ServiceTestSuite struct {
	suite.Suite
}

// Run all test suite tets
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) TestIncludeSuppressedBeneficiaries() {
	tests := []struct {
		name          string
		cclfFileNew   *CCLFFile
		cclfFileOld   *CCLFFile
		funcUnderTest func(s *service) error
	}{
		{
			"GetNewAndExistingBeneficiaries",
			getCCLFFile(1),
			getCCLFFile(2),
			func(serv *service) error {
				_, _, err := serv.getNewAndExistingBeneficiaries("cmsID", time.Now())
				return err
			},
		},
		{
			"GetBeneficiaries",
			getCCLFFile(3),
			nil,
			func(serv *service) error {
				_, err := serv.getBeneficiaries("cmsID", FileTypeDefault)
				return err
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(8)
			sp := suppressionParameters{true, lookbackDays}
			repository := &MockRepository{}
			repository.On("GetLatestCCLFFile", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(timeIsSetMatcher), time.Time{}, FileTypeDefault).Return(tt.cclfFileNew, nil)
			repository.On("GetLatestCCLFFile", mock.Anything, mock.Anything, mock.Anything, time.Time{}, mock.MatchedBy(timeIsSetMatcher), FileTypeDefault).Return(tt.cclfFileOld, nil)
			if tt.cclfFileOld != nil {
				repository.On("GetCCLFBeneficiaryMBIs", tt.cclfFileOld.ID).Return([]string{"1", "2", "3"}, nil)
			}

			var suppressedMBIs []string
			repository.On("GetCCLFBeneficiaries", tt.cclfFileNew.ID, suppressedMBIs).Return([]*CCLFBeneficiary{getCCLFBeneficiary(1, "1")}, nil)
			serviceInstance := &service{repository: repository, sp: sp, cutoffDuration: 1 * time.Hour}

			err := tt.funcUnderTest(serviceInstance)
			assert.NoError(t, err)

			repository.AssertNotCalled(t, "GetSuppressedMBIs", lookbackDays)
		})
	}
}

func (s *ServiceTestSuite) TestGetNewAndExistingBeneficiaries() {
	tests := []struct {
		name string

		cclfFileNew *CCLFFile
		cclfFileOld *CCLFFile

		oldMBIs []string

		expectedErr error
	}{
		{
			"NewAndExistingBenes",
			getCCLFFile(1),
			getCCLFFile(2),
			[]string{"123", "456"},
			nil,
		},
		{
			"NewBenesOnly",
			getCCLFFile(3),
			nil,
			nil,
			nil,
		},
		{
			"NoNewCCLFFileFound",
			nil,
			nil,
			nil,
			fmt.Errorf("no CCLF8 file found for cmsID"),
		},
		{
			"NoBenesFoundNew",
			getCCLFFile(4),
			nil,
			nil,
			fmt.Errorf("Found 0 new beneficiaries from CCLF8 file for cmsID"),
		},
		{
			"NoBenesFoundNewAndOld",
			getCCLFFile(5),
			getCCLFFile(6),
			nil,
			fmt.Errorf("Found 0 new or existing beneficiaries from CCLF8 file for cmsID"),
		},
		{
			"NoMBIsForOldCCLF",
			getCCLFFile(7),
			getCCLFFile(8),
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(30)
			fileNum := int(8)
			repository := &MockRepository{}
			cutoffDuration := 1 * time.Hour
			cmsID := "cmsID"
			since := time.Now().Add(-1 * time.Hour)

			var benes []*CCLFBeneficiary
			oldMBIs := make(map[string]bool)
			newMBIs := make(map[string]bool)
			beneID := uint(1)
			for _, mbiOld := range tt.oldMBIs {
				benes = append(benes, getCCLFBeneficiary(beneID, mbiOld))
				oldMBIs[mbiOld] = true
				beneID++
			}

			// Skip populating new benes under certain test conditions
			if tt.name != "NoBenesFoundNew" && tt.name != "NoBenesFoundNewAndOld" {
				for i := 0; i < 10; i++ {
					mbi := fmt.Sprintf("NewMBI%d", i)
					benes = append(benes, getCCLFBeneficiary(beneID, mbi))
					newMBIs[mbi] = true
					beneID++
				}
			}

			repository.On("GetLatestCCLFFile", cmsID, fileNum, constants.ImportComplete,
				// Verify our cutoffTime is bsed on our provided duration
				mock.MatchedBy(func(t time.Time) bool {
					// Since we're using time.Now() within the service call, we can't compare directly.
					// Make sure we're close enough.
					return time.Now().Add(-1*cutoffDuration).Sub(t) < time.Second
				}),
				time.Time{},
				FileTypeDefault).Return(tt.cclfFileNew, nil)
			repository.On("GetLatestCCLFFile", cmsID, fileNum, constants.ImportComplete, time.Time{}, since, FileTypeDefault).Return(tt.cclfFileOld, nil)

			if tt.cclfFileOld != nil {
				repository.On("GetCCLFBeneficiaryMBIs", tt.cclfFileOld.ID).Return(tt.oldMBIs, nil)
			}
			suppressedMBI := "suppressedMBI"
			if tt.cclfFileNew != nil {
				repository.On("GetCCLFBeneficiaries", tt.cclfFileNew.ID, []string{suppressedMBI}).Return(benes, nil)
			}
			repository.On("GetSuppressedMBIs", lookbackDays).Return([]string{suppressedMBI}, nil)

			serviceInstance := NewService(repository, 1*time.Hour, lookbackDays).(*service)
			newBenes, oldBenes, err := serviceInstance.getNewAndExistingBeneficiaries("cmsID", since)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.expectedErr.Error()),
					"Error %s does not contain substring %s", err.Error(), tt.expectedErr.Error())
				return
			}
			assert.NoError(t, err)

			for _, bene := range oldBenes {
				assert.True(t, oldMBIs[bene.MBI], "MBI %s should be found in old MBI map %v", bene.MBI, oldMBIs)
			}
			for _, bene := range newBenes {
				assert.True(t, newMBIs[bene.MBI], "MBI %s should be found in new MBI map %v", bene.MBI, newMBIs)
			}

		})
	}
}

func (s *ServiceTestSuite) TestGetBeneficiaries() {
	tests := []struct {
		name        string
		fileType    CCLFFileType
		cclfFile    *CCLFFile
		expectedErr error
	}{
		{
			"BenesReturned",
			FileTypeDefault,
			getCCLFFile(1),
			nil,
		},
		{
			"NoCCLFFileFound",
			FileTypeDefault,
			nil,
			fmt.Errorf("no CCLF8 file found for cmsID"),
		},
		{
			"NoBenesFound",
			FileTypeDefault,
			getCCLFFile(2),
			fmt.Errorf("Found 0 beneficiaries from CCLF8 file for cmsID"),
		},
		{
			"BenesReturnedRunout",
			FileTypeRunout,
			getCCLFFile(3),
			nil,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(30)
			fileNum := int(8)
			repository := &MockRepository{}
			cutoffDuration := 1 * time.Hour
			cmsID := "cmsID"

			var benes []*CCLFBeneficiary
			mbis := make(map[string]bool)
			beneID := uint(1)
			// Skip populating benes under certain test conditions
			if tt.name != "NoBenesFound" {
				for i := 0; i < 10; i++ {
					mbi := fmt.Sprintf("MBI%d", i)
					benes = append(benes, getCCLFBeneficiary(beneID, mbi))
					mbis[mbi] = true
					beneID++
				}
			}
			repository.On("GetLatestCCLFFile", cmsID, fileNum, constants.ImportComplete,
				// Verify our cutoffTime is based on our provided duration
				mock.MatchedBy(func(t time.Time) bool {
					// Since we're using time.Now() within the service call, we can't compare directly.
					// Make sure we're close enough.
					return time.Now().Add(-1*cutoffDuration).Sub(t) < time.Second
				}),
				time.Time{}, tt.fileType).Return(tt.cclfFile, nil)

			suppressedMBI := "suppressedMBI"
			repository.On("GetSuppressedMBIs", lookbackDays).Return([]string{suppressedMBI}, nil)
			if tt.cclfFile != nil {
				repository.On("GetCCLFBeneficiaries", tt.cclfFile.ID, []string{suppressedMBI}).Return(benes, nil)
			}

			serviceInstance := NewService(repository, 1*time.Hour, lookbackDays).(*service)
			benes, err := serviceInstance.getBeneficiaries("cmsID", tt.fileType)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), tt.expectedErr.Error()),
					"Error %s does not contain substring %s", err.Error(), tt.expectedErr.Error())
				return
			}
			assert.NoError(t, err)

			for _, bene := range benes {
				assert.True(t, mbis[bene.MBI], "MBI %s should be found in MBI map %v", bene.MBI, mbis)
			}
		})
	}
}

func getCCLFFile(id uint) *CCLFFile {
	return &CCLFFile{
		Model: gorm.Model{ID: id},
	}
}

func getCCLFBeneficiary(id uint, mbi string) *CCLFBeneficiary {
	return &CCLFBeneficiary{
		Model: gorm.Model{ID: id},
		MBI:   mbi,
	}
}

func timeIsSetMatcher(t time.Time) bool {
	return !t.IsZero()
}
