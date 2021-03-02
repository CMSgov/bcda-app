package gen

import (
	"encoding/csv"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCSV(t *testing.T) {
	mbiCount := rand.Intn(1000)
	path, cleanup := testUtils.CopyToTemporaryDirectory(t, "testdata")
	defer cleanup()

	csvPath := filepath.Join(path, "PY21ALRTemplatePrelimProspTable1.csv")
	err := UpdateCSV(csvPath, randomMBIGenerator{t, mbiCount})
	assert.NoError(t, err)

	file, err := os.Open(csvPath)
	assert.NoError(t, err)
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	assert.NoError(t, err)

	// One more record to account for header
	assert.Equal(t, mbiCount+1, len(records))
}

type randomMBIGenerator struct {
	*testing.T
	count int
}

func (gen randomMBIGenerator) GetMBIs() ([]string, error) {
	mbis := make([]string, gen.count)
	for i := 0; i < gen.count; i++ {
		mbis[i] = testUtils.RandomMBI(gen.T)
	}
	return mbis, nil
}
