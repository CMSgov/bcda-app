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
		expectedErr   bool
	}{
		// Valid date
		{
			name:          "year only",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024"}},
			expectedErr:   false,
		},
		{
			name:          "year month",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01"}},
			expectedErr:   false,
		},
		{
			name:          "full date",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15"}},
			expectedErr:   false,
		},
		{
			name:          "date time no timezone",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00"}},
			expectedErr:   false,
		},
		{
			name:          "date time with Z",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00Z"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00Z"}},
			expectedErr:   false,
		},
		{
			name:          "date time minus offset",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00-05:00"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00-05:00"}},
			expectedErr:   false,
		},
		{
			name:          "date time plus offset",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-01-15T10:30:00+05:00"},
			expectedDate:  DateParam{Name: "date", Prefix: "", Datetimes: []string{"2024-01-15T10:30:00+05:00"}},
			expectedErr:   false,
		},
		{
			name:          "greater than date",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "gt2024-01-15"},
			expectedDate:  DateParam{Name: "date", Prefix: "gt", Datetimes: []string{"2024-01-15"}},
			expectedErr:   false,
		},
		{
			name:          "less than date time",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "lt2024-01-15T10:30:00Z"},
			expectedDate:  DateParam{Name: "date", Prefix: "lt", Datetimes: []string{"2024-01-15T10:30:00Z"}},
			expectedErr:   false,
		},
		{
			name:          "equals year",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "eq2024"},
			expectedDate:  DateParam{Name: "date", Prefix: "eq", Datetimes: []string{"2024"}},
			expectedErr:   false,
		},

		// Invalid date
		{
			name:          "empty",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: ""},
			expectedErr:   true,
		},
		{
			name:          "invalid month",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-13-01"},
			expectedErr:   true,
		},
		{
			name:          "invalid day",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "2024-12-33"},
			expectedErr:   true,
		},
		{
			name:          "unsupported format",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "01-15-2024"},
			expectedErr:   true,
		},
		{
			name:          "random string",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "randomstring"},
			expectedErr:   true,
		},
		{
			name:          "invalid prefix",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "xx2024-01-15"},
			expectedErr:   true,
		},
		{
			name:          "prefix only",
			subqueryParam: TypeFilterSubqueryParam{Name: "date", Value: "eq"},
			expectedErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			date, err := ParseDateParam(tt.subqueryParam)

			if tt.expectedErr {
				assert.NotNil(t, err)
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
		expectedErr bool
	}{
		{
			name:        "valid service date less-than date time",
			dateParams:  []DateParam{{Name: "service-date", Prefix: "lt", Datetimes: []string{"2024-01-15T10:30:00Z"}}},
			expectedErr: false,
		},
		{
			name: "valid upper and lower bounds",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "lt", Datetimes: []string{"2005"}},
				{Name: "service-date", Prefix: "gt", Datetimes: []string{"2004"}},
			},
			expectedErr: false,
		},
		{
			name:        "invalid multiple OR date times",
			dateParams:  []DateParam{{Name: "service-date", Datetimes: []string{"2004", "2003"}}},
			expectedErr: true,
		},
		{
			name: "invalid multiple upper bounds",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "lt", Datetimes: []string{"2004"}},
				{Name: "service-date", Prefix: "lt", Datetimes: []string{"2005"}},
			},
			expectedErr: true,
		},
		{
			name: "invalid multiple lower bounds",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "gt", Datetimes: []string{"2004"}},
				{Name: "service-date", Prefix: "gt", Datetimes: []string{"2005"}},
			},
			expectedErr: true,
		},
		{
			name: "invalid multiple equals",
			dateParams: []DateParam{
				{Name: "service-date", Prefix: "eq", Datetimes: []string{"2004"}},
				{Name: "service-date", Prefix: "eq", Datetimes: []string{"2005"}},
			},
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServiceDates(tt.dateParams)

			if tt.expectedErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
