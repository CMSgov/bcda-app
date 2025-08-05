package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/database/databasetest"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/CMSgov/bcda-app/bcda/models/postgres/postgrestest"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcdaworker/queueing/worker_types"
	"github.com/CMSgov/bcda-app/conf"
	"github.com/CMSgov/bcda-app/middleware"
	"github.com/ccoveille/go-safecast"
	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const (
	defaultRunoutCutoff = 120 * 24 * time.Hour
)

var (
	defaultRunoutClaimThru = time.Date(time.Now().Year()-1, time.December, 31, 23, 59, 59, 999999, time.UTC)
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

		{"CKCC too short", "C999", false},
		{"CKCC too long", "C99999", false},
		{"CKCC invalid characters", "C999V", false},
		{"valid CKCC", "C9999", true},

		{"KCF too short", "K999", false},
		{"KCF too long", "K99999", false},
		{"KCF invalid characters", "K999V", false},
		{"valid KCF", "K9999", true},

		{"DC too short", "D999", false},
		{"DC too long", "D99999", false},
		{"DC invalid characters", "D999V", false},
		{"valid DC", "D9999", true},

		{"MDTCOC too short", "CT999", false},
		{"MDTCOC too long", "CT9999999", false},
		{"MDTCOC invalid characters", "CT999V", false},
		{"valid MDTCOC", "CT99999", true},

		{"CDAC too short", "DA999", false},
		{"CDAC too long", "DA9999999", false},
		{"CDAC invalid characters", "DA999V", false},
		{"valid CDAC", "DA9999", true},

		{"GUIDE too short", "GUIDE-999", false},
		{"GUIDE too long", "GUIDE-9999999", false},
		{"GUIDE invalid characters", "GUIDE99999", false},
		{"valid GUIDE", "GUIDE-99999", true},

		{"SBX too short", "SBXB1", false},
		{"SBX too long", "SBXPA0123", false},
		{"SBX invalid characters 1", "SBX0A123", false},
		{"SBX invalid characters 2", "SBXA0123", false},
		{"SBX invalid characters 3", "SBXADXYZ", false},
		{"valid SBX", "SBXAD123", true},

		{"Unregistered ACO", "Z1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(sub *testing.T) {
			match := IsSupportedACO(tt.cmsID)
			assert.Equal(sub, tt.isSupported, match)
		})
	}
}

func TestGetMaxBeneCount(t *testing.T) {
	defer func() {
		conf.UnsetEnv(t, "BCDA_FHIR_MAX_RECORDS_EOB")
		conf.UnsetEnv(t, "BCDA_FHIR_MAX_RECORDS_PATIENT")
		conf.UnsetEnv(t, "BCDA_FHIR_MAX_RECORDS_COVERAGE")
		conf.UnsetEnv(t, "BCDA_FHIR_MAX_RECORDS_CLAIM")
		conf.UnsetEnv(t, "BCDA_FHIR_MAX_RECORDS_CLAIM_RESPONSE")
	}()

	getEnvVar := func(resourceType string) string {
		switch resourceType {
		case "ExplanationOfBenefit":
			return "BCDA_FHIR_MAX_RECORDS_EOB"
		case "Patient":
			return "BCDA_FHIR_MAX_RECORDS_PATIENT"
		case "Coverage":
			return "BCDA_FHIR_MAX_RECORDS_COVERAGE"
		case "Claim":
			return "BCDA_FHIR_MAX_RECORDS_CLAIM"
		case "ClaimResponse":
			return "BCDA_FHIR_MAX_RECORDS_CLAIM_RESPONSE"
		default:
			return ""
		}
	}

	clearer := func(resourceType string, val int) {
		conf.UnsetEnv(t, getEnvVar(resourceType))
	}
	setter := func(resourceType string, val int) {
		conf.SetEnv(t, getEnvVar(resourceType), strconv.Itoa(val))
	}

	tests := []struct {
		name     string
		resource string
		expVal   int
		setup    func(resourceType string, val int)
	}{
		{"DefaultEOB", "ExplanationOfBenefit", 50, clearer},
		{"MaxEOB", "ExplanationOfBenefit", 5, setter},
		{"DefaultPatient", "Patient", 5000, clearer},
		{"MaxPatient", "Patient", 10, setter},
		{"DefaultCoverage", "Coverage", 4000, clearer},
		{"MaxCoverage", "Coverage", 15, setter},
		{"defaultClaim", "Claim", 4000, clearer},
		{"MaxClaim", "Claim", 20, setter},
		{"defaultClaimResponse", "ClaimResponse", 4000, clearer},
		{"MaxClaimResponse", "ClaimResponse", 25, setter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(sub *testing.T) {
			tt.setup(tt.resource, tt.expVal)
			max, err := getMaxBeneCount(tt.resource)
			assert.NoError(sub, err)
			assert.Equal(sub, tt.expVal, max)
		})
	}

	// Invalid type
	max, err := getMaxBeneCount("Coverages")
	assert.Equal(t, -1, max)
	assert.EqualError(t, err, "invalid request type")
}

func TestGetJobAndKeys(t *testing.T) {
	ipJob := models.Job{ID: 22, Status: models.JobStatusInProgress}
	ipJobEmptyKey := models.Job{ID: 46, Status: models.JobStatusInProgress}
	completeJob := models.Job{ID: 49, Status: models.JobStatusCompleted}
	key := models.JobKey{ID: 101, FileName: "goodFile.ndjson"}
	emptyKey := models.JobKey{ID: 155, FileName: models.BlankFileName}

	repository := models.NewMockRepository(t)
	configureMockRepo := func(job models.Job, jobKeys []*models.JobKey) {
		repository.On("GetJobByID", testUtils.CtxMatcher, job.ID).Return(&job, nil)
		repository.On("GetJobKeys", testUtils.CtxMatcher, job.ID).Return(jobKeys, nil)
	}
	ctx := context.Background()
	serviceInstance := NewService(repository, &Config{}, "").(*service)

	tests := []struct {
		name         string
		job          models.Job
		jobKeys      []*models.JobKey
		expectedKeys []*models.JobKey
	}{
		{"In Progress job with 1 key returns job and key", ipJob, []*models.JobKey{&key}, []*models.JobKey{&key}},
		{"In Progress job with 1 empty key returns job and no keys", ipJobEmptyKey, []*models.JobKey{&emptyKey}, nil},
		{"Complete job with 1 key returns job and key", completeJob, []*models.JobKey{&key}, []*models.JobKey{&key}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configureMockRepo(tt.job, tt.expectedKeys)
			serviceJob, serviceKeys, err := serviceInstance.GetJobAndKeys(ctx, tt.job.ID)
			assert.Equal(t, tt.job.ID, serviceJob.ID)
			assert.True(t, slices.Equal(tt.expectedKeys, serviceKeys))
			assert.Nil(t, err)
		})
	}
}

func (s *ServiceTestSuiteWithDatabase) TestGetTimeConstraints() {
	repository := &models.MockRepository{}
	svc := NewService(repository, &Config{}, "")

	repository.On("GetACOByCMSID", testUtils.CtxMatcher, mock.Anything).Return(&models.ACO{}, nil)

	tC, err := svc.GetTimeConstraints(context.Background(), "A0000")

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), tC, TimeConstraints{})
}

func (s *ServiceTestSuiteWithDatabase) TestGetTimeConstraints_TerminatedACO() {
	now := time.Now()
	repository := &models.MockRepository{}
	svc := NewService(repository, &Config{}, "")

	repository.On("GetACOByCMSID", testUtils.CtxMatcher, mock.Anything).Return(&models.ACO{
		ID: 2,
		TerminationDetails: &models.Termination{
			TerminationDate:     now,
			AttributionStrategy: models.AttributionHistorical,
			ClaimsStrategy:      models.ClaimsHistorical,
			OptOutStrategy:      models.OptOutHistorical,
		},
	}, nil)

	tC, err := svc.GetTimeConstraints(context.Background(), "A0000")

	assert.Nil(s.T(), err)
	assert.Equal(s.T(), tC, TimeConstraints{
		AttributionDate: now,
		ClaimsDate:      now,
		OptOutDate:      now,
	})
}

func (s *ServiceTestSuite) TestGetTimeConstraints_ACOError() {
	args := worker_types.PrepareJobArgs{ACOID: uuid.NewRandom(), CMSID: uuid.New()}
	repository := &models.MockRepository{}
	repository.On("GetACOByCMSID", testUtils.CtxMatcher, mock.AnythingOfType("string")).
		Return(nil, context.DeadlineExceeded)
	defer repository.AssertExpectations(s.T())
	service := &service{repository: repository}
	tC, err := service.GetTimeConstraints(context.Background(), args.CMSID)
	assert.Equal(s.T(), tC, TimeConstraints{})
	assert.True(s.T(), errors.Is(err, context.DeadlineExceeded), "Root cause should be deadline exceeded")
}

///////////////////////////////////////////////////////////////////////////
/////////////////////////// INTEGRATION TESTS /////////////////////////////
///////////////////////////////////////////////////////////////////////////

type ServiceTestSuite struct {
	suite.Suite
	priorityACOsEnvVar string
}

