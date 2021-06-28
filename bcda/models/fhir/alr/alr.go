package alr

import (
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/log"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/v1"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/v2"
)

type AlrFhirBulk struct {
	*v1.AlrBulkV1
	*v2.AlrBulkV2
}

func ToFHIR(alr *models.Alr, version string) *AlrFhirBulk {

	bulk := &AlrFhirBulk{}

	switch version {
	case "fhir/v1":
		bulk.AlrBulkV1 = v1.ToFHIRV1(alr)
	case "fhir/v2":
		bulk.AlrBulkV2 = v2.ToFHIRV2(alr)
	default:
		log.API.Errorf("Version endpoint %d not supported.", version)
		return nil
	}

	return bulk
}

