package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseStringParam(t *testing.T) {
	tests := []struct {
		name           string
		subqueryParam  TypeFilterSubqueryParam
		expectedString StringParam
		expectedErr    bool
	}{
		{
			name:           "any patients with a name containing a given part with eve at the start of the name",
			subqueryParam:  TypeFilterSubqueryParam{Name: "given", Value: "eve"},
			expectedString: StringParam{Name: "given", Modifier: "", Values: []string{"eve"}},
			expectedErr:    false,
		},
		{
			name:           "any patients with a name with a given part containing eve at any position",
			subqueryParam:  TypeFilterSubqueryParam{Name: "given:contains", Value: "eve"},
			expectedString: StringParam{Name: "given", Modifier: "contains", Values: []string{"eve"}},
			expectedErr:    false,
		},
		{
			name:           "any patients with a name with a given part that is exactly Eve",
			subqueryParam:  TypeFilterSubqueryParam{Name: "given:exact", Value: "Eve"},
			expectedString: StringParam{Name: "given", Modifier: "exact", Values: []string{"Eve"}},
			expectedErr:    false,
		},
		{
			name:          "invalid empty value",
			subqueryParam: TypeFilterSubqueryParam{Name: "given", Value: ""},
			expectedErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stringParam, err := ParseStringParam(tt.subqueryParam)

			if tt.expectedErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.expectedString, stringParam)
			}
		})
	}
}

func TestValidateOutcome(t *testing.T) {
	tests := []struct {
		name        string
		stringParam StringParam
		expectedErr bool
	}{
		{
			name:        "valid complete",
			stringParam: StringParam{Name: "outcome", Values: []string{"complete"}},
			expectedErr: false,
		},
		{
			name:        "valid partial",
			stringParam: StringParam{Name: "outcome", Values: []string{"partial"}},
			expectedErr: false,
		},
		{
			name:        "invalid value",
			stringParam: StringParam{Name: "outcome", Values: []string{"somethingelse"}},
			expectedErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutcome(tt.stringParam)

			if tt.expectedErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