// Run all test suite tests
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) SetupTest() {
	s.priorityACOsEnvVar = conf.GetEnv("PRIORITY_ACO_REG_EX")
}

func (s *ServiceTestSuite) TearDownTest() {
	conf.SetEnv(s.T(), "PRIORITY_ACO_REG_EX", s.priorityACOsEnvVar)
}

func (s *ServiceTestSuite) TestIncludeSuppressedBeneficiaries_Integration() {
	args := worker_types.PrepareJobArgs{
		CMSID: "cmsID",
		Since: time.Now(),
	}
	tests := []struct {
		name          string
		cclfFileNew   *models.CCLFFile
		cclfFileOld   *models.CCLFFile
		funcUnderTest func(s *service) error
	}{
		{
			"GetNewAndExistingBeneficiaries",
			getCCLFFile(1, false, false),
			getCCLFFile(2, false, false),
			func(serv *service) error {
				_, _, err := serv.getNewAndExistingBeneficiaries(context.Background(), args)
				return err
			},
		},
		{
			"GetBeneficiaries",
			getCCLFFile(3, false, false),
			nil,
			func(serv *service) error {
				_, err := serv.getBeneficiaries(context.Background(), args)
				return err
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(8)
			sp := suppressionParameters{true, lookbackDays}
			repository := &models.MockRepository{}
			repository.On("GetCCLFFileByID", testUtils.CtxMatcher, mock.AnythingOfType("uint")).Return(tt.cclfFileNew, nil)
			if tt.cclfFileOld != nil {
				repository.On("GetCCLFFileByID", testUtils.CtxMatcher, tt.cclfFileOld.ID).Return(tt.cclfFileOld, nil)
				repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, mock.AnythingOfType("uint")).Return([]string{"1", "2", "3"}, nil)
			}

			var suppressedMBIs []string
			repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, mock.AnythingOfType("uint"), suppressedMBIs).Return([]*models.CCLFBeneficiary{getCCLFBeneficiary(1, "1")}, nil)
			serviceInstance := &service{repository: repository, sp: sp, stdCutoffDuration: 1 * time.Hour}

			err := tt.funcUnderTest(serviceInstance)
			assert.NoError(t, err)

			repository.AssertNotCalled(t, "GetSuppressedMBIs", testUtils.CtxMatcher, lookbackDays, time.Time{})
		})
	}
}

func (s *ServiceTestSuite) TestGetNewAndExistingBeneficiaries_Integration() {
	tests := []struct {
		name        string
		cclfFileNew *models.CCLFFile
		cclfFileOld *models.CCLFFile
		oldMBIs     []string
		expectedErr error
	}{
		{
			"NewAndExistingBenes",
			getCCLFFile(1, false, false),
			getCCLFFile(2, false, false),
			[]string{"123", "456"},
			nil,
		},
		{
			"NewBenesOnly",
			getCCLFFile(3, false, false),
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
			getCCLFFile(4, false, false),
			nil,
			nil,
			fmt.Errorf("found 0 new or existing beneficiaries from CCLF8 file for cmsID"),
		},
		{
			"NoBenesFoundNewAndOld",
			getCCLFFile(5, false, false),
			getCCLFFile(6, false, false),
			nil,
			fmt.Errorf("found 0 new or existing beneficiaries from CCLF8 file for cmsID"),
		},
		{
			"NoMBIsForOldCCLF",
			getCCLFFile(7, false, false),
			getCCLFFile(8, false, false),
			nil,
			nil,
		},
		// no longer validating CCLF files in this function, test no longer relevant
		// {
		// 	"NoCCLFPerfYearIncompatible",
		// 	getCCLFFile(7, false, true),
		// 	getCCLFFile(8, false, false),
		// 	nil,
		// 	fmt.Errorf("no CCLF8 file found for cmsID"),
		// },
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(30)
			repository := &models.MockRepository{}
			cmsID := "A0000"
			since := time.Now().Add(-1 * time.Hour)
			now := time.Now().Round(time.Millisecond)
			// Since we're using time.Now() within the service call, we can't compare directly.
			// Make sure we're close enough.
			mockUpperBound := mock.MatchedBy(func(t time.Time) bool {
				return now.Sub(t) < time.Second
			})

			var benes []*models.CCLFBeneficiary
			oldMBIs := make(map[string]bool)
			newMBIs := make(map[string]bool)
			beneID := uint(1)
			for _, mbiOld := range tt.oldMBIs {
				benes = append(benes, getCCLFBeneficiary(beneID, mbiOld))
				oldMBIs[mbiOld] = true
				beneID++
			}

			// Skip populating new benes under certain test args
			if tt.name != "NoBenesFoundNew" && tt.name != "NoBenesFoundNewAndOld" {
				for i := 0; i < 10; i++ {
					mbi := fmt.Sprintf("NewMBI%d", i)
					benes = append(benes, getCCLFBeneficiary(beneID, mbi))
					newMBIs[mbi] = true
					beneID++
				}
			}

			repository.On("GetCCLFFileByID", testUtils.CtxMatcher, mock.AnythingOfType("uint")).Return(tt.cclfFileNew, nil)
			if tt.cclfFileNew != nil {
				repository.On("GetCCLFFileByID", testUtils.CtxMatcher, mock.AnythingOfType("uint")).Return(tt.cclfFileOld, nil)
			}
			repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, mock.AnythingOfType("uint")).Return(tt.oldMBIs, nil)
			suppressedMBI := "suppressedMBI"
			if tt.cclfFileNew != nil {
				repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, tt.cclfFileNew.ID, []string{suppressedMBI}).Return(benes, nil)
			}
			repository.On("GetSuppressedMBIs", testUtils.CtxMatcher, lookbackDays, mockUpperBound).Return([]string{suppressedMBI}, nil)

			acoConfigs, _ := LoadConfig()

			cfg := &Config{
				CutoffDuration:          time.Hour,
				SuppressionLookbackDays: lookbackDays,
				RunoutConfig: RunoutConfig{
					CutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
				ACOConfigs: acoConfigs.ACOConfigs,
			}
			serviceInstance := NewService(repository, cfg, "").(*service)
			newBenes, oldBenes, err := serviceInstance.getNewAndExistingBeneficiaries(
				context.Background(),
				worker_types.PrepareJobArgs{
					CMSID: cmsID,
					Since: since,
				},
			)

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

// * Live database test *
//
// Given the following example scenario:
// - CCLF File 1 (July 1 timestamp)
// - CCLF File 2 (July 1 timestamp, July 3rd created at)
//
// This tests two scenarios:
// - Request made with "since" parameter on July 2nd
// - Request made with "since" parameter after July 3rd
//
// We should diff between the correct files:
// - Diff between CCLF File 1 and CCLF File 2
// - No diff - consider all beneficiaries as pre-existing
func (s *ServiceTestSuite) TestGetNewAndExistingBeneficiaries_RecentSinceParameter_Integration() {
	db := database.GetConnection()
	acoID := "A0005"

	// Test Setup
	testSetup := func(t *testing.T, populateBenes bool) (uint, uint, []string, func()) {
		postgrestest.DeleteCCLFFilesByCMSID(t, db, "A0005")

		performanceYear := time.Now().Year() % 100
		cclfFileOld := &models.CCLFFile{CCLFNum: 8, ACOCMSID: acoID, Timestamp: time.Now().Add(-48 * time.Hour), PerformanceYear: performanceYear, Name: "T.BCD.A0005.ZC8Y23.D231119.T1000009", ImportStatus: constants.ImportComplete}
		cclfFileNew := &models.CCLFFile{CCLFNum: 8, ACOCMSID: acoID, Timestamp: time.Now().Add(-24 * time.Hour), PerformanceYear: performanceYear, Name: "T.BCD.A0005.ZC8Y23.D231120.T1000009", ImportStatus: constants.ImportComplete}
		postgrestest.CreateCCLFFile(t, db, cclfFileOld)
		postgrestest.CreateCCLFFile(t, db, cclfFileNew)

		if populateBenes {
			bene1OldRecord := &models.CCLFBeneficiary{FileID: cclfFileOld.ID, MBI: testUtils.RandomMBI(t), BlueButtonID: testUtils.RandomHexID()}
			bene1NewRecord := &models.CCLFBeneficiary{FileID: cclfFileNew.ID, MBI: bene1OldRecord.MBI, BlueButtonID: testUtils.RandomHexID()}
			bene2NewRecord := &models.CCLFBeneficiary{FileID: cclfFileNew.ID, MBI: testUtils.RandomMBI(t), BlueButtonID: testUtils.RandomHexID()}

			postgrestest.CreateCCLFBeneficiary(t, db, bene1OldRecord)
			postgrestest.CreateCCLFBeneficiary(t, db, bene1NewRecord)
			postgrestest.CreateCCLFBeneficiary(t, db, bene2NewRecord)
			return cclfFileNew.ID, cclfFileOld.ID, []string{bene1OldRecord.MBI, bene2NewRecord.MBI}, func() { postgrestest.DeleteCCLFFilesByCMSID(t, db, "A0005") }
		} else {
			return 0, 0, []string{}, func() { postgrestest.DeleteCCLFFilesByCMSID(t, db, "A0005") }
		}
	}

	tests := []struct {
		name                  string
		sinceOffset           time.Duration
		expectedOldMBIIndexes []int
		expectedNewMBIIndexes []int
		populateBenes         bool
	}{
		{
			"BetweenTimestampAndCreatedAt",
			-12,
			[]int{0},
			[]int{1},
			true,
		},
		{
			"LaterThanCreatedAt",
			1,
			[]int{0, 1},
			[]int{},
			true,
		},
		// no longer relevant, we dont check these things anymore, they are done before
		// {
		// 	"LaterThanCreatedAtNoBenes",
		// 	1,
		// 	[]int{},
		// 	[]int{},
		// 	false,
		// },
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			newCCLFID, oldCCLFID, generatedMbis, cleanup := testSetup(t, tt.populateBenes)
			defer cleanup()

			acoCfgs, err := LoadConfig()
			assert.Nil(err)

			cfg := &Config{
				CutoffDuration:          -50 * time.Hour,
				SuppressionLookbackDays: int(30),
				RunoutConfig: RunoutConfig{
					CutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
				ACOConfigs: acoCfgs.ACOConfigs,
			}

			since := time.Now().Add(tt.sinceOffset * time.Hour)

			repository := postgres.NewRepository(db)
			serviceInstance := NewService(repository, cfg, "").(*service)
			newBenes, oldBenes, err := serviceInstance.getNewAndExistingBeneficiaries(
				context.Background(),
				worker_types.PrepareJobArgs{
					CMSID:         acoID,
					Since:         since,
					CCLFFileNewID: newCCLFID,
					CCLFFileOldID: oldCCLFID,
				},
			)

			// Assert
			if !tt.populateBenes {
				assert.ErrorContains(err, "found 0 new or existing beneficiaries from CCLF8 file for cmsID A0005")
			} else {
				assert.NoError(err)
				assert.Len(oldBenes, len(tt.expectedOldMBIIndexes))
				assert.Len(newBenes, len(tt.expectedNewMBIIndexes))

				contains := func(arr []*models.CCLFBeneficiary, mbi string) bool {
					for _, bene := range arr {
						if bene.MBI == mbi {
							return true
						}
					}
					return false
				}

				for _, mbiIdx := range tt.expectedOldMBIIndexes {
					assert.True(contains(oldBenes, generatedMbis[mbiIdx]), "MBI %s should be found in old MBI map %v", generatedMbis[mbiIdx], oldBenes)
				}

				for _, mbiIdx := range tt.expectedNewMBIIndexes {
					assert.True(contains(newBenes, generatedMbis[mbiIdx]), "MBI %s should be found in new MBI map %v", generatedMbis[mbiIdx], newBenes)
				}
			}
		})
	}
}

