package search

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/stretchr/testify/assert"
)

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
