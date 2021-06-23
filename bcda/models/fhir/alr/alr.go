package alr

import (
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/log"
    v1 "github.com/CMSgov/bcda-app/bcda/models/fhir/alr/v1"

    r4Patient "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"
    r4Coverage "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/coverage_go_proto"
    r4Group "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/group_go_proto"
    r4Risk "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/risk_assessment_go_proto"
    r4Obs "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/observation_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

type AlrBulkV1 struct {
	Patient     *fhirmodels.Patient
	Coverage    *fhirmodels.Coverage
	Group       *fhirmodels.Group
	Risk        []*fhirmodels.RiskAssessment
	Observation *fhirmodels.Observation
}

type AlrBulkV2 struct {
	Patient     *r4Patient.Patient
	Coverage    *r4Coverage.Coverage
	Group       *r4Group.Group
	Risk        []*r4Risk.RiskAssessment
	Observation *r4Obs.Observation
}

type AlrFhirBulk struct {

}

type AlrV1 struct {}
type AlrV2 struct {}

// ToFHIR encodes the models.Alr into a FHIR Patient and N FHIR Observation resources
func (_ *AlrV1) ToFHIR(alr *models.Alr) *AlrBulkV1 {

	kvArenaInstance := keyValueMapper(alr)
	hccVersion := kvArenaInstance.hccVersion
    // there should only be one entry in the slice, but here we just check for at least one
	if len(hccVersion) < 1 {
		log.API.Warnf("Could not get HCC version.")
		return nil
	}

	sub := v1.Patient(alr)
	cov := coverage(alr.BeneMBI, kvArenaInstance.enrollment)
	group := group(alr.BeneMBI, kvArenaInstance.group)
	risk := riskAssessment(alr.BeneMBI, kvArenaInstance.riskScore)
	obs := observations(hccVersion[0].value, alr.BeneMBI, kvArenaInstance.riskFlag)

	return &AlrBulkV1{
		Patient:     sub,
		Coverage:    cov,
		Group:       group,
		Risk:        risk,
		Observation: obs,
	}
}

func (_ *AlrV2) ToFHIR(alr *models.Alr) *AlrBulkV2 {

	kvArenaInstance := keyValueMapper(alr)
	hccVersion := kvArenaInstance.hccVersion
    // there should only be one entry in the slice, but here we just check for at least one
	if len(hccVersion) < 1 {
		log.API.Warnf("Could not get HCC version.")
		return nil
	}

	sub := v1.Patient(alr)
	cov := coverage(alr.BeneMBI, kvArenaInstance.enrollment)
	group := group(alr.BeneMBI, kvArenaInstance.group)
	risk := riskAssessment(alr.BeneMBI, kvArenaInstance.riskScore)
	obs := observations(hccVersion[0].value, alr.BeneMBI, kvArenaInstance.riskFlag)

	return &AlrBulkV2{
		Patient:     sub,
		Coverage:    cov,
		Group:       group,
		Risk:        risk,
		Observation: obs,
	}
}
