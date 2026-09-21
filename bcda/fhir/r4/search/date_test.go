package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDateParam(t *testing.T) {
	tests := []struct {
		name          string
		subqueryParam TypeFilterSubqueryParam
		expectedDate  DateParam
		expectedErr   error
	}{
		// Valid date
		{
			name:          "year only",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024"}},
			expectedErr:   nil,
		},
		{
			name:          "year month",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01"}},
			expectedErr:   nil,
		},
		{
			name:          "full date",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15"}},
			expectedErr:   nil,
		},
		{
			name:          "date time no timezone",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00"}},
			expectedErr:   nil,
		},
		{
			name:          "date time with Z",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00Z"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00Z"}},
			expectedErr:   nil,
		},
		{
			name:          "date time minus offset",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00-05:00"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00-05:00"}},
			expectedErr:   nil,
		},
		{
			name:          "date time plus offset",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00+05:00"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00+05:00"}},
			expectedErr:   nil,
		},
		{
			name:          "greater than date",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "gt2024-01-15"},
			expectedDate:  DateParam{Name: "date", Prefix: "gt", Datetimes: []string{"2024-01-15"}},
			expectedErr:   nil,
		},
		{
			name:          "less than date time",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "lt2024-01-15T10:30:00Z"},
			expectedDate:  DateParam{Name: "date", Prefix: "lt", Datetimes: []string{"2024-01-15T10:30:00Z"}},
			expectedErr:   nil,
		},
		{
			name:          "equals year",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "eq2024"},
			expectedDate:  DateParam{Name: "date", Prefix: "eq", Datetimes: []string{"2024"}},
			expectedErr:   nil,
		},

		// Invalid date
		{
			name:          "empty",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: ""},
			expectedErr:   ParameterParsingError{},
		},
		{
			name:          "invalid month",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-13-01"},
			expectedErr:   ParameterParsingError{},
		},
		{
			name:          "invalid day",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-12-33"},
			expectedErr:   ParameterParsingError{},
		},
		{
			name:          "unsupported format",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "01-15-2024"},
			expectedErr:   ParameterParsingError{},
		},
		{
			name:          "random string",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "randomstring"},
			expectedErr:   ParameterParsingError{},
		},
		{
			name:          "invalid prefix",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "xx2024-01-15"},
			expectedErr:   ParameterParsingError{},
		},
		{
			name:          "prefix only",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "eq"},
			expectedErr:   ParameterParsingError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDateParam(tt.subqueryParam)

			if tt.expectedErr != nil {
				assert.ErrorAs(t, err, &tt.expectedErr)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedDate, date)
			}
		})
	}
}

func TestValidateServiceDates(t *testing.T) {
	tests := []struct {
		name        string
		dateParams  []DateParam
		expectedErr error
	}{
		{
			name:        "valid service date less-than date time",
			dateParams:  []DateParam{{Name: "service-date", Prefix: "lt", Datetimes: []string{"2024-01-15T10:30:00Z"}}},
			expectedErr: nil,
		},
		{
			name: "valid upper and lower bounds",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "lt", Datetimes: []string{"2005"}},
				{Name: "service-date", Prefix: "gt", Datetimes: []string{"2004"}},
			},
			expectedErr: nil,
		},
		{
			name:        "invalid multiple OR date times",
			dateParams:  []DateParam{{Name: "service-date", Datetimes: []string{"2004", "2003"}}},
			expectedErr: ParameterValidationError{},
		},
		{
			name: "invalid multiple upper bounds",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "lt", Datetimes: []string{"2004"}},
				{Name: "service-date", Prefix: "lt", Datetimes: []string{"2005"}},
			},
			expectedErr: ParameterValidationError{},
		},
		{
			name: "invalid multiple lower bounds",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "gt", Datetimes: []string{"2004"}},
				{Name: "service-date", Prefix: "gt", Datetimes: []string{"2005"}},
			},
			expectedErr: ParameterValidationError{},
		},
		{
			name: "invalid multiple equals",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "eq", Datetimes: []string{"2004"}},
				{Name: "service-date", Prefix: "eq", Datetimes: []string{"2005"}},
			},
			expectedErr: ParameterValidationError{},
		},
		{
			name: "invalid multiple equals (with blank)",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "eq", Datetimes: []string{"2004"}},
				{Name: "service-date", Prefix: "", Datetimes: []string{"2005"}},
			},
			expectedErr: ParameterValidationError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceDates(tt.dateParams)

			if tt.expectedErr != nil {
				assert.ErrorAs(t, err, &tt.expectedErr)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
