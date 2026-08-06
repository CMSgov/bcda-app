package service

import (
	"fmt"
	"os"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models/postgres"
	"github.com/stretchr/testify/assert"
)

func TestSetupWarningsAndInfoFile(t *testing.T) {
	pool := database.ConnectPool()
	repo := postgres.NewPgxRepositoryWithPool(pool)
	defer pool.Close()

	tmp := os.TempDir()
	origDir := os.Getenv("FHIR_PAYLOAD_DIR")
	err := os.Setenv("FHIR_PAYLOAD_DIR", tmp)
	assert.NoError(t, err)
	defer func() {
		err = os.Setenv("FHIR_PAYLOAD_DIR", origDir)
		assert.NoError(t, err)
	}()

	err = SetupWarningsAndInfoFile(t.Context(), repo, uint(33))
	assert.NoError(t, err)

	payloadPath := fmt.Sprintf("%s/%d", os.Getenv("FHIR_PAYLOAD_DIR"), 33)
	assert.DirExists(t, payloadPath)

	var id int
	err = pool.QueryRow(t.Context(), "SELECT id FROM job_keys WHERE job_id = 33 AND file_name = $1", constants.WarningsAndInfoFileName).Scan(&id)
	assert.NoError(t, err)
	assert.NotZero(t, id)
}
