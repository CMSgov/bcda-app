package utils

import (
	"regexp"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/log"
)

// This part of the package is responsible for divvying up the the K:V pair in
// models.Alr into respective categories depicted in kvMap struct.
// Note, a models.Alr represents a single row from a dataframe.

type KvArena struct {
	Enrollment    []KvPair
	RiskFlag      []KvPair
	RiskScore     []KvPair
	Group         []KvPair
	HccVersion    []KvPair
	CovidEpsisode []KvPair
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
	GroupPattern,
	CovidPattern *regexp.Regexp
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
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)|(VA_SELECTION_ONLY)|` +
		`(PLUR_R05)|(AB_R01)|(HMO_R03)|(NO_US_R02)|(MDM_R04)|(NOFND_R06)$`)

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
		"OTHER_SHARED_SAV_INIT":  "Beneficiary included in other Shared Savings Initiatives",
		"PLUR_R05":               "Beneficiary did not receive the plurality of their primary care services from the ACO",
		"AB_R01":                 "Beneficiary had at least 1 month of Part A-only or Part B-only coverage",
		"HMO_R03":                "Beneficiary had at least 1 month in a Medicare health plan",
		"NO_US_R02":              "Beneficiary does not reside in the United States",
		"MDM_R04":                "Beneficiary included in other Shared Savings Initiatives",
		"NOFND_R06":              "Beneficiary did not have a physician visit with an ACO professional or was not assigned for any other reason not listed",
	}
	// These are for Covid19 Episode of Care
	CovidPattern = regexp.MustCompile(`^((COVID19_EPISODE)|(COVID19_MONTH(0[1-9]|1[0-2]))` +
		`|(ADMISSION_DT)|(DISCHARGE_DT)|(U071)|(B9729))$`)
}

// keyValueMapper take the K:V pair from models.Alr and puts them into one of
// various maps in the kvArena struct. kvMap struct is then used to generate FHIR data.
func KeyValueMapper(alr *models.Alr) KvArena {

	hccVersionFields := []KvPair{}
	enrollmentFields := []KvPair{}
	groupFields := []KvPair{}
	riskFlagFields := []KvPair{}
	riskScoreFields := []KvPair{}
	covidFields := []KvPair{}

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
		} else if CovidPattern.MatchString(k) {
			covidFields = append(covidFields, KvPair{k, v})
		} else {
			// The k:v from ALR has not been mapped. For now, we do not include them into FHIR and log.
			// TODO: Should we include these "novel" fields in a catch all resource like observation?
			log.API.Warnf("Could not map key %s with value %s to a FHIR resource. Ignoring this field.", k, v)
		}
	}

	return KvArena{
		Enrollment:    enrollmentFields,
		RiskFlag:      riskFlagFields,
		RiskScore:     riskScoreFields,
		HccVersion:    hccVersionFields,
		Group:         groupFields,
		CovidEpsisode: covidFields,
	}
}
