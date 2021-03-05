package alr

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// ToFHIR encodes the models.Alr into a FHIR Patient and N FHIR Observation resources
func ToFHIR(alr *models.Alr, lastUpdated time.Time) (*fhirmodels.Patient, []*fhirmodels.Observation) {
	p := patient(alr)
	p.Meta = &fhirdatatypes.Meta{LastUpdated: fhirInstant(lastUpdated)}

	obs := observations(alr)
	for _, o := range obs {
		o.Meta = &fhirdatatypes.Meta{LastUpdated: fhirInstant(lastUpdated)}
	}

	return p, obs
}
