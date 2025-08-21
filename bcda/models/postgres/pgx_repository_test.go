package postgres

import (
	"testing"

	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgxRepository_CCLFFileOperations(t *testing.T) {
	pool := database.ConnectPool()
	defer pool.Close()

	repo := NewPgxRepositoryWithPool(pool)
	assert.NotNil(t, repo)

	assert.NotNil(t, repo)
}

func TestPgxRepository_NewPgxRepositoryWithPool(t *testing.T) {
	pool := database.ConnectPool()
	defer pool.Close()

	repo := NewPgxRepositoryWithPool(pool)
	require.NotNil(t, repo)

	_, ok := interface{}(repo).(*PgxRepository)
	assert.True(t, ok, "NewPgxRepositoryWithPool should return *PgxRepository")
}