func (s *ServiceTestSuite) TestGetBeneficiaries_Integration() {
	tests := []struct {
		name        string
		fileType    models.CCLFFileType
		cclfFile    *models.CCLFFile
		expectedErr error
	}{
		{
			"BenesReturned",
			models.FileTypeDefault,
			getCCLFFile(1, false, false),
			nil,
		},
		{
			"NoCCLFFileFound",
			models.FileTypeDefault,
			nil,
			fmt.Errorf("no CCLF8 file found for cmsID"),
		},
		{
			"NoBenesFound",
			models.FileTypeDefault,
			getCCLFFile(2, false, false),
			fmt.Errorf("found 0 beneficiaries from CCLF8 file for cmsID"),
		},
		{
			"BenesReturnedRunout",
			models.FileTypeRunout,
			getCCLFFile(3, true, false),
			nil,
		},
		// function no longer does various logic trying to find a specific cclf file, these tests are no longer relevant
		// {
		// 	"NoBenesReturnedOld",
		// 	models.FileTypeRunout,
		// 	getCCLFFile(4, false, true),
		// 	fmt.Errorf("no CCLF8 file found for cmsID"),
		// },
		// {
		// 	"NoBenesReturnedOldRunout",
		// 	models.FileTypeRunout,
		// 	getCCLFFile(4, true, true),
		// 	fmt.Errorf("no CCLF8 file found for cmsID"),
		// },
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(30)
			repository := &models.MockRepository{}
			cmsID := "A0000"
			now := time.Now().Round(time.Millisecond)
			// Since we're using time.Now() within the service call, we can't compare directly.
			// Make sure we're close enough.
			mockUpperBound := mock.MatchedBy(func(t time.Time) bool {
				return now.Sub(t) < time.Second
			})

			var benes []*models.CCLFBeneficiary
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
			repository.On("GetCCLFFileByID", testUtils.CtxMatcher, mock.AnythingOfType("uint")).Return(tt.cclfFile, nil)
			suppressedMBI := "suppressedMBI"
			repository.On("GetSuppressedMBIs", testUtils.CtxMatcher, lookbackDays, mockUpperBound).Return([]string{suppressedMBI}, nil)
			if tt.cclfFile != nil {
				repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, tt.cclfFile.ID, []string{suppressedMBI}).Return(benes, nil)
			}

			acoConfigs, _ := LoadConfig()

			cfg := &Config{
				CutoffDuration:          time.Hour,
				SuppressionLookbackDays: lookbackDays,
				RunoutConfig: RunoutConfig{
					CutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
				ACOConfigs: acoConfigs.ACOConfigs,
			}
			serviceInstance := NewService(repository, cfg, "").(*service)
			benes, err := serviceInstance.getBeneficiaries(
				context.Background(),
				worker_types.PrepareJobArgs{
					CMSID: cmsID,
				},
			)

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

func (s *ServiceTestSuite) TestGetQueJobs_Integration() {
	defaultACOID, lookbackACOID := "SOME_ACO_ID", "LOOKBACK_ACO"

	defaultACO := ACOConfig{
		patternExp: regexp.MustCompile(defaultACOID),
		Data:       []string{constants.Adjudicated},
	}

	lookbackACO := ACOConfig{
		patternExp:    regexp.MustCompile(lookbackACOID),
		LookbackYears: 3,
		perfYear:      time.Now(),
		Data:          []string{constants.Adjudicated},
	}

	acoCfgs := []ACOConfig{
		defaultACO,
		lookbackACO,
	}

	benes1, benes2 := make([]*models.CCLFBeneficiary, 10), make([]*models.CCLFBeneficiary, 20)
	allBenes := [][]*models.CCLFBeneficiary{benes1, benes2}
	for idx, b := range allBenes {
		for i := 0; i < len(b); i++ {
			id, _ := safecast.ToUint(idx*10000 + i + 1)
			b[i] = getCCLFBeneficiary(id, fmt.Sprintf("MBI%d", id))
		}
	}
	benes1MBI := make([]string, 0, len(benes1))
	benes1ID := make(map[string]struct{})
	for _, bene := range benes1 {
		benes1MBI = append(benes1MBI, bene.MBI)
		benes1ID[strconv.FormatUint(uint64(bene.ID), 10)] = struct{}{}
	}

	since := time.Now()
	terminationHistorical := &models.Termination{
		ClaimsStrategy:      models.ClaimsHistorical,
		AttributionStrategy: models.AttributionHistorical,
		OptOutStrategy:      models.OptOutHistorical,
		TerminationDate:     time.Now().Add(-30 * 24 * time.Hour).Round(time.Millisecond).UTC(),
	}

	terminationLatest := &models.Termination{
		ClaimsStrategy:      models.ClaimsLatest,
		AttributionStrategy: models.AttributionLatest,
		OptOutStrategy:      models.OptOutLatest,
		TerminationDate:     time.Now().Add(-30 * 24 * time.Hour).Round(time.Millisecond).UTC(),
	}

	sinceAfterTermination := terminationHistorical.TerminationDate.Add(10 * 24 * time.Hour)
	sinceBeforeTermination := terminationHistorical.TerminationDate.Add(-10 * 24 * time.Hour)

	type claimsWindow struct {
		LowerBound time.Time
		UpperBound time.Time
	}

	type test struct {
		name               string
		acoID              string
		RequestType        constants.DataRequestType
		ComplexRequestType string
		expSince           time.Time
		expClaimsWindow    claimsWindow
		expBenes           []*models.CCLFBeneficiary
		resourceTypes      []string
		terminationDetails *models.Termination
	}

	baseTests := []test{
		{"BasicRequest (non-Group)", defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, time.Time{}, claimsWindow{}, benes1, nil, nil},
		{"BasicRequest with Since (non-Group) ", defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, since, claimsWindow{}, benes1, nil, nil},
		{"GroupAll", defaultACOID, constants.RetrieveNewBeneHistData, constants.GetNewAndExistingBenes, since, claimsWindow{}, append(benes1, benes2...), nil, nil},
		{"RunoutRequest", defaultACOID, constants.Runout, constants.GetExistingBenes, time.Time{}, claimsWindow{UpperBound: defaultRunoutClaimThru}, benes1, nil, nil},
		{"RunoutRequest with Since", defaultACOID, constants.Runout, constants.GetExistingBenes, since, claimsWindow{UpperBound: defaultRunoutClaimThru}, benes1, nil, nil},

		// Terminated ACOs: historical
		{"Since After Termination", defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, sinceAfterTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
		{"Since Before Termination", defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, sinceBeforeTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
		{"New Benes With Since After Termination", defaultACOID, constants.RetrieveNewBeneHistData, constants.GetNewAndExistingBenes, sinceAfterTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
		{"New Benes With Since Before Termination", defaultACOID, constants.RetrieveNewBeneHistData, constants.GetNewAndExistingBenes, sinceBeforeTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, append(benes1, benes2...), nil, terminationHistorical},
		// Runout cutoff takes precedence over termination cutoff
		{"TerminatedACORunout", defaultACOID, constants.Runout, constants.GetExistingBenes, time.Time{}, claimsWindow{UpperBound: defaultRunoutClaimThru}, benes1, nil, terminationHistorical},

		// Terminated ACOs: latest
		{"Since After Termination", defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, sinceAfterTermination, claimsWindow{}, benes1, nil, terminationLatest},
		{"Since Before Termination", defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, sinceBeforeTermination, claimsWindow{}, benes1, nil, terminationLatest},
		// should still receive full benes since Attribution is set to latest
		{"New Benes With Since After Termination", defaultACOID, constants.RetrieveNewBeneHistData, constants.GetNewAndExistingBenes, sinceAfterTermination, claimsWindow{}, append(benes1, benes2...), nil, terminationLatest},
		{"New Benes With Since Before Termination", defaultACOID, constants.RetrieveNewBeneHistData, constants.GetNewAndExistingBenes, sinceBeforeTermination, claimsWindow{}, append(benes1, benes2...), nil, terminationLatest},

		// ACO with lookback period
		{"ACO with lookback", lookbackACOID, constants.DefaultRequest, constants.GetExistingBenes, time.Time{}, claimsWindow{LowerBound: lookbackACO.LookbackTime()}, benes1, nil, nil},
		{"Terminated ACO with lookback", lookbackACOID, constants.DefaultRequest, constants.GetExistingBenes, time.Time{}, claimsWindow{LowerBound: lookbackACO.LookbackTime(), UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
	}

	// Add all combinations of resource types
	var tests []test
	for _, resourceTypes := range [][]string{{"ExplanationOfBenefit"}, {"Patient"}, {"Coverage"},
		{"ExplanationOfBenefit", "Coverage"}, {"ExplanationOfBenefit", "Patient"}, {"Patient", "Coverage"},
		{"ExplanationOfBenefit", "Patient", "Coverage"}} {
		for _, baseTest := range baseTests {
			baseTest.resourceTypes = resourceTypes
			baseTest.name = fmt.Sprintf("%s-%s", baseTest.name, strings.Join(resourceTypes, ","))
			tests = append(tests, baseTest)
		}
	}

	basePath := "/v2/fhir"
	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			args := worker_types.PrepareJobArgs{
				CMSID:                  tt.acoID,
				ACOID:                  uuid.NewUUID(),
				ResourceTypes:          tt.resourceTypes,
				Since:                  tt.expSince,
				RequestType:            tt.RequestType,
				ComplexDataRequestType: tt.ComplexRequestType,
				BFDPath:                basePath,
				ClaimsDate:             tt.expClaimsWindow.UpperBound,
			}

			repository := &models.MockRepository{}
			repository.On("GetACOByCMSID", testUtils.CtxMatcher, args.CMSID).
				Return(&models.ACO{UUID: args.ACOID, TerminationDetails: tt.terminationDetails}, nil)
			if tt.RequestType == constants.Runout {
				repository.On("GetCCLFFileByID", testUtils.CtxMatcher, mock.Anything).Return(getCCLFFile(1, true, false), nil)
			} else {
				repository.On("GetCCLFFileByID", testUtils.CtxMatcher, mock.Anything).Return(getCCLFFile(1, false, false), nil)
			}
			repository.On("GetSuppressedMBIs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(nil, nil)
			repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(tt.expBenes, nil)
			// use benes1 as the "old" benes. Allows us to verify the since parameter is populated as expected
			repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, mock.Anything).Return(benes1MBI, nil)

			cfg := &Config{
				CutoffDuration:          time.Hour,
				SuppressionLookbackDays: 0,
				RunoutConfig: RunoutConfig{
					CutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
			}
			serviceInstance := NewService(repository, cfg, basePath)
			serviceInstance.(*service).acoConfigs = acoCfgs
			ctx := context.Background()

			queJobs, err := serviceInstance.GetQueJobs(context.WithValue(ctx, middleware.CtxTransactionKey, uuid.New()), args)
			assert.NoError(t, err)
			// map tuple of resourceType:beneID
			benesInJob := make(map[string]map[string]struct{})
			for _, qj := range queJobs {
				assert.True(t, tt.expClaimsWindow.LowerBound.Equal(qj.ClaimsWindow.LowerBound),
					"Lower bounds should equal. Have %s. Want %s", qj.ClaimsWindow.LowerBound, tt.expClaimsWindow.LowerBound)
				assert.True(t, tt.expClaimsWindow.UpperBound.Equal(qj.ClaimsWindow.UpperBound),
					"Upper bounds should equal. Have %s. Want %s", qj.ClaimsWindow.UpperBound, tt.expClaimsWindow.UpperBound)

				subMap := benesInJob[qj.ResourceType]
				if subMap == nil {
					subMap = make(map[string]struct{})
					benesInJob[qj.ResourceType] = subMap
				}

				// Need to see if the bene is considered "new" or not. If the bene
				// is new, we should not provide a since parameter (need full history)
				var expectedTime time.Time
				if !tt.expSince.IsZero() {
					var hasNewBene bool
					for _, beneID := range qj.BeneficiaryIDs {
						if _, ok := benes1ID[beneID]; !ok {
							hasNewBene = true
							break
						}
					}
					if !hasNewBene {
						expectedTime = tt.expSince
					}
				}
				if expectedTime.IsZero() {
					assert.Empty(t, qj.Since)
				} else {
					assert.Equal(t, fmt.Sprintf("gt%s", expectedTime.Format(time.RFC3339Nano)), qj.Since)
				}

				for _, beneID := range qj.BeneficiaryIDs {
					subMap[beneID] = struct{}{}
				}

				assert.Equal(t, basePath, qj.BBBasePath)
			}

			for _, resourceType := range tt.resourceTypes {
				subMap := benesInJob[resourceType]
				assert.NotNil(t, subMap)
				for _, bene := range tt.expBenes {
					assert.Contains(t, subMap, strconv.FormatUint(uint64(bene.ID), 10))
				}
			}
		})
	}
}

func (s *ServiceTestSuite) TestGetQueJobsErrorHandling_Integration() {
	defaultACOID := "SOME_ACO_ID"

	defaultACO := ACOConfig{
		patternExp: regexp.MustCompile(defaultACOID),
		Data:       []string{constants.Adjudicated, constants.PartiallyAdjudicated},
	}

	acoCfgs := []ACOConfig{
		defaultACO,
	}

	cfg := &Config{
		CutoffDuration:          time.Hour,
		SuppressionLookbackDays: 0,
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
	}

	ctx := context.Background()
	basePath := "/v2/fhir"

	s.T().Run("Unexpected request type", func(t *testing.T) {
		args := worker_types.PrepareJobArgs{
			CMSID:       defaultACOID,
			ACOID:       uuid.NewUUID(),
			RequestType: 22,
		}
		repository := &models.MockRepository{}
		repository.On("GetACOByCMSID", testUtils.CtxMatcher, args.CMSID).Return(&models.ACO{UUID: args.ACOID, TerminationDetails: nil}, nil)
		serviceInstance := NewService(repository, cfg, basePath)
		serviceInstance.(*service).acoConfigs = acoCfgs
		_, err := serviceInstance.GetQueJobs(context.WithValue(ctx, middleware.CtxTransactionKey, uuid.New()), args)

		assert.Error(t, err, errors.New("Unsupported RequestType 22"))
	})

	s.T().Run("s.getBeneficiaries failure", func(t *testing.T) {
		args := worker_types.PrepareJobArgs{
			CMSID:       defaultACOID,
			ACOID:       uuid.NewUUID(),
			RequestType: constants.DefaultRequest,
		}
		repository := &models.MockRepository{}
		repository.On("GetACOByCMSID", testUtils.CtxMatcher, args.CMSID).Return(&models.ACO{UUID: args.ACOID, TerminationDetails: nil}, nil)
		repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("forced failure"))
		serviceInstance := NewService(repository, cfg, basePath)
		serviceInstance.(*service).acoConfigs = acoCfgs
		_, err := serviceInstance.GetQueJobs(context.WithValue(ctx, middleware.CtxTransactionKey, uuid.New()), args)

		assert.Error(t, err, errors.New("forced failure"))
	})

	s.T().Run("s.getNewAndExistingBeneficiaries failure", func(t *testing.T) {
		args := worker_types.PrepareJobArgs{
			CMSID:       defaultACOID,
			ACOID:       uuid.NewUUID(),
			RequestType: constants.RetrieveNewBeneHistData,
		}
		repository := &models.MockRepository{}
		repository.On("GetACOByCMSID", testUtils.CtxMatcher, args.CMSID).Return(&models.ACO{UUID: args.ACOID, TerminationDetails: nil}, nil)
		repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("forced failure"))
		serviceInstance := NewService(repository, cfg, basePath)
		serviceInstance.(*service).acoConfigs = acoCfgs
		_, err := serviceInstance.GetQueJobs(context.WithValue(ctx, middleware.CtxTransactionKey, uuid.New()), args)

		assert.Error(t, err, errors.New("forced failure"))
	})

	s.T().Run("s.createQueueJobs failure", func(t *testing.T) {
		args := worker_types.PrepareJobArgs{
			CMSID:       defaultACOID,
			ACOID:       uuid.NewUUID(),
			RequestType: constants.RetrieveNewBeneHistData,
		}
		repository := &models.MockRepository{}
		repository.On("GetACOByCMSID", testUtils.CtxMatcher, args.CMSID).Return(&models.ACO{UUID: args.ACOID, TerminationDetails: nil}, nil)
		repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getCCLFFile(1, false, false), nil)
		repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(nil, nil)
		repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, mock.Anything).Return([]string{"old"}, nil)
		serviceInstance := NewService(repository, cfg, basePath)
		serviceInstance.(*service).acoConfigs = acoCfgs
		_, err := serviceInstance.GetQueJobs(context.WithValue(ctx, middleware.CtxTransactionKey, uuid.New()), args)

		assert.Error(t, err, errors.New("forced failure"))
	})
}

func (s *ServiceTestSuite) TestGetQueJobsByDataType_Integration() {
	defaultACOID := "SOME_ACO_ID"

	defaultACO := ACOConfig{
		patternExp: regexp.MustCompile(defaultACOID),
		Data:       []string{constants.Adjudicated, constants.PartiallyAdjudicated},
	}

	acoCfgs := []ACOConfig{
		defaultACO,
	}

	benes1, benes2 := make([]*models.CCLFBeneficiary, 10), make([]*models.CCLFBeneficiary, 20)
	allBenes := [][]*models.CCLFBeneficiary{benes1, benes2}
	for idx, b := range allBenes {
		for i := 0; i < len(b); i++ {
			id, _ := safecast.ToUint(idx*10000 + i + 1)
			b[i] = getCCLFBeneficiary(id, fmt.Sprintf("MBI%d", id))
		}
	}
	benes1MBI := make([]string, 0, len(benes1))
	benes1ID := make(map[string]struct{})
	for _, bene := range benes1 {
		benes1MBI = append(benes1MBI, bene.MBI)
		benes1ID[strconv.FormatUint(uint64(bene.ID), 10)] = struct{}{}
	}

	type claimsWindow struct {
		LowerBound time.Time
		UpperBound time.Time
	}

	timeA := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	timeB := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	basePath := "/v2/fhir"

	tests := []struct {
		name               string
		acoID              string
		RequestType        constants.DataRequestType
		ComplexRequestType string
		expSince           time.Time
		expClaimsWindow    claimsWindow
		expBenes           []*models.CCLFBeneficiary
		expTxTime          time.Time
		resourceTypes      []string
		terminationDetails *models.Termination
		dataTypes          []string
	}{
		{constants.Adjudicated, defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, time.Time{}, claimsWindow{}, benes1, timeB, []string{"Patient"}, nil, []string{constants.Adjudicated}},
		{"PartiallyAdjudicated", defaultACOID, constants.DefaultRequest, constants.GetExistingBenes, time.Time{}, claimsWindow{}, benes1, timeA, []string{"Claim"}, nil, []string{constants.PartiallyAdjudicated}},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			args := worker_types.PrepareJobArgs{
				Job:                    models.Job{TransactionTime: tt.expTxTime},
				CMSID:                  tt.acoID,
				ACOID:                  uuid.NewUUID(),
				ResourceTypes:          tt.resourceTypes,
				Since:                  tt.expSince,
				RequestType:            tt.RequestType,
				CreationTime:           tt.expTxTime,
				BFDPath:                basePath,
				ComplexDataRequestType: tt.ComplexRequestType,
			}

			repository := &models.MockRepository{}
			repository.On("GetACOByCMSID", testUtils.CtxMatcher, args.CMSID).
				Return(&models.ACO{UUID: args.ACOID, TerminationDetails: tt.terminationDetails}, nil)
			repository.On("GetCCLFFileByID", testUtils.CtxMatcher, mock.Anything).Return(getCCLFFile(1, false, false), nil)
			repository.On("GetSuppressedMBIs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(nil, nil)
			repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(tt.expBenes, nil)
			// use benes1 as the "old" benes. Allows us to verify the since parameter is populated as expected
			repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, mock.Anything).Return(benes1MBI, nil)

			cfg := &Config{
				CutoffDuration:          time.Hour,
				SuppressionLookbackDays: 0,
				RunoutConfig: RunoutConfig{
					CutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
			}
			serviceInstance := NewService(repository, cfg, basePath)
			serviceInstance.(*service).acoConfigs = acoCfgs
			ctx := context.Background()
			queJobs, err := serviceInstance.GetQueJobs(context.WithValue(ctx, middleware.CtxTransactionKey, uuid.New()), args)
			assert.NoError(t, err)
			// map tuple of resourceType:beneID
			benesInJob := make(map[string]map[string]struct{})
			for _, qj := range queJobs {
				assert.True(t, tt.expClaimsWindow.LowerBound.Equal(qj.ClaimsWindow.LowerBound),
					"Lower bounds should equal. Have %s. Want %s", qj.ClaimsWindow.LowerBound, tt.expClaimsWindow.LowerBound)
				assert.True(t, tt.expClaimsWindow.UpperBound.Equal(qj.ClaimsWindow.UpperBound),
					"Upper bounds should equal. Have %s. Want %s", qj.ClaimsWindow.UpperBound, tt.expClaimsWindow.UpperBound)

				assert.Equal(t, tt.expTxTime, qj.TransactionTime)

				subMap := benesInJob[qj.ResourceType]
				if subMap == nil {
					subMap = make(map[string]struct{})
					benesInJob[qj.ResourceType] = subMap
				}

				// Need to see if the bene is considered "new" or not. If the bene
				// is new, we should not provide a since parameter (need full history)
				var expectedTime time.Time
				if !tt.expSince.IsZero() {
					var hasNewBene bool
					for _, beneID := range qj.BeneficiaryIDs {
						if _, ok := benes1ID[beneID]; !ok {
							hasNewBene = true
							break
						}
					}
					if !hasNewBene {
						expectedTime = tt.expSince
					}
				}
				if expectedTime.IsZero() {
					assert.Empty(t, qj.Since)
				} else {
					assert.Equal(t, fmt.Sprintf("gt%s", expectedTime.Format(time.RFC3339Nano)), qj.Since)
				}

				for _, beneID := range qj.BeneficiaryIDs {
					subMap[beneID] = struct{}{}
				}

				assert.Equal(t, basePath, qj.BBBasePath)
			}

			for _, resourceType := range tt.resourceTypes {
				subMap := benesInJob[resourceType]
				assert.NotNil(t, subMap)
				for _, bene := range tt.expBenes {
					assert.Contains(t, subMap, strconv.FormatUint(uint64(bene.ID), 10))
				}
			}
		})
	}
}

