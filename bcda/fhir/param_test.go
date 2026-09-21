package fhir

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/stretchr/testify/assert"
)

func TestParseStringParam(t *testing.T) {
	tests := []struct {
		name           string
		subqueryParam  TypeFilterSubqueryParam
		expectedString StringParam
		expectedErr    error
	}{
		{
			name:           "any patients with a name containing a given part with eve at the start of the name",
			subqueryParam:  TypeFilterSubqueryParam{Name: "given", Value: "eve"},
			expectedString: StringParam{Name: "given", Modifier: "", Values: []string{"eve"}},
			expectedErr:    nil,
		},
		{
			name:           "any patients with a name with a given part containing eve at any position",
			subqueryParam:  TypeFilterSubqueryParam{Name: "given:contains", Value: "eve"},
			expectedString: StringParam{Name: "given", Modifier: "contains", Values: []string{"eve"}},
			expectedErr:    nil,
		},
		{
			name:           "any patients with a name with a given part that is exactly Eve",
			subqueryParam:  TypeFilterSubqueryParam{Name: "given:exact", Value: "Eve"},
			expectedString: StringParam{Name: "given", Modifier: "exact", Values: []string{"Eve"}},
			expectedErr:    nil,
		},
		{
			name:          "invalid empty value",
			subqueryParam: TypeFilterSubqueryParam{Name: "given", Value: ""},
			expectedErr:   ParameterParsingError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stringParam, err := ParseStringParam(tt.subqueryParam)

			if tt.expectedErr != nil {
				assert.ErrorAs(t, err, &tt.expectedErr)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedString, stringParam)
			}
		})
	}
}

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

func TestParseTokenParam(t *testing.T) {
	tests := []struct {
		name          string
		subqueryParam TypeFilterSubqueryParam
		expectedToken TokenParam
		expectedErr   error
	}{
		{
			name:          "search for all the patients with an identifier with key = 2345 in the system http://acme.org/patient",
			subqueryParam: TypeFilterSubqueryParam{Name: "identifier", Value: "http://acme.org/patient|2345"},
			expectedToken: TokenParam{Name: "identifier", Modifier: "", Values: []TokenValue{{System: "http://acme.org/patient", Code: "2345"}}},
			expectedErr:   nil,
		},
		{
			name:          "search for any patient with a gender that does not have the code male",
			subqueryParam: TypeFilterSubqueryParam{Name: "gender:not", Value: "male"},
			expectedToken: TokenParam{Name: "gender", Modifier: "not", Values: []TokenValue{{System: "", Code: "male"}}},
			expectedErr:   nil,
		},
		{
			name:          "token with code that has no system property",
			subqueryParam: TypeFilterSubqueryParam{Name: "testname", Value: "|testcode"},
			expectedToken: TokenParam{Name: "testname", Modifier: "", Values: []TokenValue{{System: "", Code: "testcode"}}},
			expectedErr:   nil,
		},
		{
			name:          "token where any element of the system value matches system property of identifier or coding",
			subqueryParam: TypeFilterSubqueryParam{Name: "testname", Value: "testsystem|"},
			expectedToken: TokenParam{Name: "testname", Modifier: "", Values: []TokenValue{{System: "testsystem", Code: ""}}},
			expectedErr:   nil,
		},
		{
			name:          "token with no key",
			subqueryParam: TypeFilterSubqueryParam{Name: "", Value: "male"},
			expectedErr:   ParameterParsingError{},
		},
		{
			name:          "token with no value",
			subqueryParam: TypeFilterSubqueryParam{Name: "testname", Value: ""},
			expectedErr:   ParameterParsingError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ParseTokenParam(tt.subqueryParam)

			if tt.expectedErr != nil {
				assert.ErrorAs(t, err, &tt.expectedErr)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedToken, token)
			}
		})
	}
}

func TestValidateTag(t *testing.T) {
	tests := []struct {
		name        string
		tokenParam  TokenParam
		expectedErr error
	}{
		{
			name:        "valid system and code",
			tokenParam:  TokenParam{Name: "_tag", Values: []TokenValue{{System: constants.BFDSystemTypeURL, Code: "NationalClaimsHistory"}}},
			expectedErr: nil,
		},
		{
			name:        "invalid system",
			tokenParam:  TokenParam{Name: "_tag", Values: []TokenValue{{System: "https://example.com/fhir/CodeSystem/12345", Code: "12345"}}},
			expectedErr: ParameterValidationError{},
		},
		{
			name:        "invalid code",
			tokenParam:  TokenParam{Name: "_tag", Values: []TokenValue{{System: constants.BFDSystemTypeURL, Code: "12345"}}},
			expectedErr: ParameterValidationError{},
		},
		{
			name:        "code does not match system",
			tokenParam:  TokenParam{Name: "_tag", Values: []TokenValue{{System: constants.BFDSystemTypeURL, Code: "NotFinalAction"}}},
			expectedErr: ParameterValidationError{},
		},
		{
			name:        "invalid modifier",
			tokenParam:  TokenParam{Name: "_tag", Modifier: "not", Values: []TokenValue{{System: constants.BFDSystemTypeURL, Code: "NationalClaimsHistory"}}},
			expectedErr: ParameterValidationError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTag(tt.tokenParam)

			if tt.expectedErr != nil {
				assert.ErrorAs(t, err, &tt.expectedErr)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestValidateOutcome(t *testing.T) {
	tests := []struct {
		name        string
		tokenParam  TokenParam
		expectedErr error
	}{
		{
			name:        "valid complete",
			tokenParam:  TokenParam{Name: "outcome", Values: []TokenValue{{Code: "complete"}}},
			expectedErr: nil,
		},
		{
			name:        "valid partial",
			tokenParam:  TokenParam{Name: "outcome", Values: []TokenValue{{Code: "partial"}}},
			expectedErr: nil,
		},
		{
			name:        "invalid value",
			tokenParam:  TokenParam{Name: "outcome", Values: []TokenValue{{Code: "somethingelse"}}},
			expectedErr: ParameterValidationError{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutcome(tt.tokenParam)

			if tt.expectedErr != nil {
				assert.ErrorAs(t, err, &tt.expectedErr)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
