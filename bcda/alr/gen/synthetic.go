package gen

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"time"

	randomdata "github.com/Pallinder/go-randomdata"
	"github.com/sirupsen/logrus"
)

type mbiSupplier interface {
	GetMBIs() ([]string, error)
}

var (
	minBirthDate = time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)
	maxBirthDate = time.Date(1950, time.December, 31, 0, 0, 0, 0, time.UTC)

	minDeathDate = time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC)
)

// Links header fields to a generator that produces a string value.
// The generators used are based on the 2021 ALR data dictionary.
// NOTE: We currently only have definitions for ALR Table 1-1.
var valuegen map[*regexp.Regexp]func() string = map[*regexp.Regexp]func() string{
	regexp.MustCompile("HIC_NUM"):   func() string { return randomdata.Alphanumeric(12) },
	regexp.MustCompile("1ST_NAME"):  func() string { return randomdata.FirstName(randomdata.RandomGender) },
	regexp.MustCompile("LAST_NAME"): func() string { return randomdata.LastName() },
	regexp.MustCompile("SEX"):       func() string { return strconv.Itoa(randomdata.Number(3)) },
	regexp.MustCompile("BRTH_DT"):   func() string { return randomDate(minBirthDate, maxBirthDate) },
	regexp.MustCompile("DEATH_DT"): func() string {
		return randomEmpty(less,
			func() string { return randomDate(minDeathDate, time.Now()) })
	},
	regexp.MustCompile("CNTY"):  func() string { return randomdata.City() }, // No county data source
	regexp.MustCompile("STATE"): func() string { return randomdata.State(randomdata.Large) },
	// NOTE: This CNTY_CODE will not match up with the provided CNTY/STATE tuple.
	// Need to integrate valid counties + state codes using the FIPS data set to align
	// See: https://en.wikipedia.org/wiki/FIPS_county_code
	regexp.MustCompile("CNTY_CODE"): func() string { return strconv.Itoa(randomdata.Number(1, 52)) + "000" },
	regexp.MustCompile("VA_TIN"):    func() string { return randomdata.StringNumberExt(1, "", 9) },
	regexp.MustCompile("VA_NPI"):    func() string { return randomdata.StringNumberExt(1, "", 10) },
	regexp.MustCompile("EnrollFlag"): func() string {
		return randomEmpty(half,
			func() string { return strconv.Itoa(randomdata.Number(5)) })
	},
	regexp.MustCompile("HCC_version"): func() string { return randomdata.StringSample("V12", "V22") },
	regexp.MustCompile("HCC_COL"): func() string {
		return randomEmpty(quarter,
			func() string { return strconv.Itoa(randomdata.Number(2)) })
	},
	regexp.MustCompile("BENE_RSK"): func() string {
		res := randomdata.Decimal(1, 2, 3)
		return strconv.FormatFloat(res, 'f', 3, 64)
	},
	regexp.MustCompile("SCORE"): func() string {
		return randomEmpty(half, func() string {
			res := randomdata.Decimal(1, 2, 3)
			return strconv.FormatFloat(res, 'f', 3, 64)
		})
	},
}

// UpdateCSV uses a random generator to populate fields present in the CSV file referenced by the fileName.
// It will generate a new row for each MBI returned by the supplier.
func UpdateCSV(fileName string, supplier mbiSupplier) error {
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()
	r := csv.NewReader(file)
	w := csv.NewWriter(file)
	defer w.Flush()

	// Read in the headers
	headers, err := r.Read()
	if err != nil {
		return fmt.Errorf("failed to read in CSV header information: %w", err)
	}
	headers = headers[1:] // Remove MBI assumed to be first

	mbis, err := supplier.GetMBIs()
	if err != nil {
		return fmt.Errorf("failed to get MBIs %w", err)
	}
	for _, mbi := range mbis {
		data := make([]string, 0, len(headers))
		data = append(data, mbi)
		for _, header := range headers {
			var hasMatch bool
			for exp, generator := range valuegen {
				if exp.MatchString(header) {
					data = append(data, generator())
					hasMatch = true
					break
				}
			}

			if !hasMatch {
				logrus.Debugf("No regex match found for header %s. Defaulting to empty string",
					header)
				data = append(data, "")
			}
		}

		if err := w.Write(data); err != nil {
			return fmt.Errorf("failed to write CSV data: %w", err)
		}
	}

	return nil
}

type weight int

const (
	half    weight = 2
	quarter weight = 4
	less    weight = 10
)

var randomizer = rand.New(rand.NewSource(time.Now().UnixNano()))

// randomEmpty uses the weight value to check if we should return an empty string
func randomEmpty(w weight, supplier func() string) string {
	if randomizer.Int31()%int32(w) == 0 {
		return supplier()
	}
	return ""
}

func randomDate(min, max time.Time) string {
	const layout = "01/02/2006"
	d := randomdata.FullDateInRange(min.Format(randomdata.DateInputLayout),
		max.Format(randomdata.DateInputLayout))
	t, err := time.Parse(randomdata.DateOutputLayout, d)
	// Since we're using the same output format, this should never occur
	if err != nil {
		panic("Cannot parse to ALR time format " + err.Error())
	}

	return t.Format(layout)
}
