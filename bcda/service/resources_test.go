package service

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ResourcesTestSuite struct {
	suite.Suite
}

// Run all suite tests
func TestResourcesTestSuite(t *testing.T) {
	suite.Run(t, new(ResourcesTestSuite))
}

func (s *ResourcesTestSuite) TestSupportsDataType() {
	tests := []struct {
		name         string
		dataType     DataType
		dataTypeName string
		expected     bool
	}{
		{
			"Valid Adjudicated Type",
			DataType{Adjudicated: true, PartiallyAdjudicated: false},
			constants.Adjudicated,
			true,
		},
		{
			"Valid Partially-Adjudicated Type",
			DataType{Adjudicated: false, PartiallyAdjudicated: true},
			constants.PartiallyAdjudicated,
			true,
		},
		{
			"Invalid Type",
			DataType{Adjudicated: true, PartiallyAdjudicated: true},
			"invalid-type",
			false,
		},
		{
			"Invalid Partially-Adjudicated Type",
			DataType{Adjudicated: true, PartiallyAdjudicated: false},
			constants.PartiallyAdjudicated,
			false,
		},
		{
			"Invalid Adjudicated Type",
			DataType{Adjudicated: false, PartiallyAdjudicated: true},
			constants.Adjudicated,
			false,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.dataType.SupportsDataType(tt.dataTypeName))
		})
	}
}

func (s *ResourcesTestSuite) TestGetDataType() {
	tests := []struct {
		resourceName string
		expectedType DataType
		expectedOk   bool
	}{
		{"Patient", DataType{Adjudicated: true}, true},
		{"Coverage", DataType{Adjudicated: true}, true},
		{"ExplanationOfBenefit", DataType{Adjudicated: true}, true},
		{"Observation", DataType{Adjudicated: true}, true},
		{"Claim", DataType{Adjudicated: false, PartiallyAdjudicated: true}, true},
		{"ClaimResponse", DataType{Adjudicated: false, PartiallyAdjudicated: true}, true},
		{"InvalidResource", DataType{}, false},
	}

	for _, tt := range tests {
		s.T().Run("Testing "+tt.resourceName, func(t *testing.T) {
			actualType, actualOk := GetDataType(tt.resourceName)
			assert.Equal(t, tt.expectedType, actualType)
			assert.Equal(t, tt.expectedOk, actualOk)
		})
	}
}

func (s *ResourcesTestSuite) TestGetDataTypes() {
	tests := []struct {
		name          string
		resourceNames []string
		expectedTypes map[string]DataType
		expectedOk    bool
	}{
		{
			"Empty resource names",
			[]string{},
			map[string]DataType{
				"Patient":              {Adjudicated: true},
				"Coverage":             {Adjudicated: true},
				"ExplanationOfBenefit": {Adjudicated: true},
				"Observation":          {Adjudicated: true},
				"Claim":                {Adjudicated: false, PartiallyAdjudicated: true},
				"ClaimResponse":        {Adjudicated: false, PartiallyAdjudicated: true},
			},
			true,
		},
		{
			"Valid resource names",
			[]string{"Patient", "Claim"},
			map[string]DataType{
				"Patient": {Adjudicated: true},
				"Claim":   {Adjudicated: false, PartiallyAdjudicated: true},
			},
			true,
		},
		{
			"One invalid resource names",
			[]string{"Patient", "InvalidResource", "Claim"},
			map[string]DataType{
				"Patient": {Adjudicated: true},
				"Claim":   {Adjudicated: false, PartiallyAdjudicated: true},
			},
			false,
		},
		{
			"All invalid resource names",
			[]string{"InvalidResource1", "InvalidResource2"},
			map[string]DataType{},
			false,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			actualTypes, actualOk := GetDataTypes(tt.resourceNames...)
			assert.Equal(t, tt.expectedTypes, actualTypes)
			assert.Equal(t, tt.expectedOk, actualOk)
		})
	}
}
