package alr

import (
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

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

func 