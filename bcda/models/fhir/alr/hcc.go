package alr

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/go-gota/gota/dataframe"
	"github.com/sirupsen/logrus"
)

type hcc struct {
	flag        string
	description string
}

type hccKey struct {
	version        string
	columnPosition string
}

var crosswalk map[hccKey]hcc

func init() {
	const (
		version = "HCC Version"
		columnPosition = "HCC Column Position"
		flag = "HCC Flag"
		description = "HCC Description"
	)
	// Scan through local path then the location of the file from the RPM
	// See:
	paths := []string{"./hcc_crosswalk.csv", "/etc/sv/api/hcc_crosswalk.csv"}

	var df dataframe.DataFrame
	var r *csv.Reader
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			f, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			df = dataframe.ReadCSV(f)
		} else {
			logrus.Warnf("Failed to read file at %s. Skipping. Err: %s",
				path, err.Error())
		}
	}

	if df.Nrow() == 0 {
		panic(fmt.Sprintf("No crosswalk file found. Tried %v.", paths))
	}

	crosswalk = make(map[hccKey]hcc)
	for _, record := range df.Maps() {
		key := hccKey{version: record[version], }
	}
}
