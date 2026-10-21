package bcdaaws

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeMountPath(t *testing.T) {
	t.Run("returns 0 when mountPath is empty", func(t *testing.T) {
		count, err := ProbeMountPath("")
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("returns error when directory does not exist", func(t *testing.T) {
		count, err := ProbeMountPath("/nonexistent/path/for/test")
		require.Error(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("succeeds with empty directory", func(t *testing.T) {
		dir := t.TempDir()
		count, err := ProbeMountPath(dir)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("reads up to 10 entries when directory has more than 10 items", func(t *testing.T) {
		dir := t.TempDir()
		for i := 0; i < 15; i++ {
			err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file_%d.txt", i)), []byte("test"), 0600)
			require.NoError(t, err)
		}

		count, err := ProbeMountPath(dir)
		require.NoError(t, err)
		assert.Equal(t, 10, count)
	})

	t.Run("reads exact count when directory has fewer than 10 items", func(t *testing.T) {
		dir := t.TempDir()
		for i := 0; i < 3; i++ {
			err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file_%d.txt", i)), []byte("test"), 0600)
			require.NoError(t, err)
		}

		count, err := ProbeMountPath(dir)
		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}
