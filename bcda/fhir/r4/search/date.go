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

	d.Datetimes = strings.Split(afterPrefix, ",")
	if len(d.Datetimes) == 0 {
		return d, ParameterParsingError{Details: fmt.Sprintf("value must include a valid FHIR date for date parameter %s", d.Name)}
	}

	var datetimeFormats = []string{
		"2006",
		"2006-01",
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05+07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z",
	}

	for _, datetime := range d.Datetimes {
		valid := false
		for _, format := range datetimeFormats {
			if _, err := time.Parse(format, datetime); err == nil {
				valid = true
				break
			}
		}
		if !valid {
			return d, ParameterParsingError{Details: fmt.Sprintf("invalid date parameter value: %s", datetime)}
		}
	}
	return d, nil
}

func ValidateServiceDates(serviceDateParams []DateParam) error {
	lowerBoundCount, upperBoundCount, equalCount := 0, 0, 0
	for _, sd := range serviceDateParams {
		if len(sd.Datetimes) > 1 {
			return ParameterValidationError{Details: "invalid service-date parameter value: comma-separated values are not supported"}
		}
		switch sd.Prefix {
		case "lt", "le":
			upperBoundCount = upperBoundCount + 1
		case "gt", "ge":
			lowerBoundCount = lowerBoundCount + 1
		case "eq":
			equalCount = equalCount + 1
		}
	}
	if lowerBoundCount > 1 || upperBoundCount > 1 || equalCount > 1 {
		return ParameterParsingError{Details: "invalid service-date parameter value: conflicting prefix conditions"}
	}
	return nil
}
