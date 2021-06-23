package alr_test

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	alrcsv "github.com/CMSgov/bcda-app/bcda/alr/csv"
	alrgen "github.com/CMSgov/bcda-app/bcda/alr/gen"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/CMSgov/bcda-app/bcda/utils"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/v1"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr/v2"
	"github.com/google/fhir/go/jsonformat"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

var output = flag.Bool("output", false, "write FHIR resources to a file")
var version  = flag.Int("version", 1, "write FHIR resources to a file")
var resources = [...]string{"patient", "coverage", "group", "risk", "observations"}

// TestGenerateAlr uses our synthetic data generation tool to produce the associated FHIR resources
// To write to the FHIR resources to a file:
// go test -v github.com/CMSgov/bcda-app/bcda/models/fhir/alr -run TestGenerateAlr -output true -version 1
func TestGenerateAlr(t *testing.T) {

    if *version > 2 {
        assert.Fail(t, "Version endpoint entered is not supported")
    }
    
	p, c := testUtils.CopyToTemporaryDirectory(t, "../../../alr/gen/testdata/")
	t.Cleanup(c)
	csvPath := filepath.Join(p, "PY21ALRTemplatePrelimProspTable1.csv")
	err := alrgen.UpdateCSV(csvPath, mbiSupplier{func() string { return testUtils.RandomMBI(t) }}.GetMBIs)
	assert.NoError(t, err)

	alrs, err := alrcsv.ToALR(csvPath)
	assert.NoError(t, err)
	assert.Len(t, alrs, 1)

	//lastUpdated := time.Now().Round(time.Second)
	fhirBulk1 := alr.ToFHIR(alrs[0], "fhir/v1")
	assert.NotNil(t, fhirBulk1.AlrBulkV1)

	fhirBulk2 := alr.ToFHIR(alrs[0], "fhir/v2")
	assert.NotNil(t, fhirBulk2.AlrBulkV2)

	// Do not write the FHIR resources to a file
	if !*output {
		//Test models.Alr where hccVersion is empty
		delete(alrs[0].KeyValue, "HCC_version")
		fhirBulk1 = alr.ToFHIR(alrs[0], "fhir/v1")
		fhirBulk2 = alr.ToFHIR(alrs[0], "fhir/v2")
		assert.Nil(t, fhirBulk1)
		assert.Nil(t, fhirBulk2)
		return
	}

    if *version == 1 {
        dir := writeToFileV1(t, fhirBulk1.AlrBulkV1)
        t.Logf("FHIR resources written to: %s", dir)
    }

    dir := writeToFileV2(t, fhirBulk2.AlrBulkV2)
    t.Logf("FHIR resources written to: %s", dir)

}

// writeToFile writes the patient and observation resources to a file returning the directory
func writeToFileV1(t *testing.T, fhirBulk *v1.AlrBulkV1) string {
	tempDir, err := ioutil.TempDir("", "alr_fhir")
	assert.NoError(t, err)

	marshaller, err := jsonformat.NewPrettyMarshaller(jsonformat.STU3)
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

	//PATIENT
	patientb, err := marshaller.MarshalResource(fhirBulk.Patient)
	assert.NoError(t, err)
	patients := string(patientb) + "\n"

	// COVERAGE
	coverageb, err := marshaller.MarshalResource(fhirBulk.Coverage)
	assert.NoError(t, err)
	coverage := string(coverageb) + "\n"

	// GROUP
	groupb, err := marshaller.MarshalResource(fhirBulk.Group)
	assert.NoError(t, err)
	group := string(groupb) + "\n"

	// RISK
	var riskAssessment = []string{}

	for _, r := range fhirBulk.Risk {

		riskb, err := marshaller.MarshalResource(r)
		assert.NoError(t, err)
		risk := string(riskb) + "\n"
		riskAssessment = append(riskAssessment, risk)
	}
	risk := strings.Join(riskAssessment, "\n")

	// OBSERVATION
	observationb, err := marshaller.MarshalResource(fhirBulk.Observation)
	assert.NoError(t, err)

	observation := string(observationb) + "\n"

	alrResources := []string{patients, observation, coverage, group, risk}

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

	marshaller, err := jsonformat.NewPrettyMarshaller(jsonformat.R4)
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

	//PATIENT
	patientb, err := marshaller.MarshalResource(fhirBulk.Patient)
	assert.NoError(t, err)
	patients := string(patientb) + "\n"

	// COVERAGE
	coverageb, err := marshaller.MarshalResource(fhirBulk.Coverage)
	assert.NoError(t, err)
	coverage := string(coverageb) + "\n"

	// GROUP
	groupb, err := marshaller.MarshalResource(fhirBulk.Group)
	assert.NoError(t, err)
	group := string(groupb) + "\n"

	// RISK
	var riskAssessment = []string{}

	for _, r := range fhirBulk.Risk {

		riskb, err := marshaller.MarshalResource(r)
		assert.NoError(t, err)
		risk := string(riskb) + "\n"
		riskAssessment = append(riskAssessment, risk)
	}
	risk := strings.Join(riskAssessment, "\n")

	// OBSERVATION
	observationb, err := marshaller.MarshalResource(fhirBulk.Observation)
	assert.NoError(t, err)

	observation := string(observationb) + "\n"

	alrResources := []string{patients, observation, coverage, group, risk}

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
