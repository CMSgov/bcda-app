package v1

import (
	"strings"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	"github.com/CMSgov/bcda-app/log"

	"github.com/google/fhir/go/jsonformat"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

var marshaller *jsonformat.Marshaller

func init() {
	var err error
	marshaller, err = jsonformat.NewMarshaller(false, "", "", jsonformat.STU3)
	if err != nil {
		log.API.Panic("Could not get JSON FHIR marshaller for STU3.")
	}
}

type AlrBulkV1 struct {
	Patient      *fhirmodels.Patient
	Coverage     *fhirmodels.Coverage
	Group        *fhirmodels.Group
	Risk         []*fhirmodels.RiskAssessment
	Observation  *fhirmodels.Observation
	CovidEpisode *fhirmodels.EpisodeOfCare
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
	cov := coverage(alr.BeneMBI, kvArenaInstance.Enrollment, alr.Timestamp)
	group := group(alr.BeneMBI, kvArenaInstance.Group, alr.Timestamp)
	risk := riskAssessment(alr.BeneMBI, kvArenaInstance.RiskScore, alr.Timestamp)
	obs := observations(hccVersion[0].Value, alr.BeneMBI, kvArenaInstance.RiskFlag, alr.Timestamp)
	covid := covidEpisode(alr.BeneMBI, kvArenaInstance.CovidEpsisode, alr.Timestamp)

	return &AlrBulkV1{
		Patient:      sub,
		Coverage:     cov,
		Group:        group,
		Risk:         risk,
		Observation:  obs,
		CovidEpisode: covid,
	}
}

func (bulk *AlrBulkV1) FhirToString() ([]string, error) {

	patientb, err := marshaller.MarshalResource(bulk.Patient)
	if err != nil {
		// Make sure to send err back to the other thread
		log.API.Errorf("Could not convert patient fhir to json.")
		return nil, err
	}
	patients := string(patientb) + "\n"

	// COVERAGE
	coverageb, err := marshaller.MarshalResource(bulk.Coverage)
	if err != nil {
		// Make sure to send err back to the other thread
		log.API.Errorf("Could not convert patient fhir to json.")
		return nil, err
	}
	coverage := string(coverageb) + "\n"

	// GROUP
	groupb, err := marshaller.MarshalResource(bulk.Group)
	if err != nil {
		// Make sure to send err back to the other thread
		log.API.Errorf("Could not convert patient fhir to json.")
		return nil, err
	}
	group := string(groupb) + "\n"

	// RISK
	var riskAssessment = []string{}

	for _, r := range bulk.Risk {

		riskb, err := marshaller.MarshalResource(r)
		if err != nil {
			// Make sure to send err back to the other thread
			log.API.Errorf("Could not convert patient fhir to json.")
			return nil, err
		}
		risk := string(riskb)
		riskAssessment = append(riskAssessment, risk)
	}
	risk := strings.Join(riskAssessment, "\n") + "\n"

	// OBSERVATION
	observationb, err := marshaller.MarshalResource(bulk.Observation)
	if err != nil {
		log.API.Errorf("Could not convert patient fhir to json.")
		return nil, err
	}
	observation := string(observationb) + "\n"

	// COVID
	covidb, err := marshaller.MarshalResource(bulk.CovidEpisode)
	if err != nil {
		log.API.Errorf("Could not convert covid fhir to json.")
		return nil, err
	}
	covidEpisode := string(covidb) + "\n"

	return []string{patients, observation, coverage, group, risk, covidEpisode}, nil

}
