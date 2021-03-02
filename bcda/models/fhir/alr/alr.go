package alr

import (
	"github.com/CMSgov/bcda-app/bcda/models"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// Alr encodes the models.Alr into a FHIR Patient and N FHIR Observation resources
func Alr(alr *models.Alr) (*fhirmodels.Patient, []*fhirmodels.Observation) {
	return patient(alr), observations(alr)
}
