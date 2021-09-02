package service

import "github.com/CMSgov/bcda-app/bcda/constants"

//DataType is used to identify the type of data returned by each resource
type DataType struct {
	Adjudicated    bool
	PreAdjudicated bool
}

var dataTypeMap map[string]DataType

// SupportsDataType checks if the dataType is supported by the instanced DataType object
func (r DataType) SupportsDataType(dataType string) bool {
	switch dataType {
	case constants.PreAdjudicated:
		return r.PreAdjudicated
	case constants.Adjudicated:
		return r.Adjudicated
	default:
		return false
	}
}

// init creates a map of ResourceType => DataType configurations to be used by later functions
func init() {
	dataTypeMap = map[string]DataType{
		"Patient":              {Adjudicated: true},
		"Coverage":             {Adjudicated: true},
		"ExplanationOfBenefit": {Adjudicated: true},
		"Observation":          {Adjudicated: true},
		"Claim":                {PreAdjudicated: true},
		"ClaimResponse":        {PreAdjudicated: true},
	}
}

// GetDataType gets the DataType associated with the given resourceName
func GetDataType(resourceName string) (DataType, bool) {
	resource, ok := dataTypeMap[resourceName]

	return resource, ok
}

// GetDataTypes creates a map of the given resourceNames with their associated DataType objects
// It returns the resource map and a status flag, with true meaning all resources were found and mapped.
func GetDataTypes(resourceNames ...string) (map[string]DataType, bool) {
	foundAll := true

	returnMap := make(map[string]DataType, len(resourceNames))

	if len(resourceNames) == 0 {
		// If no resource specified, return copy of full map
		for name, entry := range dataTypeMap {
			returnMap[name] = entry
		}
	} else {
		// If resources specified, return map subset
		for _, name := range resourceNames {
			if entry, ok := dataTypeMap[name]; ok {
				returnMap[name] = entry
			} else {
				foundAll = false
			}
		}
	}

	return returnMap, foundAll
}
