package alr

import (
	"fmt"
	"os"
	"path/filepath"

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

// Populates the crosswalk with
func init() {
	const (
		version        = "HCC Version"
		columnPosition = "HCC Column Position"
		flag           = "HCC Flag"
		description    = "HCC Description"
	)
	// Scan through local path then the location of the file from the RPM
	// See: ./ops/build_and_package.sh#49
	paths := []string{"./hcc_crosswalk.tsv",
		"/etc/sv/api/hcc_crosswalk.tsv",
		"/go/src/github.com/CMSgov/bcda-app/bcda/models/fhir/alr/hcc_crosswalk.tsv"}

	var df dataframe.DataFrame
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			f, err := os.Open(filepath.Clean(path))
			if err != nil {
				panic(err)
			}
			// See: https://github.com/securego/gosec/issues/579
			defer f.Close() // #nosec G307

			df = dataframe.ReadCSV(f, dataframe.HasHeader(true), dataframe.DetectTypes(false),
				dataframe.WithDelimiter('\t'))
			if df.Err != nil {
				logrus.Warnf("Failed to parse CSV to data frame. Skipping. Err: %s", df.Err)
				continue
			}

			logrus.Debugf("Successfully loaded dataframe from %s.", path)
			break
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
		key := hccKey{version: record[version].(string), columnPosition: record[columnPosition].(string)}
		value := hcc{flag: record[flag].(string), description: record[description].(string)}
		crosswalk[key] = value
	}
}

func hccData(version, column string) *hcc {
	res, ok := crosswalk[hccKey{version: version, columnPosition: column}]
	if !ok {
		return nil
	}
	return &res
}
