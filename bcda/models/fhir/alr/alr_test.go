package alr_test

import (
	"flag"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
	"time"

	alrcsv "github.com/CMSgov/bcda-app/bcda/alr/csv"
	alrgen "github.com/CMSgov/bcda-app/bcda/alr/gen"
	"github.com/CMSgov/bcda-app/bcda/models/fhir/alr"
	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/google/fhir/go/jsonformat"
	fhirmodels "github.com/google/fhir/go/proto/google/fhir/proto/stu3/resources_go_proto"
	"github.com/stretchr/testify/assert"
)

var output = flag.Bool("output", false, "write FHIR resources to a file")

// TestGenerateAlr uses our synthetic data generation tool to produce the associated FHIR resources
// To write to the FHIR resources to a file:
// go test -v github.com/CMSgov/bcda-app/bcda/models/fhir/alr -run TestGenerateAlr -output true
func TestGenerateAlr(t *testing.T) {
	p, c := testUtils.CopyToTemporaryDirectory(t, "../../../alr/gen/testdata/")
	t.Cleanup(c)
	csvPath := filepath.Join(p, "PY21ALRTemplatePrelimProspTable1.csv")
	err := alrgen.UpdateCSV(csvPath, mbiSupplier{func() string { return testUtils.RandomMBI(t) }}.GetMBIs)
	assert.NoError(t, err)

	alrs, err := alrcsv.ToALR(csvPath)
	assert.NoError(t, err)
	assert.Len(t, alrs, 1)

	lastUpdated := time.Now().Round(time.Second)
	patient, obs := alr.ToFHIR(alrs[0], lastUpdated)
	assert.NotNil(t, patient)
	assert.Len(t, obs, 5)

	// Do not write the FHIR resources to a file
	if !*output {
		return
	}

	dir := writeToFile(t, patient, obs)
	t.Logf("FHIR resources written to: %s", dir)
}

// writeToFile writes the patient and observation resources to a file returning the directory
func writeToFile(t *testing.T, patient *fhirmodels.Patient, observations []*fhirmodels.Observation) string {
	tempDir, err := ioutil.TempDir("", "alr_fhir")
	assert.NoError(t, err)

	marshaller, err := jsonformat.NewPrettyMarshaller(jsonformat.STU3)
	assert.NoError(t, err)

	pFile, err := ioutil.TempFile(tempDir, "patient")
	assert.NoError(t, err)
	defer pFile.Close()
	writeResource(t, pFile, marshaller, &fhirmodels.ContainedResource{
		OneofResource: &fhirmodels.ContainedResource_Patient{Patient: patient}})

	for _, observation := range observations {
		func() {
			// File name contains the specific observation details
			name := strings.ReplaceAll(observation.Code.Coding[1].Code.Value, " ", "_")
			file, err := ioutil.TempFile(tempDir, name)
			assert.NoError(t, err)
			defer file.Close()
			resource := &fhirmodels.ContainedResource{
				OneofResource: &fhirmodels.ContainedResource_Observation{Observation: observation}}
			assert.NoError(t, err)
			writeResource(t, file, marshaller, resource)
		}()
	}

	return tempDir
}

func writeResource(t *testing.T, w io.Writer, marshaller *jsonformat.Marshaller, resource *fhirmodels.ContainedResource) {
	data, err := marshaller.Marshal(resource)
	assert.NoError(t, err)
	_, err = w.Write(data)
	assert.NoError(t, err)
}

type mbiSupplier struct {
	mbiGen func() string
}

func (m mbiSupplier) GetMBIs() ([]string, error) {
	return []string{m.mbiGen()}, nil
}