func (s *ServiceTestSuite) TestCancelJob_Integration() {
	ctx := context.Background()
	synthErr := fmt.Errorf("Synthetic error for testing.")
	tests := []struct {
		status           models.JobStatus
		cancellableJobID uint
		resultJobID      uint
		getJobError      error
		updateJobError   error
	}{
		{models.JobStatusPending, 123456, 123456, nil, nil},
		{models.JobStatusInProgress, 123456, 123456, nil, nil},
		{models.JobStatusFailed, 123456, 0, nil, nil},
		{models.JobStatusExpired, 123456, 0, nil, nil},
		{models.JobStatusArchived, 123456, 0, nil, nil},
		{models.JobStatusCompleted, 123456, 0, nil, nil},
		{models.JobStatusCancelled, 123456, 0, nil, nil},
		{models.JobStatusFailedExpired, 123456, 0, nil, nil},
		{models.JobStatusInProgress, 123456, 123456, synthErr, nil}, // error occurred on GetJobByID
		{models.JobStatusInProgress, 123456, 123456, nil, synthErr}, // error occurred on UpdateJob
	}

	for _, tt := range tests {
		s.T().Run(string(tt.status), func(t *testing.T) {
			repository := &models.MockRepository{}
			repository.On("GetJobByID", testUtils.CtxMatcher, mock.Anything).Return(&models.Job{Status: tt.status}, tt.getJobError)
			repository.On("UpdateJob", testUtils.CtxMatcher, mock.Anything).Return(tt.updateJobError)
			s := &service{}
			s.repository = repository
			cancelledJobID, err := s.CancelJob(ctx, tt.cancellableJobID)
			if err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, cancelledJobID, tt.resultJobID)
			}
		})
	}
}

