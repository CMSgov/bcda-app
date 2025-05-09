package models

import "testing"

// SetMockRepository sets the current repository to the one that's supplied in this function.
// It leverages the Cleanup() func to ensure the original repository is restored at the end of the test.
func SetMockRepository(t *testing.T, other *MockRepository) {
	// Ensure that we restore the original provider when the test completes
	originalRepository := repository
	t.Cleanup(func() {
		repository = originalRepository
	})
	repository = other
}
