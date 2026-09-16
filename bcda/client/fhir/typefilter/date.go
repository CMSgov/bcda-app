package typefilter

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

// https://hl7.org/fhir/R4/search.html#date
type DateParam struct {
	Name     string
	Prefix   string
	Datetime time.Time
	raw      string
}

func (d DateParam) String() string {
	return d.raw
}

func ParseDate(subqueryParam SubqueryParam) (DateParam, error) {
	d := DateParam{raw: fmt.Sprintf("%s=%s", subqueryParam.Name, subqueryParam.Value)}
	if len(subqueryParam.Name) == 0 {
		return d, errors.New("key must be present in typefilter parameters")
	}
	d.Name = subqueryParam.Name

	if len(subqueryParam.Value) == 0 {
		return d, fmt.Errorf("value must be present for date param %s", d.Name)
	}

	// BFD only supports eq, ge, gt, lt, le as of 2026-08-17. See: https://cmsgov.slack.com/archives/CMT1YS2KY/p1786716165942379
	var prefixes = []string{"eq", "lt", "gt", "le", "ge"} // "ne", "sa", "eb", "ap"}
	dtString := ""
	if slices.Contains(prefixes, subqueryParam.Value[:2]) {
		d.Prefix = subqueryParam.Value[:2]
		dtString = subqueryParam.Value[2:]
	} else {
		dtString = subqueryParam.Value
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
	for _, format := range datetimeFormats {
		if datetime, err := time.Parse(format, dtString); err == nil {
			d.Datetime = datetime
			return d, nil
		}
	}
	return d, fmt.Errorf("invalid date parameter value: %s. Pass a valid FHIR date parameter", subqueryParam.Value)
}

const SERVICE_DATE = "service-date"

func ValidateServiceDates(serviceDateParams []DateParam) error {
	return nil
}
