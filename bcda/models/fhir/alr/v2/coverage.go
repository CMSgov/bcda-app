package v2

import (
	//"github.com/CMSgov/bcda-app/bcda/models"
	//fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
    "github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
    
    r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
    r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/coverage_go_proto"
)

// This part of the package houses the logical to create coverage resource type data

// coverage takes a beneficiary and their respective K:V enrollment and returns FHIR
func coverage(mbi string, keyValue []utils.KvPair) *r4Models.Coverage {

	coverage := &r4Models.Coverage{}
	coverage.Id = &r4Datatypes.Id{Value: mbi}
	coverage.Meta = &r4Datatypes.Meta{
        Profile: []*r4Datatypes.Canonical{{
            Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Coverage",
        }},
	}
	coverage.Extension = []*r4Datatypes.Extension{}
	coverage.Beneficiary = &r4Datatypes.Reference{
		Reference: &r4Datatypes.Reference_OrganizationId{
			OrganizationId: &r4Datatypes.ReferenceId{Value: mbi},
		},
	}

	for _, kv := range keyValue {
		// FHIR does not include empty K:V pairs
		if kv.Value == "" {
			continue
		}

		ext := &r4Datatypes.Extension{}
		ext.Url = &r4Datatypes.Uri{Value: kv.Key}
		ext.Value = &r4Datatypes.Extension_ValueX{
            Choice: &r4Datatypes.Extension_ValueX_StringValue{
                StringValue: &r4Datatypes.String{ Value: kv.Value },
			},
		}

		coverage.Extension = append(coverage.Extension, ext)
	}

	return coverage
}