func (s *ServiceTestSuite) TestGetJobPriority_Integration() {
	const (
		defaultACOID  = "Some ACO"
		priorityACOID = "Priority ACO"
	)

	tests := []struct {
		name         string
		acoID        string
		resourceType string
		expSince     string
		RequestType  constants.DataRequestType
	}{
		{"Patient with Since", defaultACOID, "Patient", constants.TestSomeTime, constants.DefaultRequest},
		{"Patient without Since", defaultACOID, "Patient", "", constants.DefaultRequest},
		{"Patient Runout", defaultACOID, "Patient", constants.TestSomeTime, constants.Runout},
		{"Patient with Historic Benes", defaultACOID, "Patient", "", constants.RetrieveNewBeneHistData},
		{"Priority ACO", priorityACOID, "Patient", constants.TestSomeTime, constants.DefaultRequest},
		{"Group with Since", defaultACOID, "Coverage", constants.TestSomeTime, constants.DefaultRequest},
		{"Group without Since", defaultACOID, "Coverage", "", constants.DefaultRequest},
		{"EOB with Since", defaultACOID, "ExplanationOfBenefit", constants.TestSomeTime, constants.DefaultRequest},
		{"EOB without Since", defaultACOID, "ExplanationOfBenefit", "", constants.DefaultRequest},
		{"EOB with Historic Benes", defaultACOID, "ExplanationOfBenefit", "", constants.RetrieveNewBeneHistData},
	}

	svc := &service{}
	conf.SetEnv(s.T(), "PRIORITY_ACO_REG_EX", priorityACOID)

	for _, tt := range tests {
		expectedPriority := int16(4)

		s.T().Run(string(tt.name), func(t *testing.T) {
			if isPriorityACO(tt.acoID) {
				expectedPriority = 1
			} else if tt.resourceType == "Patient" || tt.resourceType == "Coverage" {
				expectedPriority = 2
			} else if len(tt.expSince) > 0 || tt.RequestType == constants.RetrieveNewBeneHistData {
				expectedPriority = 3
			}

			sinceParam := (len(tt.expSince) > 0) || tt.RequestType == constants.RetrieveNewBeneHistData
			jobPriority := svc.GetJobPriority(tt.acoID, tt.resourceType, sinceParam)

			assert.Equal(t, expectedPriority, jobPriority)
		})
	}
}

