package alr_test

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	alrcsv "github.com/CMSgov/bcda-app/bcda/alr/csv"
	alrgen "github.com/CMSgov/bcda-app/bcda/alr/gen"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	v1 "github.com/CMSgov/bcda-app/bcda/models/fhir/alr/v1"
	v2 "github.com/CMSgov/bcda-app/bcda/models/fhir/alr/v2"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var output *bool = flag.Bool("output", false, "write FHIR resources to a file")
var version *int = flag.Int("version", 1, "version of FHIR resources")
var resources = [...]string{"patient", "coverage", "group", "risk", "observations"}

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
	err := alrgen.UpdateCSV(csvPath, mbiSupplier{func() string { return testUtils.RandomMBI(t) }}.GetMBIs)
	assert.NoError(t, err)

	alrs, err := alrcsv.ToALR(csvPath)
	assert.NoError(t, err)
	assert.Len(t, alrs, 1)

	// FN parameter version comes from Jobalrenqueue, here we are setting it manually for testing
	// Timestamp comes from alrRequest fro api package, but manually set here
	alrs[0].Timestamp = time.Now()
	fhirBulk1 := alr.ToFHIR(alrs[0], "/v1/fhir")
	assert.NotNil(t, fhirBulk1.AlrBulkV1)

	fhirBulk2 := alr.ToFHIR(alrs[0], "/v2/fhir")
	assert.NotNil(t, fhirBulk2.AlrBulkV2)

	missing := alr.ToFHIR(alrs[0], "fhir/Not Supported")
	assert.Nil(t, missing)

	// Do not write the FHIR resources to a file
	if !*output {
		//Test models.Alr where hccVersion is empty
		delete(alrs[0].KeyValue, "HCC_version")
		fhirBulk1 = alr.ToFHIR(alrs[0], "/v1/fhir")
		fhirBulk2 = alr.ToFHIR(alrs[0], "/v2/fhir")
		assert.Nil(t, fhirBulk1.AlrBulkV1)
		assert.Nil(t, fhirBulk2.AlrBulkV2)
		return
	}

	if *version == 1 {
		dir := writeToFileV1(t, fhirBulk1.AlrBulkV1)
		t.Logf("FHIR STU3 resources written to: %s", dir)
		return
	}

	dir := writeToFileV2(t, fhirBulk2.AlrBulkV2)
	t.Logf("FHIR R4 resources written to: %s", dir)

}

// writeToFile writes the FHIR resources to a file returning the directory
func writeToFileV1(t *testing.T, fhirBulk *v1.AlrBulkV1) string {
	tempDir, err := ioutil.TempDir("", "alr_fhir")
	assert.NoError(t, err)

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
	alrResources, err := fhirBulk.FhirToString()
	assert.NoError(t, err)

	// IO operations
	for n, resource := range alrResources {

		w := writerPool[n]

		_, err = w.WriteString(resource)
		assert.NoError(t, err)
		err = w.Flush()
		assert.NoError(t, err)

	}

	return tempDir
}

func writeToFileV2(t *testing.T, fhirBulk *v2.AlrBulkV2) string {
	tempDir, err := ioutil.TempDir("", "alr_fhir")
	assert.NoError(t, err)

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

	alrResources, err := fhirBulk.FhirToString()
	assert.NoError(t, err)

	// IO operations
	for n, resource := range alrResources {

		w := writerPool[n]

		_, err = w.WriteString(resource)
		assert.NoError(t, err)
		err = w.Flush()
		assert.NoError(t, err)

	}

	return tempDir
}

type mbiSupplier struct {
	mbiGen func() string
}

func (m mbiSupplier) GetMBIs() ([]string, error) {
	return []string{m.mbiGen()}, nil
}
