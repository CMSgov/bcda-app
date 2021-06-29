package v1

import (
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// This part of the package houses the logical to create coverage resource type data

// coverage takes a beneficiary and their respective K:V enrollment and returns FHIR
func coverage(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *fhirmodels.Coverage {

	coverage := &fhirmodels.Coverage{}
	coverage.Id = &fhirdatatypes.Id{Value: mbi}
	coverage.Meta = &fhirdatatypes.Meta{
		Profile: []*fhirdatatypes.
			Uri{{Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Coverage"}},
	}
	coverage.Extension = []*fhirdatatypes.Extension{}
	coverage.Beneficiary = &fhirdatatypes.Reference{
		Reference: &fhirdatatypes.Reference_OrganizationId{
			OrganizationId: &fhirdatatypes.ReferenceId{Value: mbi},
		},
	}
	coverage.Meta = &fhirdatatypes.Meta{
		LastUpdated: &fhirdatatypes.Instant{
			Precision: fhirdatatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
	}

	for _, kv := range keyValue {
		// FHIR does not include empty K:V pairs
		if kv.Value == "" {
			continue
		}

		ext := &fhirdatatypes.Extension{}
		ext.Url = &fhirdatatypes.Uri{Value: kv.Key}
		ext.Value = &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_StringValue{
				StringValue: fhirString(kv.Value),
			},
		}

		coverage.Extension = append(coverage.Extension, ext)
	}

	return coverage
}