func (s *ServiceTestSuite) TestGetJobs_Integration() {
	repository := &models.MockRepository{}
	repository.On("GetJobs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(getJobs(1), nil)

	serviceInstance := NewService(repository, &Config{}, "").(*service)

	jobs, err := serviceInstance.GetJobs(context.Background(), uuid.NewUUID(), models.JobStatusCompleted)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, len(jobs))
	assert.Equal(s.T(), uint(1), jobs[0].ID)
}

func (s *ServiceTestSuite) TestGetJobsNotFound_Integration() {
	repository := &models.MockRepository{}
	repository.On("GetJobs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(nil, nil)

	serviceInstance := NewService(repository, &Config{}, "").(*service)

	acoID := uuid.NewUUID()
	jobs, err := serviceInstance.GetJobs(context.Background(), acoID, models.JobStatusCompleted)
	assert.Nil(s.T(), jobs)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), acoID, err.(JobsNotFoundError).ACOID)

	statuses := []models.JobStatus{models.JobStatusCompleted}
	statuses[0] = models.JobStatusCompleted
	assert.Equal(s.T(), statuses, err.(JobsNotFoundError).StatusTypes)
}

func (s *ServiceTestSuite) TestGetLatestCCLFFile_Integration() {
	repository := &models.MockRepository{}
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getCCLFFile(1, false, false), nil)

	serviceInstance := NewService(repository, &Config{}, "").(*service)

	cclfFile, err := serviceInstance.GetLatestCCLFFile(context.Background(), "Z9999", time.Time{}, time.Time{}, models.FileTypeDefault)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), uint(1), cclfFile.ID)
}

func (s *ServiceTestSuite) TestGetLatestCCLFFileNotFound_Integration() {
	repository := &models.MockRepository{}
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	serviceInstance := NewService(repository, &Config{}, "").(*service)

	cclfFile, err := serviceInstance.GetLatestCCLFFile(context.Background(), "Z9999", time.Time{}, time.Time{}, models.FileTypeDefault)
	assert.Nil(s.T(), cclfFile)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), 8, err.(CCLFNotFoundError).FileNumber)
	assert.Equal(s.T(), models.FileTypeDefault, err.(CCLFNotFoundError).FileType)
	assert.Equal(s.T(), "Z9999", err.(CCLFNotFoundError).CMSID)
	assert.Equal(s.T(), time.Time{}, err.(CCLFNotFoundError).CutoffTime)
}

func (s *ServiceTestSuite) TestGetACOConfigForID_Integration() {
	repository := &models.MockRepository{}

	specificACOPattern, _ := regexp.Compile(`A9999`)

	validACOPattern, _ := regexp.Compile(`^A\d{4}`)

	validACO := ACOConfig{
		Model:      "Model A",
		patternExp: validACOPattern,
	}

	specificACO := ACOConfig{
		Model:      "Model A9999",
		patternExp: specificACOPattern,
	}

	cfg := &Config{
		ACOConfigs: []ACOConfig{specificACO, validACO},
	}

	service := NewService(repository, cfg, "")

	tests := []struct {
		name           string
		cmsID          string
		expectedConfig *ACOConfig
		expectedOk     bool
	}{
		{
			"Valid CMSID",
			"A0000",
			&validACO,
			true,
		},
		{
			"Specific CMSID",
			"A9999",
			&specificACO,
			true,
		},
		{
			"Invalid CMSID",
			"B0000",
			nil,
			false,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			actualConfig, actualOk := service.GetACOConfigForID(tt.cmsID)
			assert.Equal(t, tt.expectedConfig, actualConfig)
			assert.Equal(t, tt.expectedOk, actualOk)
		})
	}
}

func (s *ServiceTestSuite) TestGetCutoffTime() {
	tests := []struct {
		name            string
		reqType         constants.DataRequestType
		since           time.Time
		timeConstraints TimeConstraints
		fileType        models.CCLFFileType
		serviceConfig   Config
		expectedTime    time.Time
		expectedType    string
	}{
		{
			"GetExistingBenes, no match any blocks, should return null time",
			constants.DefaultRequest,
			time.Time{},
			TimeConstraints{AttributionDate: time.Now()},
			models.FileTypeDefault,
			Config{},
			time.Time{},
			constants.GetExistingBenes,
		},
		{
			"GetExistingBenes, AttrDate, FileTypeDefault, should set cutoffTime to ~cfg.CutoffDuration",
			constants.DefaultRequest,
			time.Time{},
			TimeConstraints{},
			models.FileTypeDefault,
			Config{CutoffDuration: (time.Hour * 24)},
			time.Now().Add((time.Hour * -24)),
			constants.GetExistingBenes,
		},
		{
			"Runout, AttrDate, FileTypeDefault, should set cutoffTime to ~cfg.CutoffDuration",
			constants.Runout,
			time.Time{},
			TimeConstraints{},
			models.FileTypeDefault,
			Config{CutoffDuration: (time.Hour * 24)},
			time.Now().Add((time.Hour * -24)),
			constants.GetExistingBenes,
		},
		{
			"RetrieveNewBeneHistData, but since after attr date, should set as GetExistingBenes, null cutoff",
			constants.RetrieveNewBeneHistData,
			time.Now(),
			TimeConstraints{AttributionDate: time.Now().Add(time.Hour * -1)},
			models.FileTypeDefault,
			Config{CutoffDuration: (time.Hour * 24)},
			time.Time{},
			constants.GetExistingBenes,
		},
		{
			"GetExistingBenes, AttrDate, FileTypeRunout should set cutoffTime to ~cfg.Runout.CutoffDuration",
			constants.DefaultRequest,
			time.Time{},
			TimeConstraints{},
			models.FileTypeRunout,
			Config{RunoutConfig: RunoutConfig{CutoffDuration: (time.Hour * 24)}},
			time.Now().Add((time.Hour * -24)),
			constants.GetExistingBenes,
		},
		{
			"RetrieveNewBeneHistData, Cutoff set, should set cutoff time to config",
			constants.RetrieveNewBeneHistData,
			time.Time{},
			TimeConstraints{},
			models.FileTypeDefault,
			Config{CutoffDuration: (time.Hour * 24)},
			time.Now().Add(time.Hour * -24),
			constants.GetNewAndExistingBenes,
		},
		{
			"RetrieveNewBeneHistData, Cutoff set, AttrDate set, should return null time",
			constants.RetrieveNewBeneHistData,
			time.Time{},
			TimeConstraints{AttributionDate: time.Now()},
			models.FileTypeDefault,
			Config{CutoffDuration: (time.Hour * 24)},
			time.Time{},
			constants.GetNewAndExistingBenes,
		},
		{
			"RetrieveNewBeneHistData, should return null time",
			constants.RetrieveNewBeneHistData,
			time.Time{},
			TimeConstraints{},
			models.FileTypeDefault,
			Config{},
			time.Time{},
			constants.GetNewAndExistingBenes,
		},

		{
			"RetrieveNewBeneHistData, AttrDate set, should return null time",
			constants.RetrieveNewBeneHistData,
			time.Time{},
			TimeConstraints{AttributionDate: time.Now()},
			models.FileTypeDefault,
			Config{},
			time.Time{},
			constants.GetNewAndExistingBenes,
		},
	}

	ctx := context.Background()
	repository := &models.MockRepository{}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			service := NewService(repository, &tt.serviceConfig, "")

			actualTime, actualType := service.GetCutoffTime(ctx, tt.reqType, tt.since, tt.timeConstraints, tt.fileType)

			assert.WithinDuration(t, tt.expectedTime, actualTime, (time.Second * 10))
			assert.Equal(t, tt.expectedType, actualType)
		})
	}
}

