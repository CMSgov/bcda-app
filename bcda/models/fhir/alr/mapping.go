package alr

import (
	"regexp"

	"github.com/CMSgov/bcda-app/bcda/models"
)

// This part of the package is responsible for divvying up the the K:V pair in
// models.Alr into respective categories depicted in kvMap struct.
// Note, a models.Alr represents a single row from a dataframe.

type kvArena struct {
	enrollment []kvPair
	riskFlag   []kvPair
	riskScore  []kvPair
	group      []kvPair
	hccVersion []kvPair
}

type kvPair struct {
	key   string
	value string
}

// These are regexp patterns used for mapping fields from ALR data
var (
	enrollmentPattern,
	riskFlagsPattern,
	riskScoresPattern,
	hccVersion,
	groupPattern *regexp.Regexp
)

func init() {
	// There should be one field in table 1-1 that tells use what HCC version
	// is used for that particular ALR
	hccVersion = regexp.MustCompile(`^HCC_version$`)

	// These are for coverage
	enrollmentPattern = regexp.MustCompile(`^EnrollFlag\d+$`)

	// These are for groups
	groupPattern = regexp.MustCompile(`^(IN_VA_MAX)|(CBA_FLAG)|(ASSIGNMENT_TYPE)` +
		`|(ASSIGNED_BEFORE)|(ASG_STATUS)|(EXCLUDED)|(DECEASED_EXCLUDED)|` +
		`(MISSING_ID_EXCLUDED)|(PART_A_B_ONLY_EXCLUDED)|` +
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)$`)

	// These are risk scores, which current go under observation
	riskFlagsPattern = regexp.MustCompile(`^HCC_COL_\d+$`)

	// These are for risk assessment
	riskScoresPattern = regexp.MustCompile(`^(BENE_RSK_R_SCRE_\d{2,})|(((ESRD)|` +
		`(DIS)|(AGDU)|(AGND)|(DEM_ESRD)|(DEM_DIS)|(DEM_AGDU)|(DEM_AGND))_SCORE)$`)
}

// keyValueMapper take the K:V pair from models.Alr and puts them into one of
// various maps in the kvArena struct. kvMap struct is then used to generate FHIR data.
func keyValueMapper(alr *models.Alr) kvArena {

	hccVersionFields := []kvPair{}
	enrollmentFields := []kvPair{}
	groupFields := []kvPair{}
	riskFlagFields := []kvPair{}
	riskScoreFields := []kvPair{}

	for k, v := range alr.KeyValue {
		if enrollmentPattern.MatchString(k) {
			enrollmentFields = append(enrollmentFields, kvPair{k, v})
		} else if riskFlagsPattern.MatchString(k) {
			riskFlagFields = append(riskFlagFields, kvPair{k, v})
		} else if riskScoresPattern.MatchString(k) {
			riskScoreFields = append(riskScoreFields, kvPair{k, v})
		} else if groupPattern.MatchString(k) {
			groupFields = append(groupFields, kvPair{k, v})
		} else if hccVersion.MatchString(k) {
			hccVersionFields = append(hccVersionFields, kvPair{k, v})
		}
	}

	return kvArena{
		enrollment:   enrollmentFields,
		riskFlag:     riskFlagFields,
		riskScore:    riskScoreFields,
		hccVersion:   hccVersionFields,
		group: groupFields,
	}
}
