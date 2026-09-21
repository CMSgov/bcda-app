package fhir

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/CMSgov/bcda-app/bcda/constants"
)

type TypeFilterSubquery struct {
	ResourceType    string
	QueryParameters []TypeFilterSubqueryParam
}

type TypeFilterSubqueryParam struct {
	Name  string
	Value string
}

type TypeFilterParamName string

const (
	TypeFilterParamOutcome     TypeFilterParamName = "outcome"
	TypeFilterParamServiceDate TypeFilterParamName = "service-date"
	TypeFilterParamTag         TypeFilterParamName = "_tag"
)

type ParameterParsingError struct {
	Details string
}

func (e ParameterParsingError) Error() string {
	return fmt.Sprintf("malformed parameter: %s", e.Details)
}

type ParameterValidationError struct {
	Details string
}

func (e ParameterValidationError) Error() string {
	return fmt.Sprintf("invalid parameter: %s", e.Details)
}

func ParseTypeFilterSubquery(s string) (TypeFilterSubquery, error) {
	// The subquery is url-encoded. So we will first decode so we can parse it
	decodedQuery, err := url.QueryUnescape(s)
	if err != nil {
		return TypeFilterSubquery{}, ParameterParsingError{Details: fmt.Sprintf("failed to unescape %s", s)}
	}

	// Expected format is: <resourceType>?<paramList>
	resourceType, params, found := strings.Cut(decodedQuery, "?")
	if !found {
		return TypeFilterSubquery{}, ParameterParsingError{Details: fmt.Sprintf("_typeFilter parameter missing question mark %s", decodedQuery)}
	} else if len(resourceType) == 0 {
		return TypeFilterSubquery{}, ParameterParsingError{Details: fmt.Sprintf("_typeFilter parameter missing resource type %s", decodedQuery)}
	} else if len(params) == 0 {
		return TypeFilterSubquery{}, ParameterParsingError{Details: fmt.Sprintf("_typeFilter parameter missing value %s", decodedQuery)}
	}

	var subqueryParams []TypeFilterSubqueryParam
	paramAry := strings.SplitSeq(params, "&")
	for paramPair := range paramAry {
		name, value, found := strings.Cut(paramPair, "=")
		if !found {
			return TypeFilterSubquery{}, ParameterParsingError{Details: fmt.Sprintf("_typeFilter value missing equals sign: %s", paramPair)}
		}
		subqueryParams = append(subqueryParams, TypeFilterSubqueryParam{Name: name, Value: value})
	}
	return TypeFilterSubquery{ResourceType: resourceType, QueryParameters: subqueryParams}, nil
}

func ValidateTypeFilterSubquery(subquery TypeFilterSubquery) error {
	if subquery.ResourceType != "ExplanationOfBenefit" {
		return fmt.Errorf("invalid _typeFilter Resource Type (Only EOBs valid): %s", subquery.ResourceType)
	}

	var serviceDateParams []DateParam
	for _, param := range subquery.QueryParameters {
		if strings.HasPrefix(param.Name, string(TypeFilterParamTag)) {
			tag, err := ParseTokenParam(param)
			if err != nil {
				return err
			}
			err = ValidateTag(tag)
			if err != nil {
				return err
			}
		} else if strings.HasPrefix(param.Name, string(TypeFilterParamServiceDate)) {
			date, err := ParseDateParam(param)
			if err != nil {
				return err
			}
			serviceDateParams = append(serviceDateParams, date)
		} else if strings.HasPrefix(param.Name, string(TypeFilterParamOutcome)) {
			outcome, err := ParseTokenParam(param)
			if err != nil {
				return err
			}
			err = ValidateOutcome(outcome)
			if err != nil {
				return err
			}
		} else {
			return ParameterValidationError{Details: fmt.Sprintf("invalid _typeFilter subquery parameter: %s", param.Name)}
		}
	}
	err := ValidateServiceDates(serviceDateParams)
	if err != nil {
		return err
	}
	return nil
}

func GetTagParams(subquery TypeFilterSubquery) ([]TokenParam, error) {
	tagParams := []TokenParam{}
	for _, param := range subquery.QueryParameters {
		if strings.HasPrefix(param.Name, string(TypeFilterParamTag)) {
			tag, err := ParseTokenParam(param)
			if err != nil {
				return nil, err
			}
			tagParams = append(tagParams, tag)
		}
	}
	return tagParams, nil
}

