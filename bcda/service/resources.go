package service

import "github.com/CMSgov/bcda-app/bcda/constants"

// ClaimType is used to identify the type of data returned by each resource
type ClaimType struct {
	Adjudicated          bool
	PartiallyAdjudicated bool
}

var fhirResourceTypeMap = map[string]ClaimType{
	"Patient":              {Adjudicated: true},
	"Coverage":             {Adjudicated: true},
	"ExplanationOfBenefit": {Adjudicated: true},
	"Observation":          {Adjudicated: true},
	"Claim":                {PartiallyAdjudicated: true},
	"ClaimResponse":        {PartiallyAdjudicated: true},
}

// SupportsClaimType checks if the dataType is supported by the instanced DataType object
func (r ClaimType) SupportsClaimType(claimType string) bool {
	switch claimType {
	case constants.PartiallyAdjudicated:
		return r.PartiallyAdjudicated
	case constants.Adjudicated:
		return r.Adjudicated
	default:
		return false
	}
}

// GetClaimType gets the claim type associated with the given fhir resource
func GetClaimType(fhirResource string) (ClaimType, bool) {
	resource, ok := fhirResourceTypeMap[fhirResource]

	return resource, ok
}

// GetClaimTypesMap creates a map of the given fhir resources with their associated claim type objects
// It returns the resource map and a status flag, with true meaning all resources were found and mapped.
func GetClaimTypesMap(fhirResource ...string) (map[string]ClaimType, bool) {
	foundAll := true

	returnMap := make(map[string]ClaimType, len(fhirResource))

	if len(fhirResource) == 0 {
		// If no resource specified, return copy of full map
		for name, entry := range fhirResourceTypeMap {
			returnMap[name] = entry
		}
	} else {
		// If resources specified, return map subset
		for _, name := range fhirResource {
			if entry, ok := fhirResourceTypeMap[name]; ok {
				returnMap[name] = entry
			} else {
				foundAll = false
			}
		}
	}

	return returnMap, foundAll
}
