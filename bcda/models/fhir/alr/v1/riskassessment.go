package v1

import (
	"regexp"

	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
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

func riskAssessment(mbi string, keyValue []kvPair) []*fhirmodels.RiskAssessment {
	// Setting up the four possible Risk Assessments

	ra := []*fhirmodels.RiskAssessment{}

	mrsCollection := []kvPair{}

	for _, kv := range keyValue {

		if kv.value == "" {
			continue
		}

		switch {
		case monthlyRiskScore.MatchString(kv.key):
			// All monthlyRiskScore need to be in one RiskAssessment
			// So we collect them in a kvPair, and then process them separately
			// after this for loop with monthlyRiskScoreMaker
			mrsCollection = append(mrsCollection, kvPair{
				key:   kv.key,
				value: kv.value,
			})
		case esrdRiskScore.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-esrd",
				"https://bluebutton.cms.gov/resources/variables/alr/esrd-score",
				kv.key, "CMS-HCC Risk Score for ESRD")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)

		case disabledRiskScore.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-disabled",
				"https://bluebutton.cms.gov/resources/variables/alr/disabled-score",
				kv.key, "CMS-HCC Risk Score for disabled")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)

		case ageduRiskScore.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-aged-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/aged-dual-score",
				kv.key, "CMS-HCC Risk Score for Aged/Dual")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)

		case agendRiskScore.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-aged-non-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/aged-non-dual-score",
				kv.key, "CMS-HCC Risk Score for Aged/Non-dual Status")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)

		case demoEsrd.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-esrd",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-esrd-score",
				kv.key, "Demographic Risk Score for ESRD Status")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)
		case demoDisabled.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-disabled",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-disabled-score",
				kv.key, "Demographic Risk Score for Disabled Status")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)
		case demoDuRiskScore.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-aged-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-aged-dual-score",
				kv.key, "Demographic Risk Score for Aged/Dual Status")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)
		case demoNdRiskScore.MatchString(kv.key):

			risk := riskMaker(mbi, "example-id-risk-score-demo-aged-non-dual",
				"https://bluebutton.cms.gov/resources/variables/alr/demo-aged-non-dual-score",
				kv.key, "Demographic Risk Score for Aged/Non-dual Status")
			risk.Prediction = predictionMaker(kv.key, kv.value)

			ra = append(ra, risk)
		}
	}

	mrsRA := monthlyRiskScoreMaker(mbi, mrsCollection)
	ra = append(ra, mrsRA)

	return ra
}

func monthlyRiskScoreMaker(mbi string, keyValue []kvPair) *fhirmodels.RiskAssessment {

	risk := riskMaker(mbi, "example-id-monthly-risk-score",
		"https://bluebutton.cms.gov/resources/variables/alr/bene_rsk_r_scre",
		"BENE_RSK_R_SCRE", "CMS-HCC Monthly Risk Scores")
	prediction := []*fhirmodels.RiskAssessment_Prediction{}

	for _, kv := range keyValue {
		prediction = append(prediction, &fhirmodels.RiskAssessment_Prediction{
			Probability: &fhirmodels.RiskAssessment_Prediction_Probability{
				Probability: &fhirmodels.RiskAssessment_Prediction_Probability_Decimal{
					Decimal: &fhirdatatypes.Decimal{Value: kv.value},
				},
			},
			Id: &fhirdatatypes.String{Value: kv.key},
		})
	}

	risk.Prediction = prediction

	return risk
}

func riskMaker(mbi, id, system, code, display string) *fhirmodels.RiskAssessment {

	risk := &fhirmodels.RiskAssessment{}
	risk.Id = &fhirdatatypes.Id{Value: id}
	risk.Subject = &fhirdatatypes.Reference{
		Reference: &fhirdatatypes.Reference_BasicId{
			BasicId: &fhirdatatypes.ReferenceId{Value: mbi},
		},
	}
	risk.Meta = &fhirdatatypes.Meta{
		Profile: []*fhirdatatypes.Uri{{
			Value: "http://alr.cms.gov/ig/StructureDefinition/alr-RiskAssessment",
		}},
	}

	risk.Code = &fhirdatatypes.CodeableConcept{
		Coding: []*fhirdatatypes.Coding{{
			System:  &fhirdatatypes.Uri{Value: system},
			Code:    &fhirdatatypes.Code{Value: code},
			Display: &fhirdatatypes.String{Value: display},
		}},
	}

	return risk
}

func predictionMaker(key, value string) []*fhirmodels.RiskAssessment_Prediction {
	prediction := []*fhirmodels.RiskAssessment_Prediction{{
		Probability: &fhirmodels.RiskAssessment_Prediction_Probability{
			Probability: &fhirmodels.RiskAssessment_Prediction_Probability_Decimal{
				Decimal: &fhirdatatypes.Decimal{Value: value},
			},
		},
		Id: &fhirdatatypes.String{Value: key},
	}}
	return prediction
}
