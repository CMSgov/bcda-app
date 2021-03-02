package alr

import (
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

var assignment, enrollment, exclusion, riskFlags, riskScores  *regexp.Regexp

func init() {
	assignment = regexp.MustCompile(`^(IN_VA_MAX)|(CBA_FLAG)|(ASSIGNMENT_TYPE)|(ASSIGNED_BEFORE)|(ASG_STATUS)$`)
	enrollment = regexp.MustCompile(`^EnrollFlag\d+$`)
	exclusion = regexp.MustCompile(`^(EXCLUDED)|(DECEASED_EXCLUDED)|(MISSING_ID_EXCLUDED)|(PART_A_B_ONLY_EXCLUDED)|` +
	`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)$`)
	riskFlags = regexp.MustCompile(`^(HCC_version)|(HCC_COL_\d+)$`)
	riskScores = regexp.MustCompile(`^(BENE_RSK_R_SCRE_\d{2,})|(((ESRD)|(DIS)|(AGDU)|(AGND)|(DEM_ESRD)|(DEM_DIS)|(DEM_AGDU)|(DEM_AGND))_SCORE)$`)
}

func getAssignment(alr models.Alr) *fhirmodels.Observation {
	observation := &fhirmodels.Observation{}
	observation.Identifier = []*fhirdatatypes.Identifier{mbiIdentifier(alr.BeneMBI)}
	observation.Code = codeableConcept("Assignment", "Assignment flags, step, newly assigned, etc.")
	observation.Subject = &fhirdatatypes.Reference{Identifier: mbiIdentifier(alr.BeneMBI)}
}

func codeableConcept(code, display string) *fhirdatatypes.CodeableConcept {
	return &fhirdatatypes.CodeableConcept{
		Coding: []*fhirdatatypes.Coding{
			{System: fhirURI("http://cms.gov/CodeSystem/alr"),
				Code:    &fhirdatatypes.Code{Value: code},
				Display: fhirString(display)}},
	}
}

func assignmentFields() regexp.Regexp {
	[]string{"IN_VA_MAX", "CBA_FLAG", "ASSIGNMENT_TYPE", "ASSIGNED_BEFORE", "ASG_STATUS"}
}