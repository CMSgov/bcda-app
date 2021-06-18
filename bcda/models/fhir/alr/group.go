package alr

import (
	"regexp"
	"strconv"

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
		`(GHP_EXCLUDED)|(OUTSIDE_US_EXCLUDED)|(OTHER_SHARED_SAV_INIT)$`)
	claimsBasedAssignmentFlagP    = regexp.MustCompile(`^CBA_FLAG$`)
	claimsBasedAssignmentStepP    = regexp.MustCompile(`^ASSIGNMENT_TYPE$`)
	newlyAssignedBeneficiaryFlagP = regexp.MustCompile(`^ASG_STATUS$`)
	pervAssignedBeneficiaryFlagP  = regexp.MustCompile(`^ASSIGNED_BEFORE$`)
	voluntaryAlignmentFlagP       = regexp.MustCompile(`^IN_VA_MAX$`)
)

// group takes a beneficiary and their respective K:V enrollment and returns FHIR
func group(mbi string, keyValue []kvPair) *fhirmodels.Group {
	group := &fhirmodels.Group{}
	group.Member = []*fhirmodels.Group_Member{{}}
	extension := []*fhirdatatypes.Extension{}

	for _, kv := range keyValue {
		switch {
		case changeTypeP.MatchString(kv.key):
			// ext - changeType
			var val = "nochange"
			// Mapping to DaVinci ATR
			if kv.value == "1" {
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
		case changeReasonP.MatchString(kv.key):
			// ext - changeReason

			// get the variable name from the map set in mapping.go
			display := groupPatternDescriptions[kv.key]

			ext := extensionMaker("reasonCode",
				"", kv.key, "https://bluebutton.cms.gov/resources/variables/alr/changeReason/", display)
			// TODO: Need to put in diplay when we figure out best way

			extension = append(extension, ext)
		case claimsBasedAssignmentFlagP.MatchString(kv.key):
			// ext - claimsBasedAssignmentFlag
			var val = true
			if kv.value == "0" {
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
		case claimsBasedAssignmentStepP.MatchString(kv.key):
			// ext - claimsBasedAssignmentStep

			val, err := strconv.ParseInt(kv.value, 10, 32)
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
		case newlyAssignedBeneficiaryFlagP.MatchString(kv.key):
			// ext - newlyAssignedBeneficiaryFlag
			var val = true
			if kv.value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-newlyAssignedBeneficiaryFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}
		case pervAssignedBeneficiaryFlagP.MatchString(kv.key):
			// ext - pervAssignedBeneficiaryFlag
			var val = true
			if kv.value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-prevAssignedBeneficiaryFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}
		case voluntaryAlignmentFlagP.MatchString(kv.key):
			// ext - voluntaryAlignmentFlag
			var val = true
			if kv.value == "0" {
				val = false
			}

			ext := extensionMaker("http://alr.cms.gov/ig/StructureDefinition/ext-newlyAssignedBeneficiaryFlag",
				"", "", "", "")
			ext.Value = &fhirdatatypes.Extension_ValueX{
				Choice: &fhirdatatypes.Extension_ValueX_Boolean{
					Boolean: &fhirdatatypes.Boolean{Value: val},
				},
			}
		}

	}

	// NOTE: there is only one element in Member slice
	group.Member[0].Extension = extension
	group.Member[0].Entity = &fhirdatatypes.Reference{Id: &fhirdatatypes.String{Value: mbi}}

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