// https://hl7.org/fhir/R4/search.html#string
type StringParam struct {
	Name     string
	Modifier string
	Values   []string
}

func ParseStringParam(subqueryParam TypeFilterSubqueryParam) (StringParam, error) {
	s := StringParam{}
	s.Name, s.Modifier, _ = strings.Cut(subqueryParam.Name, ":")
	if len(s.Name) == 0 {
		return s, ParameterParsingError{Details: "string parameter missing name"}
	}
	if len(subqueryParam.Value) == 0 {
		return s, ParameterParsingError{Details: fmt.Sprintf("string parameter missing value: %s", s.Name)}
	}
	s.Values = strings.Split(subqueryParam.Value, ",")
	return s, nil
}

// https://hl7.org/fhir/R4/search.html#date
type DateParam struct {
	Name      string
	Prefix    string
	Datetimes []string
}

func ParseDateParam(subqueryParam TypeFilterSubqueryParam) (DateParam, error) {
	d := DateParam{}
	if len(subqueryParam.Name) == 0 {
		return d, ParameterParsingError{Details: "date parameter missing key"}
	}
	d.Name = subqueryParam.Name

	if len(subqueryParam.Value) == 0 {
		return d, ParameterParsingError{Details: fmt.Sprintf("date parameter missing value: %s", d.Name)}
	}

	// BFD only supports eq, ge, gt, lt, le as of 2026-08-17. See: https://cmsgov.slack.com/archives/CMT1YS2KY/p1786716165942379
	var prefixes = []string{"eq", "lt", "gt", "le", "ge"} // "ne", "sa", "eb", "ap"}
	afterPrefix := ""
	if len(subqueryParam.Value) >= 2 && slices.Contains(prefixes, subqueryParam.Value[:2]) {
		d.Prefix = subqueryParam.Value[:2]
		afterPrefix = subqueryParam.Value[2:]
	} else {
		afterPrefix = subqueryParam.Value
	}

	if len(afterPrefix) == 0 {
		return d, ParameterParsingError{Details: fmt.Sprintf("value must include a valid FHIR date for date parameter %s", d.Name)}
	}

	d.Datetimes = strings.Split(afterPrefix, ",")

	for _, datetime := range d.Datetimes {
		_, err := ParseDateString(datetime)
		if err != nil {
			return d, ParameterParsingError{Details: fmt.Sprintf("invalid date parameter value: %s", datetime)}
		}
	}
	return d, nil
}

func ParseDateString(datetime string) (time.Time, error) {
	var datetimeFormats = []string{
		"2006",
		"2006-01",
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05+07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
	}
	for _, format := range datetimeFormats {
		if parsed, err := time.Parse(format, datetime); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, ParameterParsingError{Details: fmt.Sprintf("invalid date parameter value: %s", datetime)}
}

func ValidateServiceDates(serviceDateParams []DateParam) error {
	earlyBoundCount, lateBoundCount, equalCount := 0, 0, 0
	for _, sd := range serviceDateParams {
		if sd.Name != string(TypeFilterParamServiceDate) {
			return ParameterValidationError{Details: fmt.Sprintf("invalid key for service-date parameter: %s", sd.Name)}
		}
		if len(sd.Datetimes) > 1 {
			return ParameterValidationError{Details: "invalid service-date parameter value: comma-separated values are not supported"}
		}
		switch sd.Prefix {
		case "lt", "le":
			lateBoundCount = lateBoundCount + 1
		case "gt", "ge":
			earlyBoundCount = earlyBoundCount + 1
		case "eq", "":
			equalCount = equalCount + 1
		}
	}
	if earlyBoundCount > 1 || lateBoundCount > 1 || equalCount > 1 {
		return ParameterValidationError{Details: "invalid service-date parameter value: conflicting prefix conditions"}
	}
	return nil
}

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

func ValidateOutcome(t TokenParam) error {
	if t.Name != string(TypeFilterParamOutcome) {
		return ParameterValidationError{Details: fmt.Sprintf("invalid key for outcome parameter: %s", t.Name)}
	}
	if len(t.Modifier) > 0 {
		return ParameterValidationError{Details: fmt.Sprintf("invalid outcome parameter; modifier %s not supported", t.Modifier)}
	}
	for _, value := range t.Values {
		if value.Code != "complete" && value.Code != "partial" {
			return ParameterValidationError{Details: fmt.Sprintf("invalid outcome code: %s. Supported outcome codes are 'complete' and 'partial'", value.Code)}
		}
	}
	return nil
}
