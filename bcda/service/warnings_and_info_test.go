package service

import (
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

	filePath, err := SetupWarningsAndInfoFile(t.Context(), repo, uint(1))
	assert.NoError(t, err)

	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	var id int
	err = pool.QueryRow(t.Context(), "SELECT * FROM job_keys WHERE job_id = 1 AND file_name = $1", constants.WarningsAndInfoFileName).Scan(&id)
	assert.NoError(t, err)
	assert.NotZero(t, id)

	err = os.Remove(filePath)
	assert.NoError(t, err)
}
