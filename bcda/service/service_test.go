package service

import (
	context "context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/conf"
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
		{"DefaultEOB", "ExplanationOfBenefit", 200, clearer},
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

type ServiceTestSuite struct {
	suite.Suite
	priorityACOsEnvVar string
}

// Run all test suite tets
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

func (s *ServiceTestSuite) SetupTest() {
	s.priorityACOsEnvVar = conf.GetEnv("PRIORITY_ACO_REG_EX")
}

func (s *ServiceTestSuite) TearDownTest() {
	conf.SetEnv(s.T(), "PRIORITY_ACO_REG_EX", s.priorityACOsEnvVar)
}

func (s *ServiceTestSuite) TestIncludeSuppressedBeneficiaries() {
	conditions := RequestConditions{
		CMSID:    "cmsID",
		Since:    time.Now(),
		fileType: models.FileTypeDefault,
	}
	tests := []struct {
		name          string
		cclfFileNew   *models.CCLFFile
		cclfFileOld   *models.CCLFFile
		funcUnderTest func(s *service) error
	}{
		{
			"GetNewAndExistingBeneficiaries",
			getCCLFFile(1),
			getCCLFFile(2),
			func(serv *service) error {
				_, _, err := serv.getNewAndExistingBeneficiaries(context.Background(), conditions)
				return err
			},
		},
		{
			"GetBeneficiaries",
			getCCLFFile(3),
			nil,
			func(serv *service) error {
				_, err := serv.getBeneficiaries(context.Background(), conditions)
				return err
			},
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(8)
			sp := suppressionParameters{true, lookbackDays}
			repository := &models.MockRepository{}
			repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(timeIsSetMatcher), time.Time{}, models.FileTypeDefault).Return(tt.cclfFileNew, nil)
			repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, time.Time{}, mock.MatchedBy(timeIsSetMatcher), models.FileTypeDefault).Return(tt.cclfFileOld, nil)
			if tt.cclfFileOld != nil {
				repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, tt.cclfFileOld.ID).Return([]string{"1", "2", "3"}, nil)
			}

			var suppressedMBIs []string
			repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, tt.cclfFileNew.ID, suppressedMBIs).Return([]*models.CCLFBeneficiary{getCCLFBeneficiary(1, "1")}, nil)
			serviceInstance := &service{repository: repository, sp: sp, stdCutoffDuration: 1 * time.Hour}

			err := tt.funcUnderTest(serviceInstance)
			assert.NoError(t, err)

			repository.AssertNotCalled(t, "GetSuppressedMBIs", testUtils.CtxMatcher, lookbackDays, time.Time{})
		})
	}
}

