package alr

import (
	"fmt"
	"regexp"

	"github.com/CMSgov/bcda-app/bcda/models"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

var assignmentPattern, enrollmentPattern, exclusionPattern, riskFlagsPattern, riskScoresPattern *regexp.Regexp

func init() {
	assignmentPattern = regexp.MustCompile(`^(IN_VA_MAX)|(CBA_FLAG)|(ASSIGNMENT_TYPE)|(ASSIGNED_BEFORE)|(ASG_STATUS)$`)
	enrollmentPattern = regexp.MustCompile(`^EnrollFlag\d+$`)
	exclusionPattern = regexp.MustCompile(`^(EXCLUDED)|(DECEASED_EXCLUDED)|(MISSING_ID_EXCLUDED)|(PART_A_B_ONLY_EXCLUDED)|` +
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)$`)
	riskFlagsPattern = regexp.MustCompile(`^(HCC_version)|(HCC_COL_\d+)$`)
	riskScoresPattern = regexp.MustCompile(`^(BENE_RSK_R_SCRE_\d{2,})|(((ESRD)|(DIS)|(AGDU)|(AGND)|(DEM_ESRD)|(DEM_DIS)|(DEM_AGDU)|(DEM_AGND))_SCORE)$`)
}

func observations(alr *models.Alr) []*fhirmodels.Observation {
	mbi := alr.BeneMBI
	assignmentFields := make(map[string]string)
	enrollmentFields := make(map[string]string)
	exclusionFields := make(map[string]string)
	riskFlagFields := make(map[string]string)
	riskScoreFields := make(map[string]string)

	for k, v := range alr.KeyValue {
		if assignmentPattern.MatchString(k) {
			assignmentFields[k] = v
		} else if enrollmentPattern.MatchString(k) {
			enrollmentFields[k] = v
		} else if exclusionPattern.MatchString(k) {
			exclusionFields[k] = v
		} else if riskFlagsPattern.MatchString(k) {
			riskFlagFields[k] = v
		} else if riskScoresPattern.MatchString(k) {
			riskScoreFields[k] = v
		}
	}

	return []*fhirmodels.Observation{
		assignment(mbi, assignmentFields),
		enrollment(mbi, enrollmentFields),
		exclusion(mbi, exclusionFields),
		riskFlags(mbi, riskFlagFields),
		riskScores(mbi, riskScoreFields),
	}
}

func assignment(mbi string, keyValue map[string]string) *fhirmodels.Observation {
	observation := &fhirmodels.Observation{}
	observation.Code = codeableConcept("Assignment", "Assignment flags, step, newly assigned, etc.")
	observation.Subject = &fhirdatatypes.Reference{Identifier: mbiIdentifier(mbi)}
	observation.Component = observationComponents(assignmentPattern, "assignment", keyValue)
	return observation
}

func enrollment(mbi string, keyValue map[string]string) *fhirmodels.Observation {
	observation := &fhirmodels.Observation{}
	observation.Code = codeableConcept("Enrollment", "Monthly enrollment flags")
	observation.Subject = &fhirdatatypes.Reference{Identifier: mbiIdentifier(mbi)}
	observation.Component = observationComponents(enrollmentPattern, "enrollment", keyValue)
	return observation
}

func exclusion(mbi string, keyValue map[string]string) *fhirmodels.Observation {
	observation := &fhirmodels.Observation{}
	observation.Code = codeableConcept("Exclusion", "Exclusion reasons")
	observation.Subject = &fhirdatatypes.Reference{Identifier: mbiIdentifier(mbi)}
	observation.Component = observationComponents(exclusionPattern, "exclusion", keyValue)
	return observation
}

func riskFlags(mbi string, keyValue map[string]string) *fhirmodels.Observation {
	observation := &fhirmodels.Observation{}
	observation.Code = codeableConcept("Risk Flags", "HCC risk flags")
	observation.Subject = &fhirdatatypes.Reference{Identifier: mbiIdentifier(mbi)}
	observation.Component = riskFlagObservationComponents(keyValue)
	return observation
}

func riskScores(mbi string, keyValue map[string]string) *fhirmodels.Observation {
	observation := &fhirmodels.Observation{}
	observation.Code = codeableConcept("Risk Scores", "HCC and other risk scores")
	observation.Subject = &fhirdatatypes.Reference{Identifier: mbiIdentifier(mbi)}
	observation.Component = observationComponents(exclusionPattern, "risk_scores", keyValue)
	return observation
}

func codeableConcept(code, display string) *fhirdatatypes.CodeableConcept {
	return &fhirdatatypes.CodeableConcept{
		Coding: []*fhirdatatypes.Coding{
			{System: fhirURI("http://cms.gov/CodeSystem/alr"),
				Code: &fhirdatatypes.Code{Value: "ALR"},
			},
			{System: fhirURI("http://cms.gov/CodeSystem/alr"),
				Code:    &fhirdatatypes.Code{Value: code},
				Display: fhirString(display),
			},
		},
	}
}

func observationComponents(pattern *regexp.Regexp, system string, keyValue map[string]string) []*fhirmodels.Observation_Component {
	var components []*fhirmodels.Observation_Component
	for k, v := range keyValue {
		component := observationComponent(system, k, v)
		components = append(components, component)
	}
	return components
}

func riskFlagObservationComponents(keyValue map[string]string) []*fhirmodels.Observation_Component {
	const hccVersion = "HCC_version"
	version := keyValue[hccVersion]
	if version == "" {
		return nil
	}
	system := fmt.Sprintf("hcc/%s", version)

	var components []*fhirmodels.Observation_Component
	for k, v := range keyValue {
		hcc := hccData(version, k)
		if hcc == nil {
			continue
		}
		component := observationComponent(system, hcc.flag, v)
		component.Code.Coding[0].Version = fhirString(version)
		component.Code.Coding[0].Display = fhirString(hcc.description)
		components = append(components, component)
	}

	return components
}

func observationComponent(system, code, value string) *fhirmodels.Observation_Component {
	return &fhirmodels.Observation_Component{
		Code: &fhirdatatypes.CodeableConcept{
			Coding: []*fhirdatatypes.Coding{
				{
					System: fhirURI(fmt.Sprintf("http://cms.gov/CodeSystem/alr/%s", system)),
					Code:   &fhirdatatypes.Code{Value: code},
				},
			},
		},
		Value: &fhirmodels.Observation_Component_Value{
			Value: &fhirmodels.Observation_Component_Value_StringValue{
				StringValue: fhirString(value),
			},
		},
	}
}
