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

// SupportsDataType checks if the dataType is supported by the instanced DataType object
func (r ClaimType) SupportsDataType(dataType string) bool {
	switch dataType {
	case constants.PartiallyAdjudicated:
		return r.PartiallyAdjudicated
	case constants.Adjudicated:
		return r.Adjudicated
	default:
		return false
	}
}

// GetDataType gets the DataType associated with the given resourceName
func GetDataType(resourceName string) (ClaimType, bool) {
	resource, ok := fhirResourceTypeMap[resourceName]

	return resource, ok
}

// GetDataTypes creates a map of the given resourceNames with their associated DataType objects
// It returns the resource map and a status flag, with true meaning all resources were found and mapped.
func GetDataTypes(resourceNames ...string) (map[string]ClaimType, bool) {
	foundAll := true

	returnMap := make(map[string]ClaimType, len(resourceNames))

	if len(resourceNames) == 0 {
		// If no resource specified, return copy of full map
		for name, entry := range fhirResourceTypeMap {
			returnMap[name] = entry
		}
	} else {
		// If resources specified, return map subset
		for _, name := range resourceNames {
			if entry, ok := fhirResourceTypeMap[name]; ok {
				returnMap[name] = entry
			} else {
				foundAll = false
			}
		}
	}

	return returnMap, foundAll
}
