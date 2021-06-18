package alr

import (
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/log"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

type AlrFhirBulk struct {
	Patient     *fhirmodels.Patient
	Coverage    *fhirmodels.Coverage
	Group       *fhirmodels.Group
	Risk        []*fhirmodels.RiskAssessment
	Observation *fhirmodels.Observation
}

// ToFHIR encodes the models.Alr into a FHIR Resources
func ToFHIR(alr *models.Alr) *AlrFhirBulk {

	kvArenaInstance := keyValueMapper(alr)
	hccVersion := kvArenaInstance.hccVersion
    // there should only be one entry in the slice, but here we just check for at least one
	if len(hccVersion) < 1 {
		log.API.Warnf("Could not get HCC version.")
		return nil
	}

	sub := patient(alr)
	cov := coverage(alr.BeneMBI, kvArenaInstance.enrollment)
	group := group(alr.BeneMBI, kvArenaInstance.group)
	risk := riskAssessment(alr.BeneMBI, kvArenaInstance.riskScore)
	obs := observations(hccVersion[0].value, alr.BeneMBI, kvArenaInstance.riskFlag)

	return &AlrFhirBulk{
		Patient:     sub,
		Coverage:    cov,
		Group:       group,
		Risk:        risk,
		Observation: obs,
	}
}
