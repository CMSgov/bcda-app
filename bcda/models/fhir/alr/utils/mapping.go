package utils

import (
	"regexp"

	"github.com/CMSgov/bcda-app/bcda/models"
)

// This part of the package is responsible for divvying up the the K:V pair in
// models.Alr into respective categories depicted in kvMap struct.
// Note, a models.Alr represents a single row from a dataframe.

type KvArena struct {
	Enrollment []KvPair
	RiskFlag   []KvPair
	RiskScore  []KvPair
	Group      []KvPair
	HccVersion []KvPair
}

type KvPair struct {
    Key   string
	Value string
}

// These are regexp patterns used for mapping fields from ALR data
var (
	EnrollmentPattern,
	RiskFlagsPattern,
	RiskScoresPattern,
	HccVersion,
	GroupPattern *regexp.Regexp
)

var GroupPatternDescriptions map[string]string

func init() {
	// There should be one field in table 1-1 that tells use what HCC version
	// is used for that particular ALR
	HccVersion = regexp.MustCompile(`^HCC_version$`)

	// These are for coverage
	EnrollmentPattern = regexp.MustCompile(`^EnrollFlag\d+$`)

	// These are for groups
	GroupPattern = regexp.MustCompile(`^(IN_VA_MAX)|(CBA_FLAG)|(ASSIGNMENT_TYPE)` +
		`|(ASSIGNED_BEFORE)|(ASG_STATUS)|(EXCLUDED)|(DECEASED_EXCLUDED)|` +
		`(MISSING_ID_EXCLUDED)|(PART_A_B_ONLY_EXCLUDED)|` +
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)$`)

	// These are risk scores, which current go under observation
	RiskFlagsPattern = regexp.MustCompile(`^HCC_COL_\d+$`)

	// These are for risk assessment
	RiskScoresPattern = regexp.MustCompile(`^(BENE_RSK_R_SCRE_\d{2,})|(((ESRD)|` +
		`(DIS)|(AGDU)|(AGND)|(DEM_ESRD)|(DEM_DIS)|(DEM_AGDU)|(DEM_AGND))_SCORE)$`)

	// Possibly temporary, but storing some field name from the data dictionary
	GroupPatternDescriptions = map[string]string{
		"DECEASED_EXCLUDED":      "Beneficiary had a date of death prior to the start of the performance year",
		"MISSING_ID_EXCLUDED":    "Beneficiary identifier is missing",
		"PART_A_B_ONLY_EXCLUDED": "Beneficiary had at least one month of Part A-only Or Part B-only Coverage",
		"GHP_EXCLUDED":           "Beneficiary had at least one month in a Medicare Health Plan",
		"OUTSIDE_US_EXCLUDED":    "Beneficiary does not reside in the United States",
		"OTHER_SHARED_SAV_INIT":   "Beneficiary included in other Shared Savings Initiatives",
	}
}

// keyValueMapper take the K:V pair from models.Alr and puts them into one of
// various maps in the kvArena struct. kvMap struct is then used to generate FHIR data.
func KeyValueMapper(alr *models.Alr) KvArena {

	hccVersionFields := []KvPair{}
	enrollmentFields := []KvPair{}
	groupFields := []KvPair{}
	riskFlagFields := []KvPair{}
	riskScoreFields := []KvPair{}

	for k, v := range alr.KeyValue {
		if EnrollmentPattern.MatchString(k) {
			enrollmentFields = append(enrollmentFields, KvPair{k, v})
		} else if RiskFlagsPattern.MatchString(k) {
			riskFlagFields = append(riskFlagFields, KvPair{k, v})
		} else if RiskScoresPattern.MatchString(k) {
			riskScoreFields = append(riskScoreFields, KvPair{k, v})
		} else if GroupPattern.MatchString(k) {
			groupFields = append(groupFields, KvPair{k, v})
		} else if HccVersion.MatchString(k) {
			hccVersionFields = append(hccVersionFields, KvPair{k, v})
		}
	}

	return KvArena{
		Enrollment: enrollmentFields,
		RiskFlag:   riskFlagFields,
		RiskScore:  riskScoreFields,
		HccVersion: hccVersionFields,
		Group:      groupFields,
	}
}
