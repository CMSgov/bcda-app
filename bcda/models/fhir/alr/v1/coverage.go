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
	coverage.Extension = []*fhirdatatypes.Extension{}
	coverage.Beneficiary = &fhirdatatypes.Reference{
		Reference: &fhirdatatypes.Reference_PatientId{
			PatientId: &fhirdatatypes.ReferenceId{Value: mbi},
		},
	}
	// coverage.Payor = []*fhirdatatypes.Reference{}
	// payorRef := &fhirdatatypes.Reference{
	// 	Reference: &fhirdatatypes.Reference_OrganizationId{
	// 		OrganizationId: &fhirdatatypes.ReferenceId{Value: "example-org-id"},
	// 	}}
	// coverage.Payor = append(coverage.Payor, payorRef)

	coverage.Meta = &fhirdatatypes.Meta{
		LastUpdated: &fhirdatatypes.Instant{
			Precision: fhirdatatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
		Profile: []*fhirdatatypes.Uri{
			{Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Coverage"},
		},
	}

	for _, kv := range keyValue {
		// FHIR does not include empty K:V pairs
		if kv.Value == "" {
			continue
		}
		subExt := &fhirdatatypes.Extension{}

		ext := &fhirdatatypes.Extension{}
		ext.Url = &fhirdatatypes.Uri{Value: kv.Key}
		ext.Value = &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_StringValue{
				StringValue: fhirString(kv.Value),
			},
		}

		subExt.Extension = append(subExt.Extension, ext)
		subExt.Url = &fhirdatatypes.Uri{Value: "http://alr.cms.gov/ig/StructureDefinition/ext-enrollmentFlag"}

		coverage.Extension = append(coverage.Extension, subExt)
	}

	return coverage
}
