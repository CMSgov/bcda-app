package search

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

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
