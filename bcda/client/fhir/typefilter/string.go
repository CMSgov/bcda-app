package typefilter

import (
	"errors"
	"fmt"
	"strings"
)

// https://hl7.org/fhir/R4/search.html#string
type StringParam struct {
	Name     string
	Modifier string
	Values   []string
}

func ParseString(subqueryParam SubqueryParam) (StringParam, error) {
	s := StringParam{}
	if len(subqueryParam.Name) == 0 {
		return s, errors.New("keys must be present in typefilter parameter")
	}
	s.Name, s.Modifier, _ = strings.Cut(subqueryParam.Name, ":")
	if len(subqueryParam.Value) == 0 {
		return s, fmt.Errorf("value must be present for string param %s", s.Name)
	}
	s.Values = strings.Split(subqueryParam.Value, ",")
	return s, nil
}

const OUTCOME = "outcome"

func ValidateOutcome(s StringParam) error {
	if len(s.Modifier) > 0 {
		return fmt.Errorf("invalid outcome parameter; modifier %s not supported", s.Modifier)
	}
	for _, value := range s.Values {
		if value != "complete" && value != "partial" {
			return fmt.Errorf("invalid outcome value: %s. Supported outcome values are 'complete' and 'partial'", value)
		}
	}
	return nil
}
