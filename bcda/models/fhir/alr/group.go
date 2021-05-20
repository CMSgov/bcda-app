package alr

import (
	"fmt"
	"regexp"

	fhirdatatypes "github.com/google/fhir/go/proto/google/fhir/proto/stu3/datatypes_go_proto"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
)

// This part of the package houses the logical to create group resource type data

var groupExtensions = [...]string{
	"changeType",
	"changeReason",
	"claimsBasedAssignmentFlag",
	"claimsBasedAssignmentStep",
	"newlyAssignedBeneficiaryFlag",
	"pervAssignedBeneficiaryFlag",
	"voluntaryAlignmentFlag",
}

// The order of this const must match the order of the array above
const (
	changeType uint8 = iota
	changeReason
	claimsBasedAssignmentFlag
	claimsBasedAssignmentStep
	newlyAssignedBeneficiaryFlag
	pervAssignedBeneficiaryFlag
	voluntaryAlignmentFlag
)

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
func group(mbi string, keyValue map[string]string) *fhirmodels.Group {
	group := &fhirmodels.Group{}
	group.Member = []*fhirmodels.Group_Member{{
		Extension: make([]*fhirdatatypes.Extension, len(groupExtensions)),
	}}

	for k, v := range keyValue {
        fmt.Println(k, v)
		switch {
		// ext - changeType
		case changeTypeP.MatchString(k):

			// ext - changeReason
		case changeReasonP.MatchString(k):

			// ext - claimsBasedAssignmentFlag
		case claimsBasedAssignmentFlagP.MatchString(k):

			// ext - claimsBasedAssignmentStep
		case claimsBasedAssignmentStepP.MatchString(k):

			// ext - newlyAssignedBeneficiaryFlag
		case newlyAssignedBeneficiaryFlagP.MatchString(k):

			// ext - pervAssignedBeneficiaryFlag
		case pervAssignedBeneficiaryFlagP.MatchString(k):

			// ext - voluntaryAlignmentFlag
		case voluntaryAlignmentFlagP.MatchString(k):

		default:
			// If 99 is returned, we could not match and log as error
		}

	}

	return group
}
