package v2

import (
	//"github.com/CMSgov/bcda-app/bcda/models"
	//fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// This part of the package houses the logical to create coverage resource type data

// coverage takes a beneficiary and their respective K:V enrollment and returns FHIR
func coverage(mbi string, keyValue []kvPair) *fhirmodels.Coverage {

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

	for _, kv := range keyValue {
		// FHIR does not include empty K:V pairs
		if kv.value == "" {
			continue
		}

		ext := &fhirdatatypes.Extension{}
		ext.Url = &fhirdatatypes.Uri{Value: kv.key}
		ext.Value = &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_StringValue{
				StringValue: fhirString(kv.value),
			},
		}

		coverage.Extension = append(coverage.Extension, ext)
	}

	return coverage
}
