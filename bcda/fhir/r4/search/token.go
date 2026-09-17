package search

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CMSgov/bcda-app/bcda/constants"
)

// https://hl7.org/fhir/R4/search.html#token
type TokenParam struct {
	Name     string
	Modifier string
	Values   []TokenValue
}

type TokenValue struct {
	System string
	Code   string
}

func ParseTokenParam(subqueryParam TypeFilterSubqueryParam) (TokenParam, error) {
	t := TokenParam{}
	if len(subqueryParam.Name) == 0 {
		return t, ParameterParsingError{Details: "token parameter missing key"}
	}
	t.Name, t.Modifier, _ = strings.Cut(subqueryParam.Name, ":")
	if len(subqueryParam.Value) == 0 {
		return t, ParameterParsingError{Details: fmt.Sprintf("token parameter missing value %s", t.Name)}
	}
	values := strings.SplitSeq(subqueryParam.Value, ",")
	for value := range values {
		before, after, found := strings.Cut(value, "|")
		if found {
			t.Values = append(t.Values, TokenValue{System: before, Code: after})
		} else {
			t.Values = append(t.Values, TokenValue{Code: before})
		}
	}
	return t, nil
}

func ValidateTag(t TokenParam) error {
	for _, value := range t.Values {
		if len(value.System) == 0 || len(value.Code) == 0 {
			return ParameterValidationError{Details: "invalid _tag parameter value. Searching by tag requires a token (system|code) to be specified"}
		}

		validTagTokens := map[string][]string{
			constants.BFDSystemTypeURL:  {"SharedSystem", "NationalClaimsHistory", "DDPS"},
			constants.BFDFinalActionURL: {"FinalAction", "NotFinalAction"},
		}

		validTagCodes, ok := validTagTokens[value.System]
		if !ok || !slices.Contains(validTagCodes, value.Code) {
			return ParameterValidationError{Details: "invalid _tag parameter. Searching by tag requires a token (system|code) to be specified"}
		}
	}
	return nil
}
