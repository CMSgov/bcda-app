package search

import (
	"fmt"
	"strings"
)

// https://hl7.org/fhir/R4/search.html#string
type StringParam struct {
	Name     string
	Modifier string
	Values   []string
}

func ParseStringParam(subqueryParam TypeFilterSubqueryParam) (StringParam, error) {
	s := StringParam{}
	if len(subqueryParam.Name) == 0 {
		return s, ParameterParsingError{Details: "string parameter missing key"}
	}
	s.Name, s.Modifier, _ = strings.Cut(subqueryParam.Name, ":")
	if len(subqueryParam.Value) == 0 {
		return s, ParameterParsingError{Details: fmt.Sprintf("string parameter missing value: %s", s.Name)}
	}
	s.Values = strings.Split(subqueryParam.Value, ",")
	return s, nil
}

func ValidateOutcome(s StringParam) error {
	if s.Name != string(TypeFilterParamOutcome) {
		return ParameterValidationError{Details: fmt.Sprintf("invalid key for outcome parameter: %s", s.Name)}
	}
	if len(s.Modifier) > 0 {
		return ParameterValidationError{Details: fmt.Sprintf("invalid outcome parameter; modifier %s not supported", s.Modifier)}
	}
	for _, value := range s.Values {
		if value != "complete" && value != "partial" {
			return ParameterValidationError{Details: fmt.Sprintf("invalid outcome value: %s. Supported outcome values are 'complete' and 'partial'", value)}
		}
	}
	return nil
}
