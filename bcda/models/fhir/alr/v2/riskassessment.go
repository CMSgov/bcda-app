package v2

import (
	"regexp"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/risk_assessment_go_proto"
)

// This part of the package houses the logic to create risk assessment

// Further break down of Risk...
var (
	monthlyRiskScore  = regexp.MustCompile(`^BENE_RSK_R_SCRE_\d+$`)
	esrdRiskScore     = regexp.MustCompile(`^ESRD_SCORE$`)
	disabledRiskScore = regexp.MustCompile(`^DIS_SCORE$`)
	ageduRiskScore    = regexp.MustCompile(`^AGDU_SCORE$`)
	agendRiskScore    = regexp.MustCompile(`^AGND_SCORE$`)
	demoEsrd          = regexp.MustCompile(`^DEM_ESRD_SCORE$`)
	demoDisabled      = regexp.MustCompile(`^DEM_DIS_SCORE$`)
	demoDuRiskScore   = regexp.MustCompile(`^DEM_AGDU_SCORE$`)
	demoNdRiskScore   = regexp.MustCompile(`^DEM_AGND_SCORE$`)
)

func riskAssessment(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) []*r4Models.RiskAssessment {
	// Setting up the four possible Risk Assessments

	ra := []*r4Models.RiskAssessment{}

	mrsCollection := []utils.KvPair{}

	for _, kv := range keyValue {

		if kv.Value == "" {
			continue
		}

		switch {
		case monthlyRiskScore.MatchString(kv.Key):
			// All monthlyRiskScore need to be in one RiskAssessment
			// So we collect them in a kvPair, and then process them separately
			// after this for loop with monthlyRiskScoreMaker
			mrsCollection = append(mrsCollection, utils.KvPair{
				Key:   kv.Key,
				Value: kv.Value,
			})
		case esrdRiskScore.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-esrd",
				"https://bluebutton.cms.gov/resources/variables/alr/esrd-score",
				kv.Key, "CMS-HCC Risk Score for ESRD", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)

		case disabledRiskScore.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-disabled",
				"https://bluebutton.cms.gov/resources/variables/alr/disabled-score",
				kv.Key, "CMS-HCC Risk Score for disabled", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)

		case ageduRiskScore.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-aged-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/aged-dual-score",
				kv.Key, "CMS-HCC Risk Score for Aged/Dual", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)

		case agendRiskScore.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-aged-non-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/aged-non-dual-score",
				kv.Key, "CMS-HCC Risk Score for Aged/Non-dual Status", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)

		case demoEsrd.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-esrd",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-esrd-score",
				kv.Key, "Demographic Risk Score for ESRD Status", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)
		case demoDisabled.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-disabled",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-disabled-score",
				kv.Key, "Demographic Risk Score for Disabled Status", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)
		case demoDuRiskScore.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-aged-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-aged-dual-score",
				kv.Key, "Demographic Risk Score for Aged/Dual Status", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)
		case demoNdRiskScore.MatchString(kv.Key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-aged-non-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-aged-non-dual-score",
				kv.Key, "Demographic Risk Score for Aged/Non-dual Status", lastUpdated)
			risk.Prediction = predictionMaker(kv.Key, kv.Value)

			ra = append(ra, risk)
		}
	}

	mrsRA := monthlyRiskScoreMaker(mbi, mrsCollection, lastUpdated)
	ra = append(ra, mrsRA)

	return ra
}

func monthlyRiskScoreMaker(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *r4Models.RiskAssessment {

	risk := riskMaker(mbi, "example-id-monthly-risk-score",
		"https://bluebutton.cms.gov/resources/variables/alr/bene_rsk_r_scre",
		"BENE_RSK_R_SCRE", "CMS-HCC Monthly Risk Scores", lastUpdated)
	prediction := []*r4Models.RiskAssessment_Prediction{}

	for _, kv := range keyValue {
		prediction = append(prediction, &r4Models.RiskAssessment_Prediction{
			Probability: &r4Models.RiskAssessment_Prediction_ProbabilityX{
				Choice: &r4Models.RiskAssessment_Prediction_ProbabilityX_Decimal{
					Decimal: &r4Datatypes.Decimal{Value: kv.Value},
				},
			},
			Id: &r4Datatypes.String{Value: kv.Key},
		})
	}

	risk.Prediction = prediction

	return risk
}

func riskMaker(mbi, id, system, code, display string, lastUpdated time.Time) *r4Models.RiskAssessment {

	risk := &r4Models.RiskAssessment{}
	risk.Id = &r4Datatypes.Id{Value: id}
	risk.Subject = &r4Datatypes.Reference{
		Reference: &r4Datatypes.Reference_PatientId{
			PatientId: &r4Datatypes.ReferenceId{Value: mbi},
		},
	}
	risk.Basis = []*r4Datatypes.Reference{}
	obsRef := &r4Datatypes.Reference{
		Reference: &r4Datatypes.Reference_ObservationId{
			ObservationId: &r4Datatypes.ReferenceId{Value: "Observation/example-id-hcc-risk-flags"},
		},
	}
	risk.Basis = append(risk.Basis, obsRef)

	risk.Meta = &r4Datatypes.Meta{
		Profile: []*r4Datatypes.Canonical{{
			Value: "http://alr.cms.gov/ig/StructureDefinition/alr-RiskAssessment",
		}},
		LastUpdated: &r4Datatypes.Instant{
			Precision: r4Datatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
	}

	risk.Code = &r4Datatypes.CodeableConcept{
		Coding: []*r4Datatypes.Coding{{
			System:  &r4Datatypes.Uri{Value: system},
			Code:    &r4Datatypes.Code{Value: code},
			Display: &r4Datatypes.String{Value: display},
		}},
	}

	return risk
}

func predictionMaker(key, value string) []*r4Models.RiskAssessment_Prediction {
	prediction := []*r4Models.RiskAssessment_Prediction{{
		Probability: &r4Models.RiskAssessment_Prediction_ProbabilityX{
			Choice: &r4Models.RiskAssessment_Prediction_ProbabilityX_Decimal{
				Decimal: &r4Datatypes.Decimal{Value: value},
			},
		},
		Id: &r4Datatypes.String{Value: key},
	}}
	return prediction
}