func (s *ServiceTestSuite) TestFindOldCCLFFile() {
	now := time.Now()
	dayOld := now.Add(time.Hour * -24)
	dayOldOneSec := now.Add(time.Hour * -24).Add(time.Second * -1)

	tests := []struct {
		name          string
		cmsID         string
		since         time.Time
		cclfTimestamp time.Time
		expectedID    uint
		expectedErr   error
	}{
		{
			"No since",
			"A0001",
			time.Time{},
			time.Time{},
			uint(1),
			nil,
		},
		{
			"Since more recent than new CCLFFile timestamp",
			"A0002",
			now,
			dayOld,
			uint(2),
			nil,
		},
		{
			"Since older than CCLFFile timestamp",
			"A0003",
			dayOld,
			now,
			uint(3),
			nil,
		},
		{
			"SQL error",
			"A0004",
			time.Time{},
			time.Time{},
			uint(0),
			errors.New("db error"),
		},
		{
			"No CCLF found",
			"A0005",
			time.Time{},
			time.Time{},
			uint(0),
			CCLFNotFoundError{8, "A0005", 0, time.Time{}},
		},
	}

	ctx := context.Background()
	repository := &models.MockRepository{}

	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, "A0001", mock.Anything, mock.Anything, time.Time{}, time.Time{}, mock.Anything).
		Return(&models.CCLFFile{ID: 1}, nil)
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, "A0002", mock.Anything, mock.Anything, time.Time{}, dayOldOneSec, mock.Anything).
		Return(&models.CCLFFile{ID: 2}, nil)
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, "A0003", mock.Anything, mock.Anything, time.Time{}, dayOld, mock.Anything).
		Return(&models.CCLFFile{ID: 3}, nil)
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, "A0004", mock.Anything, mock.Anything, time.Time{}, time.Time{}, mock.Anything).
		Return(nil, errors.New("db error"))
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, "A0005", mock.Anything, mock.Anything, time.Time{}, time.Time{}, mock.Anything).
		Return(nil, nil)

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			service := NewService(repository, &Config{}, "")

			id, err := service.FindOldCCLFFile(ctx, tt.cmsID, tt.since, tt.cclfTimestamp)

			assert.Equal(t, tt.expectedID, id)
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

	repository.AssertExpectations(s.T())
}

type ServiceTestSuiteWithDatabase struct {
	suite.Suite
	priorityACOsEnvVar string
	repository         *postgres.Repository
	db                 *sql.DB
}

func TestServiceTestSuiteWithDatabase(t *testing.T) {
	suite.Run(t, new(ServiceTestSuiteWithDatabase))
}

func (s *ServiceTestSuiteWithDatabase) SetupSuite() {
	s.db, _ = databasetest.CreateDatabase(s.T(), "../../db/migrations/bcda/", true)
	s.repository = postgres.NewRepository(s.db)
}

func (s *ServiceTestSuiteWithDatabase) SetupTest() {
	s.priorityACOsEnvVar = conf.GetEnv("PRIORITY_ACO_REG_EX")
}

func (s *ServiceTestSuiteWithDatabase) TearDownTest() {
	conf.SetEnv(s.T(), "PRIORITY_ACO_REG_EX", s.priorityACOsEnvVar)
}

// suppressions.yml
// MBI00000001: (opted out) opted out 1 day ago and has an older record with no preference
// MBI00000002: (opted out) opted out 10 days ago and opted in 30 days ago
// MBI00000003: (opted out) has no previous records and opted out 1 day ago
// MBI00000004: has a single record with no opt out preference ("")
// MBI00000005: has no previous record but has opted in
// MBI00000006: has previous opt out records and opted in
// MBI00000007: has no opt out records
// MBIs 1,2,3 should be suppressed for ACOs that have the ignore_suppressions set in config
func (s *ServiceTestSuiteWithDatabase) TestGetBenesByID_Integration() {
	cfg, err := LoadConfig()
	if err != nil {
		s.T().Fatal(err)
	}

	service := service{
		repository: s.repository,
		sp: suppressionParameters{
			includeSuppressedBeneficiaries: false,
			lookbackDays:                   60,
		},
		acoConfigs: cfg.ACOConfigs,
	}

	test_cases := []struct {
		name      string
		cmsID     string
		beneCount int
		mbis      []string
	}{
		{"ACO config ignore suppressions is true", "CT000001", 7, []string{"MBI00000001", "MBI00000002", "MBI00000003", "MBI00000004", "MBI00000005", "MBI00000006", "MBI00000007"}},
		{"ACO config ignore suppressions is false implicit", "A0001", 4, []string{"MBI00000004", "MBI00000005", "MBI00000006", "MBI00000007"}},
		{"ACO config ignore suppressions is false explicit", "TEST001", 4, []string{"MBI00000004", "MBI00000005", "MBI00000006", "MBI00000007"}},
	}

	for _, test := range test_cases {
		s.T().Run(test.name, func(t *testing.T) {
			tf, err := testfixtures.New(
				testfixtures.Database(s.db),
				testfixtures.Dialect("postgres"),
				testfixtures.Directory("fixtures"))
			if err != nil {
				assert.FailNowf(s.T(), "Failed to setup test fixtures", err.Error())
			}
			if err := tf.Load(); err != nil {
				assert.FailNowf(s.T(), "Failed to load test fixtures", err.Error())
			}

			rc := worker_types.PrepareJobArgs{
				CMSID: test.cmsID,
			}
			actualBeneCount, err := service.getBenesByFileID(context.Background(), 1, rc)
			if err != nil {
				s.T().Fatal(err)
			}
			assert.Equal(t, test.beneCount, len(actualBeneCount))
			for i := 0; i < len(actualBeneCount); i++ {
				assert.Equal(t, test.mbis[i], actualBeneCount[i].MBI)
			}
		})
	}

}

func (s *ServiceTestSuiteWithDatabase) TestGetNewAndExistingBeneficiaries_RecentSinceParameterDatabase_Integration() {
	db := database.GetConnection()
	acoID := "A0005"

	// Test Setup
	testSetup := func(t *testing.T, populateBenes bool) (uint, uint, []string, func()) {
		postgrestest.DeleteCCLFFilesByCMSID(t, db, "A0005")

		performanceYear := time.Now().Year() % 100
		cclfFileOld := &models.CCLFFile{CCLFNum: 8, ACOCMSID: acoID, Timestamp: time.Now().Add(-48 * time.Hour), PerformanceYear: performanceYear, Name: "T.BCD.A0005.ZC8Y23.D231119.T1000009", ImportStatus: constants.ImportComplete}
		cclfFileNew := &models.CCLFFile{CCLFNum: 8, ACOCMSID: acoID, Timestamp: time.Now().Add(-24 * time.Hour), PerformanceYear: performanceYear, Name: "T.BCD.A0005.ZC8Y23.D231120.T1000009", ImportStatus: constants.ImportComplete}
		postgrestest.CreateCCLFFile(t, db, cclfFileOld)
		postgrestest.CreateCCLFFile(t, db, cclfFileNew)

		if populateBenes {
			bene1OldRecord := &models.CCLFBeneficiary{FileID: cclfFileOld.ID, MBI: testUtils.RandomMBI(t), BlueButtonID: testUtils.RandomHexID()}
			bene1NewRecord := &models.CCLFBeneficiary{FileID: cclfFileNew.ID, MBI: bene1OldRecord.MBI, BlueButtonID: testUtils.RandomHexID()}
			bene2NewRecord := &models.CCLFBeneficiary{FileID: cclfFileNew.ID, MBI: testUtils.RandomMBI(t), BlueButtonID: testUtils.RandomHexID()}

			postgrestest.CreateCCLFBeneficiary(t, db, bene1OldRecord)
			postgrestest.CreateCCLFBeneficiary(t, db, bene1NewRecord)
			postgrestest.CreateCCLFBeneficiary(t, db, bene2NewRecord)
			return cclfFileNew.ID, cclfFileOld.ID, []string{bene1OldRecord.MBI, bene2NewRecord.MBI}, func() { postgrestest.DeleteCCLFFilesByCMSID(t, db, "A0005") }
		} else {
			return 0, 0, []string{}, func() { postgrestest.DeleteCCLFFilesByCMSID(t, db, "A0005") }
		}
	}

	tests := []struct {
		name                  string
		sinceOffset           time.Duration
		expectedOldMBIIndexes []int
		expectedNewMBIIndexes []int
		populateBenes         bool
	}{
		{
			"BetweenTimestampAndCreatedAt",
			-12,
			[]int{0},
			[]int{1},
			true,
		},
		{
			"LaterThanCreatedAt",
			1,
			[]int{0, 1},
			[]int{},
			true,
		},
		{
			"LaterThanCreatedAtNoBenes",
			1,
			[]int{},
			[]int{},
			false,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			newCCLFID, oldCCLFID, generatedMbis, cleanup := testSetup(t, tt.populateBenes)
			defer cleanup()

			acoConfigs, _ := LoadConfig()

			cfg := &Config{
				CutoffDuration:          -50 * time.Hour,
				SuppressionLookbackDays: int(30),
				RunoutConfig: RunoutConfig{
					CutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
				ACOConfigs: acoConfigs.ACOConfigs,
			}

			since := time.Now().Add(tt.sinceOffset * time.Hour)

			repository := postgres.NewRepository(db)
			serviceInstance := NewService(repository, cfg, "").(*service)
			newBenes, oldBenes, err := serviceInstance.getNewAndExistingBeneficiaries(context.Background(),
				worker_types.PrepareJobArgs{
					CMSID:         acoID,
					Since:         since,
					CCLFFileNewID: newCCLFID,
					CCLFFileOldID: oldCCLFID,
				},
			)

			// Assert
			if !tt.populateBenes {
				assert.ErrorContains(err, "no CCLF8 file found for cmsID A0005")
			} else {
				assert.NoError(err)
				assert.Len(oldBenes, len(tt.expectedOldMBIIndexes))
				assert.Len(newBenes, len(tt.expectedNewMBIIndexes))

				contains := func(arr []*models.CCLFBeneficiary, mbi string) bool {
					for _, bene := range arr {
						if bene.MBI == mbi {
							return true
						}
					}
					return false
				}

				for _, mbiIdx := range tt.expectedOldMBIIndexes {
					assert.True(contains(oldBenes, generatedMbis[mbiIdx]), "MBI %s should be found in old MBI map %v", generatedMbis[mbiIdx], oldBenes)
				}

				for _, mbiIdx := range tt.expectedNewMBIIndexes {
					assert.True(contains(newBenes, generatedMbis[mbiIdx]), "MBI %s should be found in new MBI map %v", generatedMbis[mbiIdx], newBenes)
				}
			}
		})
	}
}

