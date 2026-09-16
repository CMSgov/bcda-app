package typefilter

import (
	"errors"
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

func ParseDate(subqueryParam SubqueryParam) (DateParam, error) {
	d := DateParam{}
	if len(subqueryParam.Name) == 0 {
		return d, errors.New("key must be present in date parameter")
	}
	d.Name = subqueryParam.Name

	if len(subqueryParam.Value) == 0 {
		return d, fmt.Errorf("value must be present for date parameter %s", d.Name)
	}

	// BFD only supports eq, ge, gt, lt, le as of 2026-08-17. See: https://cmsgov.slack.com/archives/CMT1YS2KY/p1786716165942379
	var prefixes = []string{"eq", "lt", "gt", "le", "ge"} // "ne", "sa", "eb", "ap"}
	afterPrefix := ""
	if slices.Contains(prefixes, subqueryParam.Value[:2]) {
		d.Prefix = subqueryParam.Value[:2]
		afterPrefix = subqueryParam.Value[2:]
	} else {
		afterPrefix = subqueryParam.Value
	}

	d.Datetimes = strings.Split(afterPrefix, ",")
	if len(d.Datetimes) == 0 {
		return d, fmt.Errorf("value must include a valid FHIR date for date parameter %s", d.Name)
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
			return d, fmt.Errorf("invalid date parameter value: %s. Pass a valid FHIR date parameter", datetime)
		}
	}
	return d, nil
}

const SERVICE_DATE = "service-date"

func ValidateServiceDates(serviceDateParams []DateParam) error {
	return nil
}
