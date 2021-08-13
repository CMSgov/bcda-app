package v2

import (
	//"github.com/CMSgov/bcda-app/bcda/models"
	//fhircodes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/codes_go_proto"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"

	r4Datatypes "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/datatypes_go_proto"
	r4Models "github.com/google/fhir/go/proto/google/fhir/proto/r4/core/resources/coverage_go_proto"
)

// This part of the package houses the logical to create coverage resource type data

// coverage takes a beneficiary and their respective K:V enrollment and returns FHIR
func coverage(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *r4Models.Coverage {

	coverage := &r4Models.Coverage{}
	coverage.Id = &r4Datatypes.Id{Value: mbi}
	coverage.Extension = []*r4Datatypes.Extension{}
	coverage.Beneficiary = &r4Datatypes.Reference{
		Reference: &r4Datatypes.Reference_PatientId{
			PatientId: &r4Datatypes.ReferenceId{Value: mbi},
		},
	}
	// coverage.Payor = []*r4Datatypes.Reference{}
	// payorRef := &r4Datatypes.Reference{
	// 	Reference: &r4Datatypes.Reference_OrganizationId{
	// 		OrganizationId: &r4Datatypes.ReferenceId{Value: "example-org-id"},
	// 	}}
	// coverage.Payor = append(coverage.Payor, payorRef)

	coverage.Meta = &r4Datatypes.Meta{
		LastUpdated: &r4Datatypes.Instant{
			Precision: r4Datatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
		Profile: []*r4Datatypes.Canonical{
			{Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Coverage"},
		},
	}

	for _, kv := range keyValue {
		// FHIR does not include empty K:V pairs
		if kv.Value == "" {
			continue
		}
		subExt := &r4Datatypes.Extension{}

		ext := &r4Datatypes.Extension{}
		ext.Url = &r4Datatypes.Uri{Value: kv.Key}
		ext.Value = &r4Datatypes.Extension_ValueX{
			Choice: &r4Datatypes.Extension_ValueX_StringValue{
				StringValue: &r4Datatypes.String{Value: kv.Value},
			},
		}
		subExt.Extension = append(subExt.Extension, ext)
		subExt.Url = &r4Datatypes.Uri{Value: "http://alr.cms.gov/ig/StructureDefinition/ext-enrollmentFlag"}

		coverage.Extension = append(coverage.Extension, subExt)
	}

	return coverage
}
