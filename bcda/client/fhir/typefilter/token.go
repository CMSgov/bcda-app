package typefilter

import (
	"errors"
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
	raw      string
}

func (t TokenParam) String() string {
	return t.raw
}

type TokenValue struct {
	System string
	Code   string
}

func ParseToken(subqueryParam SubqueryParam) (TokenParam, error) {
	t := TokenParam{raw: fmt.Sprintf("%s=%s", subqueryParam.Name, subqueryParam.Value)}
	if len(subqueryParam.Name) == 0 {
		return t, errors.New("keys must be present in typefilter parameters")
	}
	t.Name, t.Modifier, _ = strings.Cut(subqueryParam.Name, ":")
	if len(subqueryParam.Value) == 0 {
		return t, fmt.Errorf("value must be present for token param %s", t.Name)
	}
	values := strings.SplitSeq(subqueryParam.Value, ",")
	for value := range values {
		system, code, _ := strings.Cut(value, "|")
		t.Values = append(t.Values, TokenValue{System: system, Code: code})
	}
	return t, nil
}

const TAG = "_tag"

func ValidateTag(t TokenParam) error {
	for _, value := range t.Values {
		if len(value.System) == 0 || len(value.Code) == 0 {
			return fmt.Errorf("invalid _tag parameter: %s. Searching by tag requires a token (system|code) to be specified", t)
		}

		validTagTokens := map[string][]string{
			constants.BFDSystemTypeURL:  {"SharedSystem", "NationalClaimsHistory", "DDPS"},
			constants.BFDFinalActionURL: {"FinalAction", "NotFinalAction"},
		}

		validTagCodes, ok := validTagTokens[value.System]
		if !ok || !slices.Contains(validTagCodes, value.Code) {
			return fmt.Errorf("invalid _tag parameter: %s. Searching by tag requires a token (system|code) to be specified", t)
		}
	}
	return nil
}
