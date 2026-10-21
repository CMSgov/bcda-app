package bcdaaws

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// ProbeMountPath checks if mountPath exists and returns directory content (currently limited to 10)
// If mountPath is empty, it returns (0, nil).
func ProbeMountPath(mountPath string) (int, error) {
	if mountPath == "" {
		return 0, nil
	}

	cleanPath := filepath.Clean(mountPath)
	f, err := os.Open(cleanPath) // #nosec G304 -- mountPath is an supplied env var, not user input
	if err != nil {
		return 0, err
	}
	defer f.Close()

	entries, err := f.ReadDir(10)
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, err
	}

	return len(entries), nil
}
