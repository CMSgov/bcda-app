package v1

import (
	"github.com/CMSgov/bcda-app/bcda/models"
    "github.com/CMSgov/bcda-app/log"
    "github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"

	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

type AlrBulkV1 struct {
	Patient     *fhirmodels.Patient
	Coverage    *fhirmodels.Coverage
	Group       *fhirmodels.Group
	Risk        []*fhirmodels.RiskAssessment
	Observation *fhirmodels.Observation
}

// ToFHIR encodes the models.Alr into a FHIR Patient and N FHIR Observation resources
func ToFHIRV1(alr *models.Alr) *AlrBulkV1 {

	kvArenaInstance := utils.KeyValueMapper(alr)
	hccVersion := kvArenaInstance.HccVersion
    // there should only be one entry in the slice, but here we just check for at least one
	if len(hccVersion) < 1 {
		log.API.Warnf("Could not get HCC version.")
		return nil
	}

	sub := patient(alr)
	cov := coverage(alr.BeneMBI, kvArenaInstance.Enrollment)
	group := group(alr.BeneMBI, kvArenaInstance.Group)
	risk := riskAssessment(alr.BeneMBI, kvArenaInstance.RiskScore)
	obs := observations(hccVersion[0].Value, alr.BeneMBI, kvArenaInstance.RiskFlag)

	return &AlrBulkV1{
		Patient:     sub,
		Coverage:    cov,
		Group:       group,
		Risk:        risk,
		Observation: obs,
	}
}

