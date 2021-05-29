package alr

import (
	"github.com/CMSgov/bcda-app/log"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

func observations(version, mbi string, keyValue []kvPair) *fhirmodels.Observation {
	obs := &fhirmodels.Observation{}
	obs.Id = &fhirdatatypes.Id{Id: &fhirdatatypes.String{
		Value: "example-id-hcc-risk-flags",
	}}
	obs.Meta = &fhirdatatypes.Meta{
		Profile: []*fhirdatatypes.Uri{{
			Value: "http://alr.cms.gov/ig/StructureDefinition/alr-HccRiskFlag",
		}},
	}
	obs.Code = &fhirdatatypes.CodeableConcept{
		Coding: []*fhirdatatypes.Coding{{
			System: &fhirdatatypes.Uri{
				Value: "https://bluebutton.cms.gov/resources/variables/alr/hcc-risk-flags",
			},
			Code:    &fhirdatatypes.Code{Value: "hccRiskFlags"},
			Version: &fhirdatatypes.String{Value: version},
		}},
		Text: &fhirdatatypes.String{Value: "HCC Risk Flags"},
	}
	obs.Subject = &fhirdatatypes.Reference{
		Identifier: &fhirdatatypes.Identifier{
			System: &fhirdatatypes.Uri{Value: "https://bluebutton.cms.gov/resources/variables/bene_id"},
			Value:  &fhirdatatypes.String{Value: mbi},
		},
	}

	components := []*fhirmodels.Observation_Component{}

	for _, kv := range keyValue {

		// Get information from HCC Crosswalk
		hccinfo := hccData(version, kv.key)

		if hccinfo == nil {
			// If a nil is returned, we could not give the field in the crosswalk...
			// for now will skip
			log.API.Warnf("We would not find %s in the crosswalk for %s. with value %s", kv.key, version, kv.value)
			continue
		}

		comp := &fhirmodels.Observation_Component{}
		comp.Code = &fhirdatatypes.CodeableConcept{
			Coding: []*fhirdatatypes.Coding{{
				System:  &fhirdatatypes.Uri{Value: "https://bluebutton.cms.gov/resources/variables/alr/hcc-risk-flags"},
				Version: &fhirdatatypes.String{Value: version},
				Code:    &fhirdatatypes.Code{Value: hccinfo.flag},
				Display: &fhirdatatypes.String{Value: hccinfo.description},
			}},
		}
		comp.Value = &fhirmodels.Observation_Component_Value{
			Value: &fhirmodels.Observation_Component_Value_StringValue{
				StringValue: &fhirdatatypes.String{Value: kv.value},
			},
		}

		components = append(components, comp)
	}

	obs.Component = components

	return obs
}
