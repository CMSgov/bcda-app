package alr

import (
	//"github.com/CMSgov/bcda-app/bcda/models"
	//fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// This part of the package houses the logical to create coverage resource type data

// coverage takes a beneficiary and their respective K:V enrollment and returns FHIR
func coverage(mbi string, keyValue map[string]string) *fhirmodels.Coverage {

	coverage := &fhirmodels.Coverage{}
	coverage.Id = &fhirdatatypes.Id{Value: mbi}
	coverage.Meta = &fhirdatatypes.Meta{}
	coverage.Meta.Profile = []*fhirdatatypes.
		Uri{{Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Coverage"}}
	coverage.Extension = make([]*fhirdatatypes.Extension, len(keyValue))

	extension := coverage.Extension

	var cnt uint = 0
	for k, v := range keyValue {
		// FHIR does not include empty K:V pairs
		if v != "" {
			cnt++
			continue
		}

		ext := extension[cnt]
		ext.Url = fhirURI(k)
		ext.Value = &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_StringValue{
				StringValue: fhirString(v),
			},
		}
		cnt++
	}

	return coverage
}
