package v2

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	"github.com/CMSgov/bcda-app/log"
	r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/observation_go_proto"
)

func observations(version, mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *r4Models.Observation {
	obs := &r4Models.Observation{}
	obs.Id = &r4Datatypes.Id{Value: "example-id-hcc-risk-flags"}
	obs.Meta = &r4Datatypes.Meta{
		Profile: []*r4Datatypes.Canonical{{
			Value: "http://alr.cms.gov/ig/StructureDefinition/alr-HccRiskFlag",
		}},
		LastUpdated: &r4Datatypes.Instant{
			Precision: r4Datatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
	}
	obs.Code = &r4Datatypes.CodeableConcept{
		Coding: []*r4Datatypes.Coding{{
			System: &r4Datatypes.Uri{
				Value: "https://bluebutton.cms.gov/resources/variables/alr/hcc-risk-flags",
			},
			Code:    &r4Datatypes.Code{Value: "hccRiskFlags"},
			Version: &r4Datatypes.String{Value: version},
		}},
		Text: &r4Datatypes.String{Value: "HCC Risk Flags"},
	}
	obs.Subject = &r4Datatypes.Reference{
		Identifier: &r4Datatypes.Identifier{
			System: &r4Datatypes.Uri{Value: "https://bluebutton.cms.gov/resources/variables/bene_id"},
			Value:  &r4Datatypes.String{Value: mbi},
		},
	}

	components := []*r4Models.Observation_Component{}

	for _, kv := range keyValue {

		// Get information from HCC Crosswalk
		hccinfo := utils.HccData(version, kv.Key)

		if hccinfo == nil {
			// If a nil is returned, we could not give the field in the crosswalk...
			// for now will skip
			log.API.Warnf("We would not find %s in the crosswalk for %s. with value %s", kv.Key, version, kv.Value)
			continue
		}

		comp := &r4Models.Observation_Component{}
		comp.Code = &r4Datatypes.CodeableConcept{
			Coding: []*r4Datatypes.Coding{{
				System:  &r4Datatypes.Uri{Value: "https://bluebutton.cms.gov/resources/variables/alr/hcc-risk-flags"},
				Version: &r4Datatypes.String{Value: version},
				Code:    &r4Datatypes.Code{Value: hccinfo.Flag},
				Display: &r4Datatypes.String{Value: hccinfo.Description},
			}},
		}
		comp.Value = &r4Models.Observation_Component_ValueX{
			Choice: &r4Models.Observation_Component_ValueX_StringValue{
				StringValue: &r4Datatypes.String{Value: kv.Value},
			},
		}

		components = append(components, comp)
	}

	obs.Component = components

	return obs
}
