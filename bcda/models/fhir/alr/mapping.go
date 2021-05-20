package alr

import (
	"regexp"

	"github.com/CMSgov/bcda-app/bcda/models"
)

// This part of the package is responsible for divvying up the the K:V pair in
// models.Alr into respective categories depicted in kvMap struct.
// Note, a models.Alr represents a single row from a dataframe.

type kvMap struct {
	assignment   map[string]string
	enrollment   map[string]string
	exclusion    map[string]string
	riskFlag     map[string]string
	riskScore    map[string]string
	groupPattern map[string]string
}

// These are regexp patterns used for mapping fields from ALR data
var assignmentPattern, enrollmentPattern, exclusionPattern, riskFlagsPattern,
	riskScoresPattern, groupPattern *regexp.Regexp

func init() {
	assignmentPattern = regexp.MustCompile(`^(IN_VA_MAX)|(CBA_FLAG)|(ASSIGNMENT_TYPE)` +
		`|(ASSIGNED_BEFORE)|(ASG_STATUS)$`)

	enrollmentPattern = regexp.MustCompile(`^EnrollFlag\d+$`)

	exclusionPattern = regexp.MustCompile(`^(EXCLUDED)|(DECEASED_EXCLUDED)|` +
		`(MISSING_ID_EXCLUDED)|(PART_A_B_ONLY_EXCLUDED)|` +
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)$`)

	riskFlagsPattern = regexp.MustCompile(`^(HCC_version)|(HCC_COL_\d+)$`)

	riskScoresPattern = regexp.MustCompile(`^(BENE_RSK_R_SCRE_\d{2,})|(((ESRD)|` +
		`(DIS)|(AGDU)|(AGND)|(DEM_ESRD)|(DEM_DIS)|(DEM_AGDU)|(DEM_AGND))_SCORE)$`)

	groupPattern = regexp.MustCompile(`^(IN_VA_MAX)|(CBA_FLAG)|(ASSIGNMENT_TYPE)` +
		`|(ASSIGNED_BEFORE)|(ASG_STATUS)|(EXCLUDED)|(DECEASED_EXCLUDED)|` +
		`(MISSING_ID_EXCLUDED)|(PART_A_B_ONLY_EXCLUDED)|` +
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)$`)
}

// keyValueMapper take the K:V pair from models.Alr and puts them into one of
// five maps in the kvMap struct. kvMap struct is then used to generate FHIR data.
func keyValueMapper(alr *models.Alr) kvMap {

	assignmentFields := make(map[string]string)
	enrollmentFields := make(map[string]string)
	exclusionFields := make(map[string]string)
	riskFlagFields := make(map[string]string)
	riskScoreFields := make(map[string]string)

	for k, v := range alr.KeyValue {
		if assignmentPattern.MatchString(k) {
			assignmentFields[k] = v
		} else if enrollmentPattern.MatchString(k) {
			enrollmentFields[k] = v
		} else if exclusionPattern.MatchString(k) {
			exclusionFields[k] = v
		} else if riskFlagsPattern.MatchString(k) {
			riskFlagFields[k] = v
		} else if riskScoresPattern.MatchString(k) {
			riskScoreFields[k] = v
		}
	}

	return kvMap{
		assignment: assignmentFields,
		enrollment: enrollmentFields,
		exclusion:  exclusionFields,
		riskFlag:   riskFlagFields,
		riskScore:  riskFlagFields,
	}
}
