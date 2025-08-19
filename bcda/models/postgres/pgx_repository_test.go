package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPgxRepository_CCLFFileOperations(t *testing.T) {

	repo := NewPgxRepository()
	assert.NotNil(t, repo)

	assert.NotNil(t, repo)
}

func TestPgxRepository_NewPgxRepository(t *testing.T) {
	repo := NewPgxRepository()
	require.NotNil(t, repo)

	_, ok := interface{}(repo).(*PgxRepository)
	assert.True(t, ok, "NewPgxRepository should return *PgxRepository")
}