func (s *ServiceTestSuite) TestGetNewAndExistingBeneficiaries() {
	tests := []struct {
		name string

		cclfFileNew *models.CCLFFile
		cclfFileOld *models.CCLFFile

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
			repository := &models.MockRepository{}
			cutoffDuration := 1 * time.Hour
			cmsID := "cmsID"
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

			// Skip populating new benes under certain test conditions
			if tt.name != "NoBenesFoundNew" && tt.name != "NoBenesFoundNewAndOld" {
				for i := 0; i < 10; i++ {
					mbi := fmt.Sprintf("NewMBI%d", i)
					benes = append(benes, getCCLFBeneficiary(beneID, mbi))
					newMBIs[mbi] = true
					beneID++
				}
			}

			repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, cmsID, fileNum, constants.ImportComplete,
				// Verify our cutoffTime is bsed on our provided duration
				mock.MatchedBy(func(t time.Time) bool {
					// Since we're using time.Now() within the service call, we can't compare directly.
					// Make sure we're close enough.
					return time.Now().Add(-1*cutoffDuration).Sub(t) < time.Second
				}),
				time.Time{},
				models.FileTypeDefault).Return(tt.cclfFileNew, nil)
			repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, cmsID, fileNum, constants.ImportComplete, time.Time{}, since, models.FileTypeDefault).Return(tt.cclfFileOld, nil)

			if tt.cclfFileOld != nil {
				repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, tt.cclfFileOld.ID).Return(tt.oldMBIs, nil)
			}
			suppressedMBI := "suppressedMBI"
			if tt.cclfFileNew != nil {
				repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, tt.cclfFileNew.ID, []string{suppressedMBI}).Return(benes, nil)
			}
			repository.On("GetSuppressedMBIs", testUtils.CtxMatcher, lookbackDays, mockUpperBound).Return([]string{suppressedMBI}, nil)

			cfg := &Config{
				cutoffDuration:          time.Hour,
				SuppressionLookbackDays: lookbackDays,
				RunoutConfig: RunoutConfig{
					cutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
			}
			serviceInstance := NewService(repository, cfg, "").(*service)
			newBenes, oldBenes, err := serviceInstance.getNewAndExistingBeneficiaries(context.Background(),
				RequestConditions{CMSID: "cmsID", Since: since, fileType: models.FileTypeDefault})

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
		fileType    models.CCLFFileType
		cclfFile    *models.CCLFFile
		expectedErr error
	}{
		{
			"BenesReturned",
			models.FileTypeDefault,
			getCCLFFile(1),
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
			getCCLFFile(2),
			fmt.Errorf("Found 0 beneficiaries from CCLF8 file for cmsID"),
		},
		{
			"BenesReturnedRunout",
			models.FileTypeRunout,
			getCCLFFile(3),
			nil,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			lookbackDays := int(30)
			fileNum := int(8)
			repository := &models.MockRepository{}
			cutoffDuration := 1 * time.Hour
			cmsID := "cmsID"
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
			repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, cmsID, fileNum, constants.ImportComplete,
				// Verify our cutoffTime is based on our provided duration
				mock.MatchedBy(func(t time.Time) bool {
					// Since we're using time.Now() within the service call, we can't compare directly.
					// Make sure we're close enough.
					switch tt.fileType {
					case models.FileTypeDefault:
						return time.Now().Add(-1*cutoffDuration).Sub(t) < time.Second
					case models.FileTypeRunout:
						return time.Now().Add(-1*120*24*time.Hour).Sub(t) < time.Second
					default:
						return false // We do not understand this fileType
					}
				}),
				time.Time{}, tt.fileType).Return(tt.cclfFile, nil)

			suppressedMBI := "suppressedMBI"
			repository.On("GetSuppressedMBIs", testUtils.CtxMatcher, lookbackDays, mockUpperBound).Return([]string{suppressedMBI}, nil)
			if tt.cclfFile != nil {
				repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, tt.cclfFile.ID, []string{suppressedMBI}).Return(benes, nil)
			}

			cfg := &Config{
				cutoffDuration:          time.Hour,
				SuppressionLookbackDays: lookbackDays,
				RunoutConfig: RunoutConfig{
					cutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
			}
			serviceInstance := NewService(repository, cfg, "").(*service)
			benes, err := serviceInstance.getBeneficiaries(context.Background(),
				RequestConditions{CMSID: "cmsID", fileType: tt.fileType})

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

func (s *ServiceTestSuite) TestGetQueJobs() {
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

	acoCfgs := map[*regexp.Regexp]*ACOConfig{
		defaultACO.patternExp:  &defaultACO,
		lookbackACO.patternExp: &lookbackACO,
	}

	benes1, benes2 := make([]*models.CCLFBeneficiary, 10), make([]*models.CCLFBeneficiary, 20)
	allBenes := [][]*models.CCLFBeneficiary{benes1, benes2}
	for idx, b := range allBenes {
		for i := 0; i < len(b); i++ {
			id := uint(idx*10000 + i + 1)
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
		reqType            RequestType
		expSince           time.Time
		expClaimsWindow    claimsWindow
		expBenes           []*models.CCLFBeneficiary
		resourceTypes      []string
		terminationDetails *models.Termination
	}

	baseTests := []test{
		{"BasicRequest (non-Group)", defaultACOID, DefaultRequest, time.Time{}, claimsWindow{}, benes1, nil, nil},
		{"BasicRequest with Since (non-Group) ", defaultACOID, DefaultRequest, since, claimsWindow{}, benes1, nil, nil},
		{"GroupAll", defaultACOID, RetrieveNewBeneHistData, since, claimsWindow{}, append(benes1, benes2...), nil, nil},
		{"RunoutRequest", defaultACOID, Runout, time.Time{}, claimsWindow{UpperBound: defaultRunoutClaimThru}, benes1, nil, nil},
		{"RunoutRequest with Since", defaultACOID, Runout, since, claimsWindow{UpperBound: defaultRunoutClaimThru}, benes1, nil, nil},

		// Terminated ACOs: historical
		{"Since After Termination", defaultACOID, DefaultRequest, sinceAfterTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
		{"Since Before Termination", defaultACOID, DefaultRequest, sinceBeforeTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
		{"New Benes With Since After Termination", defaultACOID, RetrieveNewBeneHistData, sinceAfterTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
		{"New Benes With Since Before Termination", defaultACOID, RetrieveNewBeneHistData, sinceBeforeTermination, claimsWindow{UpperBound: terminationHistorical.ClaimsDate()}, append(benes1, benes2...), nil, terminationHistorical},
		{"TerminatedACORunout", defaultACOID, Runout, time.Time{}, claimsWindow{UpperBound: defaultRunoutClaimThru}, benes1, nil, terminationHistorical}, // Runout cutoff takes precedence over termination cutoff

		// Terminated ACOs: latest
		{"Since After Termination", defaultACOID, DefaultRequest, sinceAfterTermination, claimsWindow{}, benes1, nil, terminationLatest},
		{"Since Before Termination", defaultACOID, DefaultRequest, sinceBeforeTermination, claimsWindow{}, benes1, nil, terminationLatest},
		// should still receive full benes since Attribution is set to latest
		{"New Benes With Since After Termination", defaultACOID, RetrieveNewBeneHistData, sinceAfterTermination, claimsWindow{}, append(benes1, benes2...), nil, terminationLatest},
		{"New Benes With Since Before Termination", defaultACOID, RetrieveNewBeneHistData, sinceBeforeTermination, claimsWindow{}, append(benes1, benes2...), nil, terminationLatest},

		// ACO with lookback period
		{"ACO with lookback", lookbackACOID, DefaultRequest, time.Time{}, claimsWindow{LowerBound: lookbackACO.LookbackTime()}, benes1, nil, nil},
		{"Terminated ACO with lookback", lookbackACOID, DefaultRequest, time.Time{}, claimsWindow{LowerBound: lookbackACO.LookbackTime(), UpperBound: terminationHistorical.ClaimsDate()}, benes1, nil, terminationHistorical},
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
			conditions := RequestConditions{
				CMSID:     tt.acoID,
				ACOID:     uuid.NewUUID(),
				Resources: tt.resourceTypes,
				Since:     tt.expSince,
				ReqType:   tt.reqType,
			}

			repository := &models.MockRepository{}
			repository.On("GetACOByCMSID", testUtils.CtxMatcher, conditions.CMSID).
				Return(&models.ACO{UUID: conditions.ACOID, TerminationDetails: tt.terminationDetails}, nil)
			repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getCCLFFile(1), nil)
			repository.On("GetSuppressedMBIs", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(nil, nil)
			repository.On("GetCCLFBeneficiaries", testUtils.CtxMatcher, mock.Anything, mock.Anything).Return(tt.expBenes, nil)
			// use benes1 as the "old" benes. Allows us to verify the since parameter is populated as expected
			repository.On("GetCCLFBeneficiaryMBIs", testUtils.CtxMatcher, mock.Anything).Return(benes1MBI, nil)

			cfg := &Config{
				cutoffDuration:          time.Hour,
				SuppressionLookbackDays: 0,
				RunoutConfig: RunoutConfig{
					cutoffDuration: defaultRunoutCutoff,
					claimThru:      defaultRunoutClaimThru,
				},
			}
			serviceInstance := NewService(repository, cfg, basePath)
			serviceInstance.(*service).acoConfig = acoCfgs

			queJobs, err := serviceInstance.GetQueJobs(context.Background(), conditions)
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

func (s *ServiceTestSuite) TestGetQueJobsFailedACOLookup() {
	conditions := RequestConditions{ACOID: uuid.NewRandom(), CMSID: uuid.New()}
	repository := &models.MockRepository{}
	repository.On("GetACOByCMSID", testUtils.CtxMatcher, conditions.CMSID).
		Return(nil, context.DeadlineExceeded)
	defer repository.AssertExpectations(s.T())
	service := &service{repository: repository}
	queJobs, err := service.GetQueJobs(context.Background(), conditions)
	assert.Nil(s.T(), queJobs)
	assert.True(s.T(), errors.Is(err, context.DeadlineExceeded), "Root cause should be deadline exceeded")
}

func (s *ServiceTestSuite) TestCancelJob() {
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

func (s *ServiceTestSuite) TestGetJobPriority() {
	const (
		defaultACOID  = "Some ACO"
		priorityACOID = "Priority ACO"
	)

	tests := []struct {
		name         string
		acoID        string
		resourceType string
		expSince     string
		reqType      RequestType
	}{
		{"Patient with Since", defaultACOID, "Patient", "some time", DefaultRequest},
		{"Patient without Since", defaultACOID, "Patient", "", DefaultRequest},
		{"Patient Runout", defaultACOID, "Patient", "some time", Runout},
		{"Patient with Historic Benes", defaultACOID, "Patient", "", RetrieveNewBeneHistData},
		{"Priority ACO", priorityACOID, "Patient", "some time", DefaultRequest},
		{"Group with Since", defaultACOID, "Coverage", "some time", DefaultRequest},
		{"Group without Since", defaultACOID, "Coverage", "", DefaultRequest},
		{"EOB with Since", defaultACOID, "ExplanationOfBenefit", "some time", DefaultRequest},
		{"EOB without Since", defaultACOID, "ExplanationOfBenefit", "", DefaultRequest},
		{"EOB with Historic Benes", defaultACOID, "ExplanationOfBenefit", "", RetrieveNewBeneHistData},
	}

	svc := &service{}
	conf.SetEnv(s.T(), "PRIORITY_ACO_REG_EX", priorityACOID)

	for _, tt := range tests {
		expectedPriority := int16(100)

		s.T().Run(string(tt.name), func(t *testing.T) {
			if isPriorityACO(tt.acoID) {
				expectedPriority = 10
			} else if tt.resourceType == "Patient" || tt.resourceType == "Coverage" {
				expectedPriority = 20
			} else if len(tt.expSince) > 0 || tt.reqType == RetrieveNewBeneHistData {
				expectedPriority = 30
			}

			sinceParam := (len(tt.expSince) > 0) || tt.reqType == RetrieveNewBeneHistData
			jobPriority := svc.GetJobPriority(tt.acoID, tt.resourceType, sinceParam)

			assert.Equal(t, expectedPriority, jobPriority)
		})
	}
}

func (s *ServiceTestSuite) TestGetLatestCCLFFile() {
	repository := &models.MockRepository{}
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(getCCLFFile(1), nil)

	serviceInstance := NewService(repository, &Config{}, "").(*service)

	cclfFile, err := serviceInstance.GetLatestCCLFFile(context.Background(), "Z9999", models.FileTypeDefault)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), uint(1), cclfFile.ID)
}

func (s *ServiceTestSuite) TestGetLatestCCLFFileNotFound() {
	repository := &models.MockRepository{}
	repository.On("GetLatestCCLFFile", testUtils.CtxMatcher, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	serviceInstance := NewService(repository, &Config{}, "").(*service)

	cclfFile, err := serviceInstance.GetLatestCCLFFile(context.Background(), "Z9999", models.FileTypeDefault)
	assert.Nil(s.T(), cclfFile)
	assert.Error(s.T(), err)
	assert.Equal(s.T(), 8, err.(CCLFNotFoundError).FileNumber)
	assert.Equal(s.T(), models.FileTypeDefault, err.(CCLFNotFoundError).FileType)
	assert.Equal(s.T(), "Z9999", err.(CCLFNotFoundError).CMSID)
	assert.Equal(s.T(), time.Time{}, err.(CCLFNotFoundError).CutoffTime)
}

func (s *ServiceTestSuite) TestGetACOConfigForID() {
	repository := &models.MockRepository{}

	validACOPattern, _ := regexp.Compile(`A\d{4}`)

	validACO := ACOConfig{
		Model:      "Model A",
		patternExp: validACOPattern,
	}

	cfg := &Config{
		ACOConfigs: []ACOConfig{validACO},
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

func getCCLFFile(id uint) *models.CCLFFile {
	return &models.CCLFFile{
		ID: id,
	}
}

func getCCLFBeneficiary(id uint, mbi string) *models.CCLFBeneficiary {
	return &models.CCLFBeneficiary{
		ID:  id,
		MBI: mbi,
	}
}

func timeIsSetMatcher(t time.Time) bool {
	return !t.IsZero()
}
