package fhir

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

func FromALR(alr models.Alr) []*fhirmodels.ContainedResource {
	return nil
}

func getPatient(alr models.Alr) *fhirmodels.Patient {
	p := &fhirmodels.Patient{}
	p.Meta = &fhirdatatypes.Meta{LastUpdated: &fhirdatatypes.Instant{
		ValueUs:   alr.Timestamp.UnixNano() / int64(time.Millisecond),
		Timezone:  time.UTC.String(),
		Precision: fhirdatatypes.Instant_MILLISECOND}}
	return nil
}
