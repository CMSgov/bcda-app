package v1

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	"github.com/CMSgov/bcda-app/log"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

func observations(version, mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *fhirmodels.Observation {
	obs := &fhirmodels.Observation{}
	obs.Id = &fhirdatatypes.Id{Value: "example-id-hcc-risk-flags"}
	obs.Meta = &fhirdatatypes.Meta{
		Profile: []*fhirdatatypes.Uri{{
			Value: "http://alr.cms.gov/ig/StructureDefinition/alr-HccRiskFlag",
		}},
		LastUpdated: &fhirdatatypes.Instant{
			Precision: fhirdatatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
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
		hccinfo := utils.HccData(version, kv.Key)

		if hccinfo == nil {
			// If a nil is returned, we could not give the field in the crosswalk...
			// for now will skip
			log.API.Warnf("We would not find %s in the crosswalk for %s. with value %s", kv.Key, version, kv.Value)
			continue
		}

		comp := &fhirmodels.Observation_Component{}
		comp.Code = &fhirdatatypes.CodeableConcept{
			Coding: []*fhirdatatypes.Coding{{
				System:  &fhirdatatypes.Uri{Value: "https://bluebutton.cms.gov/resources/variables/alr/hcc-risk-flags"},
				Version: &fhirdatatypes.String{Value: version},
				Code:    &fhirdatatypes.Code{Value: hccinfo.Flag},
				Display: &fhirdatatypes.String{Value: hccinfo.Description},
			}},
		}
		comp.Value = &fhirmodels.Observation_Component_Value{
			Value: &fhirmodels.Observation_Component_Value_StringValue{
				StringValue: &fhirdatatypes.String{Value: kv.Value},
			},
		}

		components = append(components, comp)
	}

	obs.Component = components

	return obs
}
