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

var validTagTokens = map[string][]string{
	constants.BFDSystemTypeURL:  {"SharedSystem", "NationalClaimsHistory", "DDPS"},
	constants.BFDFinalActionURL: {"FinalAction", "NotFinalAction"},
}

func ValidateTag(t TokenParam) error {
	if t.Name != string(TypeFilterParamTag) {
		return ParameterValidationError{Details: fmt.Sprintf("invalid key for tag parameter: %s", t.Name)}
	}
	if len(t.Modifier) > 0 {
		return ParameterValidationError{Details: fmt.Sprintf("invalid _tag parameter; modifier %s not supported", t.Modifier)}
	}
	for _, value := range t.Values {
		if len(value.System) == 0 || len(value.Code) == 0 {
			return ParameterValidationError{Details: "invalid _tag parameter value. Searching by tag requires a token (system|code) to be specified"}
		}
		validTagCodes, ok := validTagTokens[value.System]
		if !ok || !slices.Contains(validTagCodes, value.Code) {
			return ParameterValidationError{Details: fmt.Sprintf("invalid _tag parameter, unsupported code %s for system %s", value.Code, value.System)}
		}
	}
	return nil
}
