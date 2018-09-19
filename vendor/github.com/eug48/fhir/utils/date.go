package utils

import (
	"fmt"
	"strconv"
	"strings"
	"regexp"
	"time"
)


// Date represents a date in a search query.  FHIR search params may define
// dates to varying levels of precision, and the amount of precision affects
// the behavior of the query.  Date's value should only be interpreted in the
// context of the Precision supplied.
type Date struct {
	Value     time.Time
	Precision DatePrecision
}

// String returns a string representation of the date, honoring the supplied
// precision.
func (d *Date) String() string {
	s := d.Value.Format(d.Precision.layout())
	if strings.HasSuffix(s, "+00:00") {
		s = strings.Replace(s, "+00:00", "Z", 1)
	}
	return s
}

// RangeLowIncl represents the low end of a date range to match against.  As
// the name suggests, the low end of the range is inclusive.
func (d *Date) RangeLowIncl() time.Time {
	return d.Value
}

// RangeHighExcl represents the high end of a date range to match against.  As
// the name suggests, the high end of the range is exclusive.
func (d *Date) RangeHighExcl() time.Time {
	switch d.Precision {
	case Year:
		return d.Value.AddDate(1, 0, 0)
	case Month:
		return d.Value.AddDate(0, 1, 0)
	case Day:
		return d.Value.AddDate(0, 0, 1)
	case Minute:
		return d.Value.Add(time.Minute)
	case Second:
		return d.Value.Add(time.Second)
	case Millisecond:
		return d.Value.Add(time.Millisecond)
	default:
		return d.Value.Add(time.Millisecond)
	}
}

func MustParseDate(dateStr string) (out *Date) {
	var err error
	out, err = ParseDate(dateStr)
	if err != nil {
		panic(err)
	}
	return
}

// ParseDate parses a FHIR date string (roughly ISO 8601) into a Date object,
// maintaining the value and the precision supplied.
func ParseDate(dateStr string) (*Date, error) {
	dt := &Date{}

	dateStr = strings.TrimSpace(dateStr)
	dtRegex := regexp.MustCompile("([0-9]{4})(-(0[1-9]|1[0-2])(-(0[0-9]|[1-2][0-9]|3[0-1])(T([01][0-9]|2[0-3]):([0-5][0-9])(:([0-5][0-9])(\\.([0-9]+))?)?((Z)|(\\+|-)((0[0-9]|1[0-3]):([0-5][0-9])|(14):(00)))?)?)?)?")
	if m := dtRegex.FindStringSubmatch(dateStr); m != nil {
		y, mo, d, h, mi, s, ms, tzZu, tzOp, tzh, tzm := m[1], m[3], m[5], m[7], m[8], m[10], m[12], m[14], m[15], m[17], m[18]

		switch {
		case ms != "":
			dt.Precision = Millisecond

			// Fix milliseconds (.9 -> .900, .99 -> .990, .999999 -> .999 )
			switch len(ms) {
			case 1:
				ms += "00"
			case 2:
				ms += "0"
			case 3:
				// do nothing
			default:
				ms = ms[:3]
			}
		case s != "":
			dt.Precision = Second
		case mi != "":
			dt.Precision = Minute
		// NOTE: Skip hour precision since FHIR specification disallows it
		case d != "":
			dt.Precision = Day
		case mo != "":
			dt.Precision = Month
		case y != "":
			dt.Precision = Year
		default:
			dt.Precision = Millisecond
		}

		// Get the location (if no time components or no location, use local)
		loc := time.Local
		if h != "" {
			if tzZu == "Z" {
				loc, _ = time.LoadLocation("UTC")
			} else if tzOp != "" && tzh != "" && tzm != "" {
				tzhi, _ := strconv.Atoi(tzh)
				tzmi, _ := strconv.Atoi(tzm)
				offset := tzhi*60*60 + tzmi*60
				if tzOp == "-" {
					offset *= -1
				}
				loc = time.FixedZone(tzOp+tzh+tzm, offset)
			}
		}

		// Convert to a time.Time
		yInt, _ := strconv.Atoi(y)
		moInt, err := strconv.Atoi(mo)
		if err != nil {
			moInt = 1
		}
		dInt, err := strconv.Atoi(d)
		if err != nil {
			dInt = 1
		}
		hInt, _ := strconv.Atoi(h)
		miInt, _ := strconv.Atoi(mi)
		sInt, _ := strconv.Atoi(s)
		msInt, _ := strconv.Atoi(ms)

		dt.Value = time.Date(yInt, time.Month(moInt), dInt, hInt, miInt, sInt, msInt*1000*1000, loc)
		return dt, nil
	} else {
		return nil, fmt.Errorf("could not parse date/time: %s", dateStr)
	}
}

// DatePrecision is an enum representing the precision of a date.
type DatePrecision int

// Constant values for the DatePrecision enum.
const (
	Year DatePrecision = iota
	Month
	Day
	Minute
	Second
	Millisecond
)

func (p DatePrecision) layout() string {
	switch p {
	case Year:
		return "2006"
	case Month:
		return "2006-01"
	case Day:
		return "2006-01-02"
	case Minute:
		return "2006-01-02T15:04-07:00"
	case Second:
		return "2006-01-02T15:04:05-07:00"
	case Millisecond:
		return "2006-01-02T15:04:05.000-07:00"
	default:
		return "2006-01-02T15:04:05.000-07:00"
	}
}