func TestGetBenesByFileID_Fail_NoACOConfig(t *testing.T) {
	args := worker_types.PrepareJobArgs{
		CMSID: "A0000",
	}
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: []ACOConfig{},
	}

	repository := models.NewMockRepository(t)
	ctx := context.Background()
	serviceInstance := NewService(repository, cfg, "").(*service)
	benes, err := serviceInstance.getBenesByFileID(ctx, uint(1), args)
	assert.Nil(t, benes)
	assert.ErrorContains(t, err, "failed to load or match ACO config (or potentially no ACO Configs set), CMS ID:")
}

func TestGetBenesByFileID_Fail_ACOConfigMismatch(t *testing.T) {
	args := worker_types.PrepareJobArgs{
		CMSID: "zxy00",
	}
	acoConfigs, err := LoadConfig()
	assert.Nil(t, err)
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: acoConfigs.ACOConfigs,
	}

	repository := models.NewMockRepository(t)
	ctx := context.Background()
	serviceInstance := NewService(repository, cfg, "").(*service)
	benes, err := serviceInstance.getBenesByFileID(ctx, uint(1), args)
	assert.Nil(t, benes)
	assert.ErrorContains(t, err, "failed to load or match ACO config (or potentially no ACO Configs set), CMS ID:")
}

func TestCreateQueueJobs_Fail_NoACOConfig(t *testing.T) {
	args := worker_types.PrepareJobArgs{
		CMSID: "A0000",
	}
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: []ACOConfig{},
	}

	repository := models.NewMockRepository(t)
	ctx := context.Background()
	serviceInstance := NewService(repository, cfg, "").(*service)
	jobArgs, err := serviceInstance.createQueueJobs(ctx, args, time.Now(), nil)
	assert.Nil(t, jobArgs)
	assert.ErrorContains(t, err, "failed to load or match ACO config (or potentially no ACO Configs set), CMS ID:")
}

func TestCreateQueueJobs_Fail_ACOConfigMismatch(t *testing.T) {
	args := worker_types.PrepareJobArgs{
		CMSID: "zxy00",
	}
	acoConfigs, err := LoadConfig()
	assert.Nil(t, err)
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: acoConfigs.ACOConfigs,
	}

	repository := models.NewMockRepository(t)
	ctx := context.Background()
	serviceInstance := NewService(repository, cfg, "").(*service)
	jobArgs, err := serviceInstance.createQueueJobs(ctx, args, time.Now(), nil)
	assert.Nil(t, jobArgs)
	assert.ErrorContains(t, err, "failed to load or match ACO config (or potentially no ACO Configs set), CMS ID:")
}

func TestSetClaimsDate_Runout(t *testing.T) {
	pArgs := worker_types.PrepareJobArgs{
		CMSID:       "A0000",
		RequestType: constants.Runout,
	}
	jArgs := worker_types.JobEnqueueArgs{
		CMSID: "A0000",
	}
	acoConfigs, err := LoadConfig()
	assert.Nil(t, err)
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: acoConfigs.ACOConfigs,
	}

	repository := models.NewMockRepository(t)
	serviceInstance := NewService(repository, cfg, "").(*service)
	ok := serviceInstance.setClaimsDate(&jArgs, pArgs)
	assert.True(t, ok)

	acoCfg, ok := serviceInstance.GetACOConfigForID("A0000")
	assert.True(t, ok)
	assert.Equal(t, jArgs.ClaimsWindow.UpperBound, defaultRunoutClaimThru)
	assert.Equal(t, jArgs.ClaimsWindow.LowerBound, acoCfg.LookbackTime())
}

func TestSetClaimsDate_ClaimsDateSet(t *testing.T) {
	now := time.Now()
	pArgs := worker_types.PrepareJobArgs{
		CMSID:       "A0000",
		RequestType: constants.DefaultRequest,
		ClaimsDate:  now,
	}
	jArgs := worker_types.JobEnqueueArgs{
		CMSID: "A0000",
	}
	acoConfigs, err := LoadConfig()
	assert.Nil(t, err)
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: acoConfigs.ACOConfigs,
	}

	repository := models.NewMockRepository(t)
	serviceInstance := NewService(repository, cfg, "").(*service)
	ok := serviceInstance.setClaimsDate(&jArgs, pArgs)
	assert.True(t, ok)

	acoCfg, ok := serviceInstance.GetACOConfigForID("A0000")
	assert.True(t, ok)
	assert.Equal(t, jArgs.ClaimsWindow.UpperBound, now)
	assert.Equal(t, jArgs.ClaimsWindow.LowerBound, acoCfg.LookbackTime())
}

func TestSetClaimsDate_Fail_NoACOConfig(t *testing.T) {
	pArgs := worker_types.PrepareJobArgs{
		CMSID: "A0000",
	}
	jArgs := worker_types.JobEnqueueArgs{
		CMSID: "A0000",
	}
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: []ACOConfig{},
	}

	repository := models.NewMockRepository(t)
	serviceInstance := NewService(repository, cfg, "").(*service)
	ok := serviceInstance.setClaimsDate(&jArgs, pArgs)
	assert.False(t, ok)
}

func TestSetClaimsDate_Fail_ACOConfigMismatch(t *testing.T) {
	pArgs := worker_types.PrepareJobArgs{
		CMSID: "zxy00",
	}
	jArgs := worker_types.JobEnqueueArgs{
		CMSID: "zxy00",
	}
	acoConfigs, err := LoadConfig()
	assert.Nil(t, err)
	cfg := &Config{
		CutoffDuration:          -50 * time.Hour,
		SuppressionLookbackDays: int(30),
		RunoutConfig: RunoutConfig{
			CutoffDuration: defaultRunoutCutoff,
			claimThru:      defaultRunoutClaimThru,
		},
		ACOConfigs: acoConfigs.ACOConfigs,
	}

	repository := models.NewMockRepository(t)
	serviceInstance := NewService(repository, cfg, "").(*service)
	ok := serviceInstance.setClaimsDate(&jArgs, pArgs)
	assert.False(t, ok)
}

func getCCLFFile(id uint, isRunout bool, forceIncorrect bool) *models.CCLFFile {
	performanceYear := time.Now().Year() % 100
	if isRunout {
		performanceYear -= 1
	}
	if forceIncorrect {
		performanceYear -= 10
	}
	return &models.CCLFFile{
		ID:              id,
		PerformanceYear: performanceYear,
		CreatedAt:       time.Now(),
	}
}

func getCCLFBeneficiary(id uint, mbi string) *models.CCLFBeneficiary {
	return &models.CCLFBeneficiary{
		ID:  id,
		MBI: mbi,
	}
}

func getJobs(id uint) []*models.Job {
	return []*models.Job{
		{
			ID: id,
		},
	}
}
