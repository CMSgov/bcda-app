package v1

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/utils"
	"github.com/CMSgov/bcda-app/log"
	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// This part of the package houses the logical to create group resource type data

// Further break down of groupPattern; the order does not matter
var (
	changeTypeP   = regexp.MustCompile(`^EXCLUDED$`)
	changeReasonP = regexp.MustCompile(`^(DECEASED_EXCLUDED)|` +
		`(MISSING_ID_EXCLUDED)|(PART_A_B_ONLY_EXCLUDED)|` +
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)|` +
		`(PLUR_R05)|(AB_R01)|(HMO_R03)|(NO_US_R02)|(MDM_R04)|(NOFND_R06)$`)
	claimsBasedAssignmentFlagP    = regexp.MustCompile(`^CBA_FLAG$`)
	claimsBasedAssignmentStepP    = regexp.MustCompile(`^ASSIGNMENT_TYPE$`)
	newlyAssignedBeneficiaryFlagP = regexp.MustCompile(`^ASG_STATUS$`)
	pervAssignedBeneficiaryFlagP  = regexp.MustCompile(`^ASSIGNED_BEFORE$`)
	voluntaryAlignmentFlagP       = regexp.MustCompile(`^IN_VA_MAX$`)
	vaSelectionOnlyP              = regexp.MustCompile(`^VA_SELECTION_ONLY$`)
)

// group takes a beneficiary and their respective K:V enrollment and returns FHIR
func group(mbi string, keyValue []utils.KvPair, lastUpdated time.Time) *fhirmodels.Group {
	group := &fhirmodels.Group{}
	group.Id = &fhirdatatypes.Id{Value: "example-id-group"}
	group.Member = []*fhirmodels.Group_Member{{}}
	extension := []*fhirdatatypes.Extension{}
	reasonCodes := &fhirdatatypes.Extension{
		Url: &fhirdatatypes.Uri{
			Value: "http://alr.cms.gov/ig/StructureDefinition/ext-changeReason",
		}}
	group.Meta = &fhirdatatypes.Meta{
		LastUpdated: &fhirdatatypes.Instant{
			Precision: fhirdatatypes.Instant_SECOND,
			ValueUs:   lastUpdated.UnixNano() / int64(time.Microsecond),
		},
		Profile: []*fhirdatatypes.Uri{
			{Value: "http://alr.cms.gov/ig/StructureDefinition/alr-Group"},
		},
	}

	for _, kv := range keyValue {
		switch {
		case changeTypeP.MatchString(kv.Key):
			// ext - changeType
			var val = "nochange"
			// Mapping to DaVinci ATR
			if kv.Value == "1" {
				val = "dropped"
			}

			ext := extensionMaker("http://hl7.org/fhir/us/davinci-atr/STU1/StructureDefinition-ext-changeType.html",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Code{
					Code: &fhirdatatypes.Code{Value: val},
				},
			}

			extension = append(extension, ext)
		case changeReasonP.MatchString(kv.Key):
			// ext - changeReason

			// Data with a value of 0 should not be included in the FHIR resource
			if kv.Value != "0" {
				// get the variable name from the map set in mapping.go
				display := utils.GroupPatternDescriptions[kv.Key]

				subExt := extensionMaker("reasonCode",
					"", kv.Key, "https://bluebutton.cms.gov/resources/variables/alr/changeReason/", display)

				reasonCodes.Extension = append(reasonCodes.Extension, subExt)
			}
		case claimsBasedAssignmentFlagP.MatchString(kv.Key):
			// ext - claimsBasedAssignmentFlag
			var val = true
			if kv.Value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-claimsBasedAssignmentFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}

			extension = append(extension, ext)
		case claimsBasedAssignmentStepP.MatchString(kv.Key):
			// ext - claimsBasedAssignmentStep

			val, err := strconv.ParseInt(kv.Value, 10, 32)
			if err != nil {
				log.API.Warnf("Could convert string to int for {}: {}", mbi, err)
			}
			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-claimsBasedAssignmentStep",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Integer{
					Integer: &fhirdatatypes.Integer{Value: int32(val)},
				},
			}
		case newlyAssignedBeneficiaryFlagP.MatchString(kv.Key):
			// ext - newlyAssignedBeneficiaryFlag
			var val = true
			if kv.Value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-newlyAssignedBeneficiaryFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}
		case pervAssignedBeneficiaryFlagP.MatchString(kv.Key):
			// ext - pervAssignedBeneficiaryFlag
			var val = true
			if kv.Value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-prevAssignedBeneficiaryFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}
		case voluntaryAlignmentFlagP.MatchString(kv.Key):
			// ext - voluntaryAlignmentFlag
			var val = true
			if kv.Value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-newlyAssignedBeneficiaryFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}
		case vaSelectionOnlyP.MatchString(kv.Key):
			// ext - vaSelectionOnlyFlag
			var val = true
			if kv.Value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-vaSelectionOnlyFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}
		}

	}
	extension = append(extension, reasonCodes)

	// NOTE: there is only one element in Member slice
	group.Member[0].Extension = extension
	group.Member[0].Entity = &fhirdatatypes.Reference{Reference: &fhirdatatypes.Reference_PatientId{
		PatientId: &fhirdatatypes.ReferenceId{Value: mbi},
	}}

	return group
}

// This is an extension resource constructor. Since values in FHIR can differ,
// this is not included in the parameter
func extensionMaker(url, reference, key, sys, disp string) *fhirdatatypes.Extension {
	extension := &fhirdatatypes.Extension{}
	// URL
	extension.Url = &fhirdatatypes.Uri{Value: url}
	// Reference
	if reference != "" {
		extension.Value = &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_Reference{
				Reference: &fhirdatatypes.Reference{
					Reference: &fhirdatatypes.Reference_Uri{Uri: &fhirdatatypes.String{Value: reference}},
				},
			},
		}
	}

	if key != "" && reference == "" {
		fmt.Println(disp)
		extension.Value = &fhirdatatypes.Extension_ValueX{
			Choice: &fhirdatatypes.Extension_ValueX_Coding{
				Coding: &fhirdatatypes.Coding{
					System:  &fhirdatatypes.Uri{Value: sys},
					Code:    &fhirdatatypes.Code{Value: key},
					Display: fhirString(disp),
				},
			},
		}
	}

	return extension
}
