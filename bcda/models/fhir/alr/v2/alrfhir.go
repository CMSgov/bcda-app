package v2

import (
	"strings"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	"github.com/CMSgov/bcda-app/log"

	"github.com/google/fhir/go/jsonformat"
	r4Coverage "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/coverage_go_proto"
	r4Eoc "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/episode_of_care_go_proto"
	r4Group "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/group_go_proto"
	r4Obs "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/observation_go_proto"
	r4Patient "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/patient_go_proto"
	r4Risk "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/risk_assessment_go_proto"
)

var marshaller *jsonformat.Marshaller

func init() {
	var err error
	marshaller, err = jsonformat.NewMarshaller(false, "", "", jsonformat.R4)
	if err != nil {
		log.API.Panic("Could not get JSON FHIR marshaller for R4.")
	}
}

type AlrBulkV2 struct {
	Patient      *r4Patient.Patient
	Coverage     *r4Coverage.Coverage
	Group        *r4Group.Group
	Risk         []*r4Risk.RiskAssessment
	Observation  *r4Obs.Observation
	CovidEpisode *r4Eoc.EpisodeOfCare
}

func ToFHIRV2(alr []*models.Alr) []*AlrBulkV2 {

	bulk := []*AlrBulkV2{}

	for i := range alr {
		alr_piece := alr[i]
		kvArenaInstance := utils.KeyValueMapper(alr_piece)
		hccVersion := kvArenaInstance.HccVersion
		// there should only be one entry in the slice, but here we just check for at least one
		if len(hccVersion) < 1 {
			log.API.Warnf("Could not get HCC version.")
			return nil
		}

		sub := patient(alr_piece)
		cov := coverage(alr_piece.BeneMBI, kvArenaInstance.Enrollment, alr_piece.Timestamp)
		risk := riskAssessment(alr_piece.BeneMBI, kvArenaInstance.RiskScore, alr_piece.Timestamp)
		obs := observations(hccVersion[0].Value, alr_piece.BeneMBI, kvArenaInstance.RiskFlag, alr_piece.Timestamp)
		covid := covidEpisode(alr_piece.BeneMBI, kvArenaInstance.CovidEpsisode, alr_piece.Timestamp)

		bulk = append(bulk, &AlrBulkV2{
			Patient:      sub,
			Coverage:     cov,
			Group:        nil,
			Risk:         risk,
			Observation:  obs,
			CovidEpisode: covid,
		})
	}

	kvArenaInstance := utils.KeyValueMapper(alr[0])
	group := group(alr, kvArenaInstance.Group, alr[0].Timestamp)
	bulk = append(bulk, &AlrBulkV2{
		Group: group,
	})

	return bulk
}

func (bulk *AlrBulkV2) FhirToString() ([]string, error) {

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
	risk := strings.Join(riskAssessment, "\n")

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
