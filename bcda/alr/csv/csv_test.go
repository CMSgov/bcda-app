package csv_test

import (
	"errors"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/alr/csv"
	"github.com/stretchr/testify/assert"
)

func TestToALR(t *testing.T) {
	alrs, err := csv.ToALR("testdata/table1.csv", "testdata/table2.csv")
	assert.NoError(t, err)
	assert.Len(t, alrs, 5)
	// Spot check a couple of fields to verify merged data sets match our expectations
	for _, alr := range alrs {
		assert.False(t, alr.BeneDOB.IsZero(), "DOB should always be set")
		// Field that's not mapped explicitly and is present in table1.csv
		assert.Contains(t, alr.KeyValue, "HCC_COL_45")
		// Field that's not mapped explicitly and is present in table2.csv
		assert.Contains(t, alr.KeyValue, "MASTER_ID")
		// Verify entries that do not satisfy the join are not included
		assert.NotRegexp(t, regexp.MustCompile("NOT_FOUND_(1|2)"), alr.BeneMBI)
	}
}

func TestToALRBadInputs(t *testing.T) {
	tests := []struct {
		file  string
		cause string
	}{
		{"missing_mbi", "required filed 'BENE_MBI_ID' not found"},
		{"empty", "load records: empty DataFrame"},
		{"wrong_num_fields", "record on line 2: wrong number of fields"},
		{"not_found", "failed to open ALR file: open testdata/bad/not_found.csv: no such file or directory"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			alrs, err := csv.ToALR(filepath.Join("testdata", "bad", tt.file+".csv"))
			assert.Nil(t, alrs)
			assert.EqualError(t, errors.Unwrap(err), tt.cause)
		})
	}
}

// TestToALRBadDates verifies that we can still parse an ALR file even if the expected dates do not match our expectations
func TestToALRBadDates(t *testing.T) {
	alrs, err := csv.ToALR("testdata/bad_dates.csv")
	assert.NoError(t, err)
	assert.Len(t, alrs, 3)
}
