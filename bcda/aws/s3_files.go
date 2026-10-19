package bcdaaws

import (
	"errors"
	"io"
	"os"
)

// ProbeMountPath checks if mountPath exists and returns directory content (currently limited to 10)
// If mountPath is empty, it returns (0, nil).
func ProbeMountPath(mountPath string) (int, error) {
	if mountPath == "" {
		return 0, nil
	}

	f, err := os.Open(mountPath)
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
