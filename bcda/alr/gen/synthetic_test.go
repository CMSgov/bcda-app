package gen

import (
	"crypto/rand"
	"encoding/csv"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/testUtils"
	"github.com/stretchr/testify/assert"
)

func TestUpdateCSV(t *testing.T) {

	isTesting = true

	mbiCount, e := rand.Int(rand.Reader, big.NewInt(1000))
	if e != nil {
		t.Fatalf("failed to generate random number: %v", e)
	}
	path, cleanup := testUtils.CopyToTemporaryDirectory(t, "testdata")
	defer cleanup()

	csvPath := filepath.Join(path, "PY21ALRTemplatePrelimProspTable1.csv")
	err := UpdateCSV(csvPath, randomMBIGenerator{t, int(mbiCount.Int64())}.GetMBIs)
	assert.NoError(t, err)

	file, err := os.Open(csvPath)
	assert.NoError(t, err)
	defer file.Close()

	records, err := csv.NewReader(file).ReadAll()
	assert.NoError(t, err)

	// One more record to account for header
	assert.Equal(t, int(mbiCount.Int64())+1, len(records))
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
