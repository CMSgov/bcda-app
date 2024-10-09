package alr_test

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	alrcsv "github.com/CMSgov/bcda-app/bcda/alr/csv"
	alrgen "github.com/CMSgov/bcda-app/bcda/alr/gen"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	workerutils "github.com/CMSgov/bcda-app/bcdaworker/worker/utils"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var output *bool = flag.Bool("output", false, "write FHIR resources to a file")
var version *int = flag.Int("version", 1, "version of FHIR resources")

const SIZE = 200

// TestGenerateAlr uses our synthetic data generation tool to produce the associated FHIR resources
// To write to the FHIR resources to a file:
// go test -v github.com/CMSgov/bcda-app/bcda/models/fhir/alr -run TestGenerateAlr -output -version 1
func TestGenerateAlr(t *testing.T) {

	// Get the flags if any, as mentioned in line 31
	flag.Parse()

	// Ensure the version is only 1 or 2
	if *version > 2 {
		panic(fmt.Sprintf("The endpoint version %d you provided is not supported.", *version))
		// We panic here because, any version value of 3 or greater doesn't exist.
		// We could have a default value and not panic if we like in the future.
	}

	p, c := testUtils.CopyToTemporaryDirectory(t, "../../../alr/gen/testdata/")
	t.Cleanup(c)
	csvPath := filepath.Join(p, "PY21ALRTemplatePrelimProspTable1.csv")
	err := alrgen.UpdateCSV(csvPath, mbiGenerator)
	assert.NoError(t, err)

	alrs, err := alrcsv.ToALR(csvPath)
	assert.NoError(t, err)
	assert.Len(t, alrs, SIZE)

	// FN parameter version comes from Jobalrenqueue, here we are setting it manually for testing
	// Timestamp comes from alrRequest fro api package, but manually set here
	alrs[0].Timestamp = time.Now()

	missing := alr.ToFHIR(alrs, "fhir/Not Supported")
	assert.Nil(t, missing)

	// Do not write the FHIR resources to a file
	if !*output {
		//Test models.Alr where hccVersion is empty
		delete(alrs[0].KeyValue, "HCC_version")
		fhirBulk1 := alr.ToFHIR(alrs, "/v1/fhir")
		fhirBulk2 := alr.ToFHIR(alrs, "/v2/fhir")
		assert.Nil(t, fhirBulk1.V1)
		assert.Nil(t, fhirBulk2.V2)
		return
	}

	if *version == 1 {
		ch := make(chan *alr.AlrFhirBulk, 1000) // 1000 rows before blocking
		workerutils.AlrSlicer(alrs, ch, 100, "/v1/fhir")
		dir := writeToFileV1(t, ch)
		t.Logf("FHIR STU3 resources written to: %s", dir)
		return
	}

	ch := make(chan *alr.AlrFhirBulk, 1000) // 1000 rows before blocking
	workerutils.AlrSlicer(alrs, ch, 100, "/v2/fhir")
	dir := writeToFileV2(t, ch)
	t.Logf("FHIR R4 resources written to: %s", dir)

}

// writeToFile writes the FHIR resources to a file returning the directory
func writeToFileV1(t *testing.T, fhirBulk chan *alr.AlrFhirBulk) string {
	assert.NotNil(t, fhirBulk)

	tempDir, err := os.MkdirTemp("", "alr_fhir")
	assert.NoError(t, err)

	var resources = [...]string{"patient", "coverage", "group", "risk", "observations", "covidEpisode"}
	fieldNum := len(resources)
	writerPool := make([]*bufio.Writer, fieldNum)

	for i := 0; i < fieldNum; i++ {
		ndjsonFilename := uuid.New()
		f, err := os.Create(fmt.Sprintf("%s/%s.ndjson", tempDir, ndjsonFilename))
		assert.NoError(t, err)

		file := f
		w := bufio.NewWriter(file)
		writerPool[i] = w
		defer utils.CloseFileAndLogError(file)
	}

	for j := range fhirBulk {

		for _, i := range j.V1 {

			// marshalling structs into JSON
			alrResources, err := i.FhirToString()
			assert.NoError(t, err)

			// IO operations
			for n, resource := range alrResources {

				w := writerPool[n]

				_, err = w.WriteString(resource)
				assert.NoError(t, err)
				err = w.Flush()
				assert.NoError(t, err)

			}
		}

	}

	return tempDir
}

func writeToFileV2(t *testing.T, fhirBulk chan *alr.AlrFhirBulk) string {

	assert.NotNil(t, fhirBulk)

	tempDir, err := os.MkdirTemp("", "alr_fhir")
	assert.NoError(t, err)

	var resources = [...]string{"patient", "coverage", "group", "risk", "observations", "covidEpisode"}
	fieldNum := len(resources)
	writerPool := make([]*bufio.Writer, fieldNum)

	for i := 0; i < fieldNum; i++ {
		ndjsonFilename := uuid.New()
		f, err := os.Create(fmt.Sprintf("%s/%s.ndjson", tempDir, ndjsonFilename))
		assert.NoError(t, err)

		file := f
		w := bufio.NewWriter(file)
		writerPool[i] = w
		defer utils.CloseFileAndLogError(file)
	}

	// marshalling structs into JSON

	for j := range fhirBulk {
		for _, i := range j.V2 {
			alrResources, err := i.FhirToString()
			assert.NoError(t, err)

			// IO operations
			for n, resource := range alrResources {

				w := writerPool[n]

				_, err = w.WriteString(resource)
				assert.NoError(t, err)
				err = w.Flush()
				assert.NoError(t, err)

			}
		}
	}

	return tempDir
}

func mbiGenerator() ([]string, error) {
	var s []string
	for i := 0; i < SIZE; i++ {
		s = append(s, testUtils.RandomMBI(&testing.T{}))
	}
	return s, nil
}
