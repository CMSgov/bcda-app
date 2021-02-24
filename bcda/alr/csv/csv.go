package csv

import (
	"fmt"
	"os"

	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/dimchansky/utfbom"
	"github.com/go-gota/gota/dataframe"
)

const (
	mbi   = "BENE_MBI_ID"
	hic   = "BENE_HIC_NUM"
	first = "BENE_1ST_NAME"
	last  = "BENE_LAST_NAME"
	sex   = "BENE_SEX_CD"
	birth = "BENE_BRTH_DT"
	death = "BENE_DEATH_DT"
)

// Fields the must be present in an ALR file
var requiredFields []string = []string{
	mbi, hic,
	first, last,
	sex,
	birth, death,
}

// Fields that will be used to join multiple dataframes together
var joinFields = requiredFields

// ToALR reads in a CSV file(s) and unmarshals the data into an ALR model.
// CSV files are joined based on a predetermined list of fields
func ToALR(names ...string) ([]models.Alr, error) {
	var mergedDF dataframe.DataFrame
	for _, name := range names {
		df, err := toDataFrame(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create dataframe: %w", err)
		}
		if err := validate(df); err != nil {
			return nil, fmt.Errorf("dataframe from %s is not valid: %w",
				name, err)
		}

		if len(mergedDF.Names()) == 0 {
			mergedDF = df
		} else {
			mergedDF = mergedDF.InnerJoin(df, joinFields...)
		}

		fmt.Printf("%d %d %d %v\n", df.Ncol(), df.Ncol(), df.Nrow(), df.Names())
		fmt.Printf("%d %d %d %v\n", mergedDF.Ncol(), mergedDF.Ncol(), mergedDF.Nrow(), mergedDF.Names())
	}

	records := mergedDF.Records()
	return toALR(records[0], records[1:])
}

func toDataFrame(fileName string) (dataframe.DataFrame, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return dataframe.DataFrame{}, fmt.Errorf("failed to open ALR file: %w", err)
	}
	defer f.Close()

	// Trim the Byte Order Marker if it's present
	// See: https://github.com/golang/go/issues/33887
	reader := utfbom.SkipOnly(f)

	df := dataframe.ReadCSV(reader, dataframe.HasHeader(true), dataframe.DetectTypes(false))
	// Any error from this read operation is written to the Err field

	return df, df.Err
}

func validate(df dataframe.DataFrame) error {
	fields := df.Names()
	m := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		m[field] = struct{}{}
	}

	for _, required := range requiredFields {
		if _, ok := m[required]; !ok {
			return fmt.Errorf("required filed '%s' not found", required)
		}
	}

	return nil
}

func toALR(headers []string, rows [][]string) ([]models.Alr, error) {
	setters := getALRSetters(headers)
	alrs := make([]models.Alr, 0, len(rows))
	for _, row := range rows {
		a := &alr{}
		a.KeyValue = make(map[string]string, len(row))
		for idx, val := range row {
			setter := setters[idx]
			// No specific field set, we'll add this to the generic K:V
			if setter == nil {
				a.KeyValue[headers[idx]] = val
			} else {
				setter(a, val)
			}
		}
		alrs = append(alrs, a.Alr)
	}

	return alrs, nil
}

// Returns a map that links column position with the method that should be
// used to populate an ALR field
func getALRSetters(headers []string) map[int]func(*alr, string) {
	setters := make(map[int]func(*alr, string))
	for idx, header := range headers {
		switch header {
		case mbi:
			setters[idx] = func(a *alr, mbi string) { a.setMBI(mbi) }
		case hic:
			setters[idx] = func(a *alr, hic string) { a.setHIC(hic) }
		case first:
			setters[idx] = func(a *alr, firstName string) { a.setFirstName(firstName) }
		case last:
			setters[idx] = func(a *alr, lastName string) { a.setLastName(lastName) }
		case sex:
			setters[idx] = func(a *alr, sex string) { a.setSex(sex) }
		case birth:
			setters[idx] = func(a *alr, birthDate string) { a.setBirth(birthDate) }
		case death:
			setters[idx] = func(a *alr, deathDate string) { a.setDeath(deathDate) }
		}
	}

	return setters
}
